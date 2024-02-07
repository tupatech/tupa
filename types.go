package tupa

type APIError struct {
	Error string
}

type HTTPMethod string

type RouteInfo struct {
	Path        string
	Method      HTTPMethod
	Handler     APIFunc
	Middlewares []MiddlewareFunc
}
