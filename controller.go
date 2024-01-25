package tupa

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Controller struct{}

type APIFunc func(http.ResponseWriter, *http.Request) error

type APIServer struct {
	listenAddr string
	router     *mux.Router
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

	if a.router.GetRoute("/") == nil {
		a.router.HandleFunc("/", WelcomeHandler).Methods(http.MethodGet)
	}

	fmt.Println(FmtBlue("Servidor iniciado na porta: " + a.listenAddr))

	routerHandler := cors.Default().Handler(a.router)

	if err := http.ListenAndServe(a.listenAddr, routerHandler); err != nil {
		log.Fatal(FmtRed("Erro ao iniciar servidor: "), err)
	}
}

func NewApiServer(listenAddr string) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		router:     mux.NewRouter(),
		// store:      store,
	}
}

func (a *APIServer) SetDefaultRoute(handlers map[HTTPMethod]APIFunc) {
	defController := Controller{}
	for method, handler := range handlers {
		a.router.HandleFunc("/", defController.MakeHTTPHandlerFuncHelper(handler, method)).Methods(string(method))
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
		return errors.New("response writer passado está nulo")
	}

	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func (c *Controller) MakeHTTPHandlerFuncHelper(f APIFunc, httpMethod HTTPMethod) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == string(httpMethod) {
			if err := f(w, r); err != nil {
				if err := WriteJSONHelper(w, http.StatusInternalServerError, APIError{Error: err.Error()}); err != nil {
					fmt.Println("Erro ao escrever resposta JSON:", err)
				}
			}
		} else {
			WriteJSONHelper(w, http.StatusMethodNotAllowed, APIError{Error: "Método HTTP não permitido"})
		}
	}
}
