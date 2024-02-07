package tupa

import "sync"

type MiddlewareFunc func(APIFunc) APIFunc

type MiddlewareChain []MiddlewareFunc

func (chain *MiddlewareChain) Use(middleware ...MiddlewareFunc) {
	*chain = append(*chain, middleware...)
}

func (a *APIServer) UseGlobalMiddleware(middleware ...MiddlewareFunc) {
	a.globalMiddlewares.Use(middleware...)
}

func (a *APIServer) GetGlobalMiddlewares() []MiddlewareFunc {
	return a.globalMiddlewares
}

// Chain of responsibility
func (chain *MiddlewareChain) execute(tc *TupaContext) error {
	for _, middleware := range *chain {
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

func (a *APIServer) executeMiddlewaresAsync(ctx *TupaContext, middlewares ...MiddlewareChain) <-chan []error {
	doneCh := make(chan []error, 1)

	var wg sync.WaitGroup

	var errorsSlice []error

	// Executa cada middleware em uma goroutine separada
	for _, middlewareChain := range middlewares {
		wg.Add(1)
		go func(chain MiddlewareChain) {
			defer wg.Done()

			// Executa o middleware
			if err := chain.execute(ctx); err != nil {
				errorsSlice = append(errorsSlice, err)
			}
		}(middlewareChain)
	}

	// Começa goroutine para fechar os channels, um de cada vez ( vai enviar os valores um por vez, a medida que chegarem)
	go func() {
		wg.Wait()
		doneCh <- errorsSlice
		close(doneCh)
	}()

	return doneCh
}
