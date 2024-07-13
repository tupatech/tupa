package tupa

import (
	"net/http"
	"slices"
)

// MIDDLEWARES NOT BEING USED LIKE THAT 'YET?'
type Middleware func(http.Handler) http.Handler
type MiddChain []Middleware

type Router struct {
	Mux         *http.ServeMux
	Middlewares []Middleware
	// *http.Handler
}

func NewRouter(mw ...Middleware) *Router {
	return &Router{
		Mux: http.NewServeMux(),
		// Middlewares: mw,
	}
}

// func (r *Router) Use(mw ...Middleware) {
// 	r.Middlewares = append(r.Middlewares, mw...)
// }

// 1. A função wrap vai garantir que todos os middlewares sejam executados antes do handler e em qualquer handler que quisermos
// 2. Permite combinar middlewares do router com qualquer um que seja passado para a função wrap
// 3. Garante que os Middlewares sejam executados na ordem correta ( LIFO - Last In First Out )
func (r *Router) Wrap(fn http.HandlerFunc, mw ...Middleware) (out http.Handler) {
	// http.Handler(fn) é um cast ( convertendo ) de fn para http.Handler
	// append(r.Middlewares, mw...) vai juntar os middlewares do router com os middlewares passados para a função wrap
	out, mw = http.Handler(fn), append(r.Middlewares, mw...)
	slices.Reverse(mw)
	// O loop abaixo vai garantir que todos os middlewares sejam executados antes do handler
	for _, m := range mw {
		out = m(out)
	}
	// Retorna o handler com todos os middlewares
	return
}

// MIDDLEWARES ARE NOT BEING USED HERE YET
func (r *Router) Handle(method, path string, fn http.HandlerFunc, mw ...Middleware) {
	r.Mux.Handle(method+" "+path, r.Wrap(fn, mw...))
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// method := req.Method
	// path := req.URL.Path

	r.Mux.ServeHTTP(w, req)
}
