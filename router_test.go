package tupa

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterHandle(t *testing.T) {
	tests := []struct {
		method     string
		path       string
		reqMethod  string
		reqPath    string
		statusCode int
		body       string
	}{
		{
			method:     "GET",
			path:       "/users/{id}",
			reqMethod:  "GET",
			reqPath:    "/users/123",
			statusCode: http.StatusOK,
			body:       "123",
		},
		{
			method:     "GET",
			path:       "/users/{id}/books/{bookId}",
			reqMethod:  "GET",
			reqPath:    "/users/123/books/456",
			statusCode: http.StatusOK,
			body:       "123 456",
		},
		{
			method:     "GET",
			path:       "/users/{id}/books/{bookId}",
			reqMethod:  "GET",
			reqPath:    "/users/123/books/",
			statusCode: http.StatusNotFound,
			body:       "404 page not found\n",
		},
		{
			method:     "GET",
			path:       "/users/{id}/books/{bookId}",
			reqMethod:  "GET",
			reqPath:    "/users/123/books",
			statusCode: http.StatusNotFound,
			body:       "404 page not found\n",
		},
		{
			method:     "GET",
			path:       "/users/{id}",
			reqMethod:  "GET",
			reqPath:    "/users/",
			statusCode: http.StatusNotFound,
			body:       "404 page not found\n",
		},
		{
			method:     "GET",
			path:       "/users/{id}",
			reqMethod:  "GET",
			reqPath:    "/user/123",
			statusCode: http.StatusNotFound,
			body:       "404 page not found\n",
		},
		{
			method:     "GET",
			path:       "/",
			reqMethod:  "GET",
			reqPath:    "/",
			statusCode: http.StatusOK,
			body:       "",
		},
	}

	for _, test := range tests {
		t.Run(test.method+"_"+test.path+"_"+test.reqMethod+"_"+test.reqPath, func(t *testing.T) {
			router := NewRouter()
			router.Handle(test.method, test.path, func(w http.ResponseWriter, r *http.Request) {
				params := Vars(r)
				if id, ok := params["id"]; ok {
					w.Write([]byte(id))
				}
				if bookId, ok := params["bookId"]; ok {
					w.Write([]byte(" " + bookId))
				}
			})

			req := httptest.NewRequest(test.reqMethod, test.reqPath, nil)
			rr := httptest.NewRecorder()

			router.Mux.ServeHTTP(rr, req)

			resp := rr.Result()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != test.statusCode {
				t.Errorf("expected status %d, got %d", test.statusCode, resp.StatusCode)
			}
			if string(body) != test.body {
				t.Errorf("expected body %q, got %q", test.body, string(body))
			}
		})
	}
}
