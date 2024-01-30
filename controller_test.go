package tupa

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func BenchmarkDirectAccessSendString(b *testing.B) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	ctx := &TupaContext{
		request:  req,
		response: w,
	}

	start := time.Now()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.SendString(http.StatusOK, "Hello, World!")
	}

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec, "ops/sec")
}

func TestParam(t *testing.T) {
	t.Run("Teste método Param com parametro", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users/123", nil)
		if err != nil {
			t.Fatal(err)
		}

		// Colocando um parametro na rota da requisição
		req = mux.SetURLVars(req, map[string]string{
			"id": "123",
		})

		tc := &TupaContext{
			request: req,
		}

		got := tc.Param("id")
		want := "123"
		if got != want {
			t.Errorf("Parametro retornado %s, queria %s", got, want)
		}
	})

	t.Run("Teste método Param sem parametro", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users/123", nil)
		if err != nil {
			t.Fatal(err)
		}

		tc := &TupaContext{
			request: req,
		}

		got := tc.Param("id")
		want := ""
		if got != want {
			t.Errorf("Parametro retornado %s, queria %s", got, want)
		}
	})
}
