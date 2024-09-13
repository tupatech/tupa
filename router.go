package tupa

import (
	"net/http"
)

// MIDDLEWARES NOT BEING USED LIKE THAT 'YET?'
type Middleware func(http.Handler) http.Handler
type MiddChain []Middleware

type Router struct {
	Mux *http.ServeMux
	// Middlewares []Middleware
}

func NewRouter() *Router {
	// func NewRouter(mw ...Middleware) *Router {
	return &Router{
		Mux: http.NewServeMux(),
		// Middlewares: mw,
	}
}

// MIDDLEWARES NÃO ESTÃO SENDO USADOS AINDA
func (r *Router) Handle(method, path string, fn http.HandlerFunc, mw ...Middleware) {
	// wrappedHandler := r.Wrap(fn, mw...)
	r.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.NotFound(w, r)
			return
		}
		// path será a assinatura da URL na api. r.URL.Path será a assinatura da URL no client
		println("path: ", path)
		println("path: ", r.URL.Path)
		params := extractParams(path, r.URL.Path)
		r = WithVars(r, params)
		fn.ServeHTTP(w, r)
	})
}

// implementando a interface Handler do método http para usar o router
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Mux.ServeHTTP(w, req)
}

// func (r *Router) Use(mw ...Middleware) {
// 	r.Middlewares = append(r.Middlewares, mw...)
// }

// 1. A função wrap vai garantir que todos os middlewares sejam executados antes do handler e em qualquer handler que quisermos
// 2. Permite combinar middlewares do router com qualquer um que seja passado para a função wrap
// 3. Garante que os Middlewares sejam executados na ordem correta ( LIFO - Last In First Out )
// func (r *Router) Wrap(fn http.HandlerFunc, mw ...Middleware) (out http.Handler) {
// 	// http.Handler(fn) é um cast ( convertendo ) de fn para http.Handler
// 	// append(r.Middlewares, mw...) vai juntar os middlewares do router com os middlewares passados para a função wrap
// 	out, mw = http.Handler(fn), append(r.Middlewares, mw...)
// 	slices.Reverse(mw)
// 	// O loop abaixo vai garantir que todos os middlewares sejam executados antes do handler
// 	for _, m := range mw {
// 		out = m(out)
// 	}
// 	// Retorna o handler com todos os middlewares
// 	return
// }
