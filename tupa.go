package tupa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type (
	Context interface {
		Request() *http.Request
		Response() http.ResponseWriter
		SendString(s string) error
		Param(param string) string
		QueryParam(param string) string
		QueryParams() map[string][]string
		SetRequest(r *http.Request)
		SetResponse(w http.ResponseWriter)
		GetCtx() context.Context
		SetContext(ctx context.Context)
		Context() context.Context
		CtxWithValue(key, value interface{}) *TupaContext
		CtxValue(key interface{}) interface{}
		NewTupaContext(req **http.Request, resp http.ResponseWriter, ctx context.Context) *TupaContext
	}

	TupaContext struct {
		Req  *http.Request
		Resp http.ResponseWriter
		Ctx  context.Context
	}
)

type APIFunc func(*TupaContext) error

type MiddlewareFuncCtx func(APIFunc) *TupaContext

type APIServer struct {
	listenAddr             string
	server                 *http.Server
	globalMiddlewares      MiddlewareChain
	globalAfterMiddlewares MiddlewareChain
	router                 *Router
	routeManager           RouteManager
}

const (
	MethodGet     HTTPMethod = http.MethodGet
	MethodPost    HTTPMethod = http.MethodPost
	MethodPut     HTTPMethod = http.MethodPut
	MethodDelete  HTTPMethod = http.MethodDelete
	MethodPatch   HTTPMethod = http.MethodPatch
	MethodOptions HTTPMethod = http.MethodOptions
)

var AllowedMethods = map[HTTPMethod]bool{
	MethodGet:     true,
	MethodPost:    true,
	MethodPut:     true,
	MethodDelete:  true,
	MethodPatch:   true,
	MethodOptions: true,
}

var allRoutes []RouteInfo

func (a *APIServer) New() {
	if a.routeManager == nil {
		defaultRouteManager()
	} else {
		a.routeManager()
	}

	a.RegisterRoutes(GetRoutes())

	c := cors.New(cors.Options{
		AllowedHeaders: []string{
			"Authorization", "authorization", "Accept", "Content-Type", "X-Requested-With", "X-Frame-Options",
			"X-XSS-Protection", "X-Content-Type-Options", "X-Permitted-Cross-Domain-Policies", "Referrer-Policy", "Expect-CT",
			"Feature-Policy", "Content-Security-Policy", "Content-Security-Policy-Report-Only", "Strict-Transport-Security",
			"Public-Key-Pins", "Public-Key-Pins-Report-Only", "Access-Control-Allow-Origin", "Access-Control-Allow-Methods",
			"Access-Control-Allow-Headers", "Access-Control-Allow-Credentials", "X-Forwarded-For", "X-Real-IP",
			"X-Csrf-Token", "X-HTTP-Method-Override",
		},
		AllowCredentials: true,
	})
	serverHandler := c.Handler(a.router)

	a.server = &http.Server{
		Addr:    a.listenAddr,
		Handler: serverHandler,
	}

	fmt.Println(FmtBlue("Servidor iniciado na porta: " + a.listenAddr))

	go func() {
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(FmtRed("Erro ao iniciar servidor: "), err)
		}
		log.Println(FmtYellow("Servidor parou de receber novas conexões"))
	}()

	signchan := make(chan os.Signal, 1)
	signal.Notify(signchan, syscall.SIGINT, syscall.SIGTERM)
	<-signchan // vai esperar um comando que encerra o servidor

	ctx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := a.server.Shutdown(ctx); err != nil {
		log.Fatal(FmtRed("Erro ao desligar servidor: "), err)
	}

	fmt.Println(FmtYellow("Servidor encerrado na porta: " + a.listenAddr))
}

func NewAPIServer(listenAddr string, routeManager RouteManager) *APIServer {
	return &APIServer{
		listenAddr:             listenAddr,
		globalMiddlewares:      MiddlewareChain{},
		globalAfterMiddlewares: MiddlewareChain{},
		router:                 NewRouter(), // não está recebendo nenhum middleware por enquanto
		routeManager:           routeManager,
	}
}

func defaultRouteManager() {
	AddRoutes(nil, func() []RouteInfo {
		return []RouteInfo{
			{
				Path:   "/",
				Method: MethodGet,
				Handler: func(tc *TupaContext) error {
					WriteJSONHelper(tc.Resp, http.StatusOK, "Seja bem vindo ao Tupã framework!")
					return nil
				},
			},
		}
	})
}

func (a *APIServer) RegisterRoutes(routeInfos []RouteInfo) {
	for _, routeInfo := range routeInfos {
		if !AllowedMethods[routeInfo.Method] {
			log.Fatalf(fmt.Sprintf(FmtRed("Método HTTP não permitido: "), "%s\nVeja como criar um novo método na documentação", routeInfo.Method))
		}

		handler := a.MakeHTTPHandlerFuncHelper(routeInfo)
		// a.router.HandleFunc(routeInfo.Path, handler).Methods(string(routeInfo.Method))
		a.router.Handle(string(routeInfo.Method), routeInfo.Path, handler)
	}
}

func WriteJSONHelper(w http.ResponseWriter, status int, v any) error {
	if w == nil {
		return errors.New("Response writer passado está nulo")
	}

	w.Header().Set("Content-Type", "application/json")

	if w.Header().Get("Content-Type") == "" {
		w.WriteHeader(status)
	}

	return json.NewEncoder(w).Encode(v)
}

func (a *APIServer) MakeHTTPHandlerFuncHelper(routeInfo RouteInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &TupaContext{
			Req:  r,
			Resp: w,
			Ctx:  r.Context(),
		}

		// Combina middlewares globais com os especificos de rota
		allMiddlewares := MiddlewareChain{}
		allMiddlewares = append(allMiddlewares, a.globalMiddlewares...)
		allMiddlewares = append(allMiddlewares, routeInfo.Middlewares...)

		doneCh := a.executeMiddlewaresAsync(ctx, allMiddlewares)
		errorsSlice := <-doneCh // espera até que algum valor seja recebido. Continua no primeiro erro recebido ( se houver ) ou se não houver nenhum erro

		if len(errorsSlice) > 0 {
			err := errorsSlice[0]
			if apiErr, ok := err.(APIHandlerErr); ok {
				slog.Error("API Error", "err:", apiErr, "status:", apiErr.Status)
				WriteJSONHelper(w, apiErr.Status, APIError{Error: apiErr.Error()})
			} else {
				slog.Error("API Error", "err:", apiErr, "status:", apiErr.Status)
				WriteJSONHelper(w, http.StatusInternalServerError, APIError{Error: err.Error()})
			}
			return
		}

		if r.Method == string(routeInfo.Method) {
			// if err := routeInfo.Handler(ctx); err != nil {
			// 	if err := WriteJSONHelper(w, http.StatusInternalServerError, APIError{Error: err.Error()}); err != nil {
			// 		fmt.Println("Erro ao escrever resposta JSON:", err)
			// 	}
			// }
			err := routeInfo.Handler(ctx)
			if err != nil {
				if apiErr, ok := err.(APIHandlerErr); ok {
					slog.Error("API Error", "err:", apiErr, "status:", apiErr.Status)
					WriteJSONHelper(w, apiErr.Status, APIError{Error: apiErr.Error()})
				} else {
					WriteJSONHelper(w, http.StatusInternalServerError, APIError{Error: err.Error()})
				}
			}
		} else {
			WriteJSONHelper(w, http.StatusMethodNotAllowed, APIError{Error: "Método HTTP não permitido"})
		}

		allAfterMiddlewares := MiddlewareChain{}
		allAfterMiddlewares = append(allAfterMiddlewares, routeInfo.AfterMiddlewares...)
		allAfterMiddlewares = append(allAfterMiddlewares, a.globalAfterMiddlewares...)

		doneCh = a.executeMiddlewaresAsync(ctx, allAfterMiddlewares)
		errorsSlice = <-doneCh

		if len(errorsSlice) > 0 {
			err := errorsSlice[0]
			if apiErr, ok := err.(APIHandlerErr); ok {
				slog.Error("API Error", "err:", apiErr, "status:", apiErr.Status)
				WriteJSONHelper(w, apiErr.Status, APIError{Error: apiErr.Error()})
			} else {
				slog.Error("API Error", "err:", apiErr, "status:", apiErr.Status)
				WriteJSONHelper(w, http.StatusInternalServerError, APIError{Error: err.Error()})
			}
			return
		}
	}
}

func AddRoutes(groupMiddlewares MiddlewareChain, routeFuncs ...func() []RouteInfo) {
	for _, routeFunc := range routeFuncs {
		routes := routeFunc()
		for i := range routes {
			allMiddlewares := append(MiddlewareChain(nil), groupMiddlewares...)
			routes[i].Middlewares = append(allMiddlewares, routes[i].Middlewares...)
		}
		allRoutes = append(allRoutes, routes...)
	}
}

func GetRoutes() []RouteInfo {
	routesCopy := make([]RouteInfo, len(allRoutes))
	copy(routesCopy, allRoutes)
	return routesCopy
}

func (tc *TupaContext) Request() *http.Request {
	return tc.Req
}

func (tc *TupaContext) Response() http.ResponseWriter {
	return tc.Resp
}

func (tc *TupaContext) SendString(s string) error {
	_, err := tc.Resp.Write([]byte(s))
	return err
}

func (tc *TupaContext) SetRequest(r *http.Request) {
	tc.Req = r
}

func (tc *TupaContext) SetResponse(w http.ResponseWriter) {
	tc.Resp = w
}

func (tc *TupaContext) Param(param string) string {
	return mux.Vars(tc.Req)[param]
}

func (tc *TupaContext) QueryParam(param string) string {
	return tc.Request().URL.Query().Get(param)
}

func (tc *TupaContext) QueryParams() map[string][]string {
	return tc.Request().URL.Query()
}

func (tc *TupaContext) Params() map[string]string {
	return mux.Vars(tc.Request())
}

func (tc *TupaContext) GetCtx() context.Context {
	return tc.Ctx
}

func (tc *TupaContext) SetContext(ctx context.Context) {
	tc.Ctx = ctx
}

func (tc *TupaContext) GetReqCtx() context.Context {
	return tc.Req.Context()
}

func NewTupaContextWithContext(w http.ResponseWriter, r *http.Request, ctx context.Context) *TupaContext {
	return &TupaContext{
		Req:  r,
		Resp: w,
		Ctx:  ctx,
	}
}

func (tc *TupaContext) CtxWithValue(key, value interface{}) *TupaContext {
	newCtx := context.WithValue(tc.Ctx, key, value)
	return NewTupaContextWithContext(tc.Resp, tc.Req, newCtx)
}

func (tc *TupaContext) CtxValue(key interface{}) interface{} {
	return tc.Ctx.Value(key)
}

func NewTupaContext(req **http.Request, resp http.ResponseWriter, ctx context.Context) *TupaContext {
	return &TupaContext{
		Req:  *req,
		Resp: resp,
		Ctx:  ctx,
	}
}
