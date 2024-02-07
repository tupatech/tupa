package tupa

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func MiddlewareSample(next APIFunc) APIFunc {

	return func(tc *TupaContext) error {
		smpMidd := "sampleMiddleware"
		reqCtx := context.WithValue(tc.request.Context(), "smpMidd", smpMidd)
		tc.request = tc.request.WithContext(reqCtx)

		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
		fmt.Println("Middleware antes de chamar o handler")

		defer getCtxFromSampleMiddleware(tc)
		defer fmt.Println("SampleMiddleware depois de chamar o handler")
		return next(tc)
	}
}

func getCtxFromSampleMiddleware(tc *TupaContext) {
	ctxValue := tc.request.Context().Value("smpMidd").(string)

	fmt.Println(ctxValue)
}

func MiddlewareLoggingWithError(next APIFunc) APIFunc {
	return func(tc *TupaContext) error {

		start := time.Now()
		errMsg := errors.New("erro no middleware LoggingMiddlewareWithError")
		ctx := context.WithValue(tc.request.Context(), "smpErrorMidd", "sampleErrorMiddleware")
		tc.request = tc.request.WithContext(ctx)

		err := next(tc)
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
		log.Printf("Fim da request para endpoint teste: %s, duração: %v", tc.request.URL.Path, time.Since(start))

		if err != nil {
			log.Printf("erro ao chamar proximo middleware %s: %v", tc.request.URL.Path, err)
		}

		return errMsg
	}
}

func MiddlewareWithCtx(next APIFunc) APIFunc {

	return func(tc *TupaContext) error {
		smpMidd := "MiddlewareWithCtx"
		reqCtx := context.WithValue(tc.request.Context(), "withCtx", smpMidd)
		tc.request = tc.request.WithContext(reqCtx)

		log.SetFlags(log.LstdFlags | log.Lmicroseconds)

		return next(tc)
	}
}

func MiddlewareWithCtxChanMsg(next APIFunc, messages chan<- string) APIFunc {

	return func(tc *TupaContext) error {
		smpMidd := "MiddlewareWithCtx"
		reqCtx := context.WithValue(tc.request.Context(), "withCtx", smpMidd)
		tc.request = tc.request.WithContext(reqCtx)

		log.SetFlags(log.LstdFlags | log.Lmicroseconds)

		messages <- "MiddlewareWithCtx passou por aqui :)"

		return next(tc)
	}
}

func TestSampleMiddleware(t *testing.T) {
	t.Run("Testado TestSampleMiddleware", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		ctx := &TupaContext{
			request:  req,
			response: w,
		}

		// handler que não retorna erro
		handler := func(tc *TupaContext) error {
			// Chacando se o middleware tem valor de context
			if val := tc.request.Context().Value("qualquerKey"); val != nil {
				t.Error("Valor de context esperado não era esperado")
			}
			return nil
		}

		err := MiddlewareSample(handler)(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Checando status code da response
		if status := w.Result().StatusCode; status != http.StatusOK {
			t.Errorf("handler retornou status code errado: recebeu %v queria %v", status, http.StatusOK)
		}
	})
}

func TestMiddlewareConcurrency(t *testing.T) {

	t.Run("Testando MiddlewareConcurrency", func(t *testing.T) {
		numGoroutines := 1000
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		middleware := MiddlewareWithCtx

		handler := func(tc *TupaContext) error {
			ctxValue := tc.request.Context().Value("withCtx").(string)

			if ctxValue != "MiddlewareWithCtx" {
				t.Errorf("Esperava valor de context 'MiddlewareWithCtx', recebeu '%s'", ctxValue)
			}

			return nil
		}

		for i := 0; i < numGoroutines; i++ {
			go func() {
				// Criando nova request com context para cada goroutine
				req := httptest.NewRequest("GET", "/test", nil)
				w := httptest.NewRecorder()

				ctx := &TupaContext{
					request:  req,
					response: w,
				}

				// Chanmando o middleware com o handler e o contexto
				err := middleware(handler)(ctx)
				if err != nil {
					t.Errorf("erro nao esperado: %v", err)
				}

				wg.Done()
			}()
		}

		wg.Wait()
	})
}

func TestMiddlewareWithCtxAndChannels(t *testing.T) {
	numGoroutines := 1000
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	ctxValues := make(chan string, numGoroutines)

	middleware := MiddlewareWithCtx

	handler := func(tc *TupaContext) error {
		ctxValue := tc.request.Context().Value("withCtx").(string)

		ctxValues <- ctxValue

		return nil
	}

	for i := 0; i < numGoroutines; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			ctx := &TupaContext{
				request:  req,
				response: w,
			}

			err := middleware(handler)(ctx)
			if err != nil {
				t.Errorf("erro inesperado: %v", err)
			}

			wg.Done()
		}()
	}

	wg.Wait()
	close(ctxValues)

	for ctxValue := range ctxValues {
		if ctxValue != "MiddlewareWithCtx" {
			t.Errorf("Esperava 'MiddlewareWithCtx', recebeu '%s'", ctxValue)
		}
	}
}

func TestMiddlewareWithCtxAndChannelAndMsg(t *testing.T) {
	numGoroutines := 1000
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	messages := make(chan string, numGoroutines)

	middleware := func(next APIFunc) APIFunc {
		return MiddlewareWithCtxChanMsg(next, messages)
	}

	handler := func(tc *TupaContext) error {
		return nil
	}

	for i := 0; i < numGoroutines; i++ {
		go func() {
			// Criando nova req para cada goroutine
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			ctx := &TupaContext{
				request:  req,
				response: w,
			}

			err := middleware(handler)(ctx)
			if err != nil {
				t.Errorf("erro inesperado: %v", err)
			}

			wg.Done()
		}()
	}

	wg.Wait()

	close(messages)

	for message := range messages {
		if message != "MiddlewareWithCtx passou por aqui :)" {
			t.Errorf("Expected message 'MiddlewareWithCtx passou por aqui :)', got '%s'", message)
		}
	}
}

func middlewareSuccess(next APIFunc) APIFunc {
	return func(tc *TupaContext) error {
		return nil
	}
}

// Define a middleware function that always fails
func middlewareFailure(ctx APIFunc) APIFunc {
	return func(tc *TupaContext) error {
		return errors.New("middleware failed")
	}
}

func TestExecuteMiddlewaresAsync_NoErrors(t *testing.T) {
	t.Run("Testando ExecuteMiddlewaresAsync sem erros", func(t *testing.T) {
		server := NewAPIServer(":8080")

		ctx := &TupaContext{}

		// Definir middleware que sempre retorna sucesso
		middlewareSuccess := MiddlewareChain{middlewareSuccess}

		// Execute os middlewares de forma assíncrona
		doneCh := server.executeMiddlewaresAsync(ctx, middlewareSuccess)

		// esperando executar
		errorsSlice := <-doneCh

		if len(errorsSlice) != 0 {
			t.Errorf("não esperava erros, recebeu %d erros", len(errorsSlice))
		}
	})

	t.Run("Testando ExecuteMiddlewaresAsync com erros", func(t *testing.T) {
		server := NewAPIServer(":8080")

		ctx := &TupaContext{}

		// Definir middleware que sempre retorna falha
		middlewareFailure := MiddlewareChain{middlewareFailure}

		// Executando os middlewares de forma assíncrona
		doneCh := server.executeMiddlewaresAsync(ctx, middlewareFailure)

		// esperando executar
		errorsSlice := <-doneCh

		if len(errorsSlice) == 0 {
			t.Errorf("expected erros mas não recebeu nenhum")
		}
	})
}
