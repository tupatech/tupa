package tupa

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

type Controller struct{}

type APIFunc func(http.ResponseWriter, *http.Request) error

type APIServer struct {
	listenAddr string
}

// func CreateController(name string) interface{} {

// 	structName := name

// 	type structName struct {
// 		Controller
// 	}

// }

func WriteJSONHelper(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")

	if w == nil {
		return errors.New("Response writer passado está nulo")
	}

	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func MakeHTTPHandlerFuncHelper(f APIFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			if err := WriteJSONHelper(w, http.StatusInternalServerError, ApiError{Error: err.Error()}); err != nil {
				fmt.Println("Erro ao escrever resposta JSON:", err)
			}
		}
	}
}

func RegisterRoutes(router *mux.Router, route string, methods []string, handler APIFunc) {
	for _, method := range methods {
		router.HandleFunc(route, MakeHTTPHandlerFuncHelper(handler)).Methods(method)
	}
}

func (a *APIServer) New() {

}
