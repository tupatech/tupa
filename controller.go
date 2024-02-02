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
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type (
	Context interface {
		Request() *http.Request
		Response() http.ResponseWriter
		SendString(status int, s string) error
		Param(param string) string
	}

	TupaContext struct {
		request  *http.Request
		response http.ResponseWriter
		context.Context
	}
)

type APIFunc func(*TupaContext) error

type APIServer struct {
	listenAddr string
	server     *http.Server
}

var (
	globalRouter     *mux.Router
	globalRouterOnce sync.Once
)

const (
	MethodGet     HTTPMethod = http.MethodGet
	MethodPost    HTTPMethod = http.MethodPost
	MethodPut     HTTPMethod = http.MethodPut
	MethodDelete  HTTPMethod = http.MethodDelete
	MethodPatch   HTTPMethod = http.MethodPatch
	MethodOptions HTTPMethod = http.MethodOptions
)

type DefaultController struct {
	router *mux.Router
}

var AllowedMethods = map[HTTPMethod]bool{
	MethodGet:    true,
	MethodPost:   true,
	MethodPut:    true,
	MethodDelete: true,
	MethodPatch:  true,
}

func (a *APIServer) New() {
	globalRouter = getGlobalRouter()

	if globalRouter.GetRoute("/") == nil {
		globalRouter.HandleFunc("/", WelcomeHandler).Methods(http.MethodGet)
	}

	routerHandler := cors.Default().Handler(globalRouter)

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

func NewApiServer(listenAddr string) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		// store:      store,
	}
}

func (dc *DefaultController) SetDefaultRoute(handlers map[HTTPMethod]APIFunc) {
	for method, handler := range handlers {
		dc.router.HandleFunc("/", dc.MakeHTTPHandlerFuncHelper(handler, method)).Methods(string(method))
	}
}

func WelcomeHandler(w http.ResponseWriter, r *http.Request) {
	WriteJSONHelper(w, http.StatusOK, "Seja bem vindo ao Tupã framework!")
}

func (dc *DefaultController) RegisterRoutes(route string, handlers map[HTTPMethod]APIFunc, middlewares ...MiddlewareChain) {
	for method, handler := range handlers {
		if !AllowedMethods[method] {
			log.Fatalf(fmt.Sprintf(FmtRed("Método HTTP não permitido: "), "%s\nVeja como criar um novo método na documentação", method))
		}

		// especificando chain de middlewares que devem ser executadas antes do handler de uma rota particular
		// Na Chain of responsibility, a request é passada pela cadeia de handlers, onde cada handler pode fazer um processamento ou modificação
		chainHandler := func(tc *TupaContext) error {
			for _, middlewareChain := range middlewares {
				if err := middlewareChain.execute(tc); err != nil {
					return err
				}
			}

			// depois da middleware chain ser processada, chama o handler original
			return handler(tc)
		}

		dc.router.HandleFunc(route, dc.MakeHTTPHandlerFuncHelper(chainHandler, method)).Methods(string(method))
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

func (dc *DefaultController) MakeHTTPHandlerFuncHelper(f APIFunc, httpMethod HTTPMethod) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := &TupaContext{
			request:  r,
			response: w,
		}
		if r.Method == string(httpMethod) {
			if err := f(ctx); err != nil {
				if err := WriteJSONHelper(w, http.StatusInternalServerError, APIError{Error: err.Error()}); err != nil {
					fmt.Println("Erro ao escrever resposta JSON:", err)
				}
			}
		} else {
			WriteJSONHelper(w, http.StatusMethodNotAllowed, APIError{Error: "Método HTTP não permitido"})
		}
	}
}

func NewController() *DefaultController {
	return &DefaultController{router: getGlobalRouter()}
}

func getGlobalRouter() *mux.Router {
	globalRouterOnce.Do(func() {
		globalRouter = mux.NewRouter()
	})
	return globalRouter
}

func (tc *TupaContext) Request() *http.Request {
	return tc.request
}

func (tc *TupaContext) Response() *http.ResponseWriter {
	return &tc.response
}

func (tc *TupaContext) SendString(status int, s string) error {
	(tc.response).WriteHeader(status)
	_, err := (tc.response).Write([]byte(s))
	return err
}

func (tc *TupaContext) Param(param string) string {
	return mux.Vars(tc.request)[param]
}
