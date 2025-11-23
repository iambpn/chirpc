package chirpc

import "net/http"

// type alias for http methods
// used for defining allowed HTTP methods in RPC routes
type HttpMethods = string

const (
	MethodGet     HttpMethods = http.MethodGet
	MethodPost    HttpMethods = http.MethodPost
	MethodPut     HttpMethods = http.MethodPut
	MethodDelete  HttpMethods = http.MethodDelete
	MethodPatch   HttpMethods = http.MethodPatch
	MethodOptions HttpMethods = http.MethodOptions
	MethodHead    HttpMethods = http.MethodHead
	MethodTrace   HttpMethods = http.MethodTrace
	MethodConnect HttpMethods = http.MethodConnect
)
