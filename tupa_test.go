package tupa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func BenchmarkDirectAccessSendString(b *testing.B) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	ctx := &TupaContext{
		Req:  req,
		Resp: w,
	}

	start := time.Now()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.SendString("Hello, World!")
	}

	elapsed := time.Since(start)
	opsPerSec := float64(b.N) / elapsed.Seconds()
	b.ReportMetric(opsPerSec, "ops/sec")
}

func TestParam(t *testing.T) {
	t.Run("Teste método Param com parametro", func(t *testing.T) {
		router := NewRouter()

		router.Handle("GET", "/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			tc := &TupaContext{Req: r, Resp: w}
			id := tc.Param("id")
			w.Write([]byte(id))
		})

		req, _ := http.NewRequest("GET", "/users/123", nil)
		rr := httptest.NewRecorder()
		router.Mux.ServeHTTP(rr, req)

		expected := "123"
		if rr.Body.String() != expected {
			t.Errorf("expected body %s, got %s", expected, rr.Body.String())
		}
	})

	t.Run("Teste método Param sem parametro", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users/123", nil)
		if err != nil {
			t.Fatal(err)
		}

		tc := &TupaContext{
			Req: req,
		}

		got := tc.Param("id")
		want := ""
		if got != want {
			t.Errorf("Parametro retornado %s, queria %s", got, want)
		}
	})
}

func TestParams(t *testing.T) {
	t.Run("Teste método Params com parametro", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users/123", nil)
		if err != nil {
			t.Fatal(err)
		}

		tc := &TupaContext{
			Req: req,
		}

		// Colocando um parametro na rota da requisição
		tc.Req = tc.SetURLVars(req, map[string]string{
			"id": "123",
		})

		got := tc.Params()
		want := map[string]string{
			"id": "123",
		}
		if got["id"] != want["id"] {
			t.Errorf("parametro retornado %v, queria %v", got, want)
		}
	})

	t.Run("Teste método Params sem parametro", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users", nil)
		if err != nil {
			t.Fatal(err)
		}

		tc := &TupaContext{
			Req: req,
		}

		got := tc.Params()
		want := map[string]string{}
		if len(got) != len(want) {
			t.Errorf("parametro retornado %v, queria %v", got, want)
		}
	})
}

func TestQueryParam(t *testing.T) {
	t.Run("Teste método QueryParam com parametro", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users?name=Victor", nil)
		if err != nil {
			t.Fatal(err)
		}

		tc := &TupaContext{
			Req: req,
		}

		got := tc.QueryParam("name")
		want := "Victor"
		if got != want {
			t.Errorf("parametro retornado %s, queria %s", got, want)
		}
	})

	t.Run("Teste método QueryParam sem parametro", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users", nil)
		if err != nil {
			t.Fatal(err)
		}

		tc := &TupaContext{
			Req: req,
		}

		got := tc.QueryParam("name")
		want := ""
		if got != want {
			t.Errorf("parametro retornado %s, queria %s", got, want)
		}
	})
}

func TestQueryParams(t *testing.T) {
	t.Run("Teste método QueryParams com parametro", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users?name=Victor&age=24", nil)
		if err != nil {
			t.Fatal(err)
		}

		tc := &TupaContext{
			Req: req,
		}

		got := tc.QueryParams()
		want := map[string]string{
			"name": "Victor",
			"age":  "24",
		}
		if got["name"][0] != want["name"] || got["age"][0] != want["age"] {
			t.Errorf("parametro retornado %v, queria %v", got, want)
		}
	})

	t.Run("Teste método QueryParams sem parametro", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users", nil)
		if err != nil {
			t.Fatal(err)
		}

		tc := &TupaContext{
			Req: req,
		}

		got := tc.QueryParams()
		want := map[string][]string{}
		if len(got) != len(want) {
			t.Errorf("parametro retornado %v, queria %v", got, want)
		}
	})
}

func TestNewTupaContext(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := context.Background()

	tupaCtx := NewTupaContext(&req, resp, ctx)

	if tupaCtx.Req != req {
		t.Errorf("Expected request to be %v, but got %v", req, tupaCtx.Req)
	}

	if tupaCtx.Resp != resp {
		t.Errorf("Expected response to be %v, but got %v", resp, tupaCtx.Resp)
	}

	if tupaCtx.Ctx != ctx {
		t.Errorf("Expected context to be %v, but got %v", ctx, tupaCtx.Ctx)
	}
}

func TestNewTupaContextRaceCondition(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = NewTupaContext(&req, resp, ctx)
		}()
	}
	wg.Wait()
}

func BenchmarkNewTupaContext(b *testing.B) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewTupaContext(&req, resp, ctx)
	}
}

func TestWriteJSONHelperStatuses(t *testing.T) {
	t.Run("Teste método WriteJSONHelper com status 200", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users", nil)
		if err != nil {
			t.Fatal(err)
		}

		tc := &TupaContext{
			Req:  req,
			Resp: httptest.NewRecorder(),
		}

		WriteJSONHelper(tc.Resp, http.StatusOK, "Hello, World!")
		if tc.Resp.(*httptest.ResponseRecorder).Code != 200 {
			t.Errorf("status retornado %d, queria %d", tc.Resp.(*httptest.ResponseRecorder).Code, 200)
		}
	})

	t.Run("Teste método WriteJSONHelper com status 400", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users", nil)
		if err != nil {
			t.Fatal(err)
		}

		tc := &TupaContext{
			Req:  req,
			Resp: httptest.NewRecorder(),
		}

		WriteJSONHelper(tc.Resp, http.StatusBadRequest, "Hello, World!")
		if tc.Resp.(*httptest.ResponseRecorder).Code != 400 {
			t.Errorf("status retornado %d, queria %d", tc.Resp.(*httptest.ResponseRecorder).Code, 400)
		}
	})

	t.Run("Teste método WriteJSONHelper com status 500", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users", nil)
		if err != nil {
			t.Fatal(err)
		}

		tc := &TupaContext{
			Req:  req,
			Resp: httptest.NewRecorder(),
		}

		WriteJSONHelper(tc.Resp, http.StatusInternalServerError, "Hello, World!")
		if tc.Resp.(*httptest.ResponseRecorder).Code != 500 {
			t.Errorf("status retornado %d, queria %d", tc.Resp.(*httptest.ResponseRecorder).Code, 500)
		}
	})

	t.Run("Teste método WriteJSONHelper com status Unauthorized", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/users", nil)
		if err != nil {
			t.Fatal(err)
		}

		tc := &TupaContext{
			Req:  req,
			Resp: httptest.NewRecorder(),
		}

		WriteJSONHelper(tc.Resp, http.StatusUnauthorized, "Hello, World!")
		if tc.Resp.(*httptest.ResponseRecorder).Code != 401 {
			t.Errorf("status retornado %d, queria %d", tc.Resp.(*httptest.ResponseRecorder).Code, 401)
		}
	})
}
