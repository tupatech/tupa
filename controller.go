package tupa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	}

	TupaContext struct {
		request  *http.Request
		response http.ResponseWriter
		context.Context
	}
)

type APIFunc func(*TupaContext) error

type APIServer struct {
	listenAddr        string
	server            *http.Server
	globalMiddlewares MiddlewareChain
	router            *mux.Router
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
	MethodGet:    true,
	MethodPost:   true,
	MethodPut:    true,
	MethodDelete: true,
	MethodPatch:  true,
}

var allRoutes []RouteInfo

func (a *APIServer) New() {
	if a.router.GetRoute("/") == nil {
		a.RegisterRoutes([]RouteInfo{
			{
				Path:    "/",
				Method:  MethodGet,
				Handler: WelcomeHandler,
			},
		})
	}

	routerHandler := cors.Default().Handler(a.router)

	a.server = &http.Server{
		Addr:    a.listenAddr,
		Handler: routerHandler,
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

func NewAPIServer(listenAddr string) *APIServer {
	router := mux.NewRouter()

	return &APIServer{
		listenAddr:        listenAddr,
		router:            router,
		globalMiddlewares: MiddlewareChain{},
	}
}

func WelcomeHandler(tc *TupaContext) error {
	WriteJSONHelper(tc.response, http.StatusOK, "Seja bem vindo ao Tupã framework!")
	return nil
}

func (a *APIServer) RegisterRoutes(routeInfos []RouteInfo) {
	for _, routeInfo := range routeInfos {
		if !AllowedMethods[routeInfo.Method] {
			log.Fatalf(fmt.Sprintf(FmtRed("Método HTTP não permitido: "), "%s\nVeja como criar um novo método na documentação", routeInfo.Method))
		}

		var allMiddlewares MiddlewareChain
		middlewaresGlobais := a.GetGlobalMiddlewares()

		allMiddlewares = append(allMiddlewares, middlewaresGlobais...)

		handler := a.MakeHTTPHandlerFuncHelper(routeInfo, allMiddlewares, a.globalMiddlewares)

		a.router.HandleFunc(routeInfo.Path, handler).Methods(string(routeInfo.Method))
	}
}

func WriteJSONHelper(w http.ResponseWriter, status int, v any) error {
	if w == nil {
		return errors.New("Response writer passado está nulo")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(v)
}

func (a *APIServer) MakeHTTPHandlerFuncHelper(routeInfo RouteInfo, middlewares MiddlewareChain, globalMiddlewares MiddlewareChain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &TupaContext{
			request:  r,
			response: w,
		}

		// Combina middlewares globais com os especificos de rota
		allMiddlewares := MiddlewareChain{}
		allMiddlewares = append(allMiddlewares, a.GetGlobalMiddlewares()...)
		allMiddlewares = append(allMiddlewares, routeInfo.Middlewares...)

		doneCh := a.executeMiddlewaresAsync(ctx, allMiddlewares)
		errorsSlice := <-doneCh // espera até que algum valor seja recebido. Continua no primeiro erro recebido ( se houver ) ou se não houver nenhum erro

		if len(errorsSlice) > 0 {
			WriteJSONHelper(w, http.StatusInternalServerError, APIError{Error: errorsSlice[0].Error()})
			return
		}

		if r.Method == string(routeInfo.Method) {
			if err := routeInfo.Handler(ctx); err != nil {
				if err := WriteJSONHelper(w, http.StatusInternalServerError, APIError{Error: err.Error()}); err != nil {
					fmt.Println("Erro ao escrever resposta JSON:", err)
				}
			}
		} else {
			WriteJSONHelper(w, http.StatusMethodNotAllowed, APIError{Error: "Método HTTP não permitido"})
		}
	}
}

func AddRoutes(groupMiddlewares MiddlewareChain, routeFuncs ...func() []RouteInfo) {
	for _, routeFunc := range routeFuncs {
		routes := routeFunc()
		for i := range routes {
			routes[i].Middlewares = append(routes[i].Middlewares, groupMiddlewares...)
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
	return tc.request
}

func (tc *TupaContext) Response() *http.ResponseWriter {
	return &tc.response
}

func (tc *TupaContext) SendString(s string) error {
	_, err := tc.response.Write([]byte(s))
	return err
}

func (tc *TupaContext) Param(param string) string {
	return mux.Vars(tc.request)[param]
}

func (tc *TupaContext) QueryParam(param string) string {
	return tc.Request().URL.Query().Get(param)
}

func (tc *TupaContext) QueryParams() map[string][]string {
	return tc.Request().URL.Query()
}

func (tc *TupaContext) SetRequest(r *http.Request) {
	tc.request = r
}
