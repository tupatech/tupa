package tupa

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouter(t *testing.T) {
	t.Run("Testing NewRouter Initialization", func(t *testing.T) {
		r := NewRouter()
		if r == nil {
			t.Error("NewRouter() should return a Router")
		}
	})
}

func TestRouterHandle(t *testing.T) {
	t.Run("Testing Router Handle response and status", func(t *testing.T) {
		r := NewRouter()
		r.Handle("GET", "/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello, World!"))
		})

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		if string(body) != "Hello, World!" {
			t.Fatalf("expected body 'Hello, World!', got %s", string(body))
		}
	})
}
