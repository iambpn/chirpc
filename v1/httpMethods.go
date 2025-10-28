package chirpc

import "net/http"

// type alias for http methods
type HttpMethods = string

const (
	GET     HttpMethods = http.MethodGet
	POST    HttpMethods = http.MethodPost
	PUT     HttpMethods = http.MethodPut
	DELETE  HttpMethods = http.MethodDelete
	PATCH   HttpMethods = http.MethodPatch
	OPTIONS HttpMethods = http.MethodOptions
	HEAD    HttpMethods = http.MethodHead
	TRACE   HttpMethods = http.MethodTrace
	CONNECT HttpMethods = http.MethodConnect
)
