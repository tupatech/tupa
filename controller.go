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

type Controller struct{}

type APIFunc func(context.Context, http.ResponseWriter, *http.Request) error

type APIServer struct {
	listenAddr string
	Router     *mux.Router
	server     *http.Server
}

type HTTPMethod string

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

func (a *APIServer) New() {

	if a.Router.GetRoute("/") == nil {
		a.Router.HandleFunc("/", WelcomeHandler).Methods(http.MethodGet)
	}

	routerHandler := cors.Default().Handler(a.Router)

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
		Router:     mux.NewRouter(),
		// store:      store,
	}
}

func (a *APIServer) SetDefaultRoute(handlers map[HTTPMethod]APIFunc) {
	defController := Controller{}
	for method, handler := range handlers {
		a.Router.HandleFunc("/", defController.MakeHTTPHandlerFuncHelper(handler, method)).Methods(string(method))
	}
}

func WelcomeHandler(w http.ResponseWriter, r *http.Request) {
	WriteJSONHelper(w, http.StatusOK, "Seja bem vindo ao Tupã framework!")
}

func (c *Controller) RegisterRoutes(router *mux.Router, route string, handlers map[HTTPMethod]APIFunc) {
	for method, handler := range handlers {
		if !AllowedMethods[method] {
			log.Fatal(fmt.Sprintf(FmtRed("Método HTTP não permitido: "), "%s\nVeja como criar um novo método na documentação", method))
		}
		router.HandleFunc(route, c.MakeHTTPHandlerFuncHelper(handler, method)).Methods(string(method))
	}
}

func WriteJSONHelper(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")

	if w == nil {
		return errors.New("Response writer passado está nulo")
	}

	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func (c *Controller) MakeHTTPHandlerFuncHelper(f APIFunc, httpMethod HTTPMethod) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Method == string(httpMethod) {
			if err := f(ctx, w, r); err != nil {
				if err := WriteJSONHelper(w, http.StatusInternalServerError, APIError{Error: err.Error()}); err != nil {
					fmt.Println("Erro ao escrever resposta JSON:", err)
				}
			}
		} else {
			WriteJSONHelper(w, http.StatusMethodNotAllowed, APIError{Error: "Método HTTP não permitido"})
		}
	}
}
