package tupa

type MiddlewareFunc func(APIFunc) APIFunc

type MiddlewareChain struct {
	middlewares []MiddlewareFunc
}

func (chain *MiddlewareChain) Use(middleware ...MiddlewareFunc) {
	chain.middlewares = append(chain.middlewares, middleware...)
}

// Chain of responsibility
func (chain *MiddlewareChain) execute(tc *TupaContext) error {
	for _, middleware := range chain.middlewares {
		// Cada middleware recebe um função next de argumento ( do tipo func(APIFunc) APIFunc)
		// Cada função next é esperado que seja um middleware por si próprio
		next := middleware(func(tc *TupaContext) error {
			return nil
		})

		// chamando a função retornada por middleware(...)
		if err := next(tc); err != nil {
			// se qualquer middleware na chain tiver erro vai retornar o erro
			return err
		}
	}

	return nil
}
