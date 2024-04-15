package tupa

type APIError struct {
	Error string
}

type APIHandlerErr struct {
	Status int
	Msg    string
}

func (e APIHandlerErr) Error() string {
	return e.Msg
}

type HandleFunc func(*TupaContext) error

type HTTPMethod string

type RouteInfo struct {
	Path        string
	Method      HTTPMethod
	Handler     APIFunc
	Middlewares []MiddlewareFunc
}
