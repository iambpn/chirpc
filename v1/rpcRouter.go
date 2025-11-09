package chirpc

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/iambpn/chirpc/internal/rpcType"
)

// RPCRouter provides a thin wrapper around a chi.Mux router and exposes
// helper methods for registering RPC-style handlers, mounting sub-routers,
// and producing an http.Server. It centralizes middleware registration and
// error handler wiring for the chirpc package.
type RPCRouter struct {
	router *chi.Mux
}

// errorHandler holds the package-wide error handling function used by RPC handlers.
// It is set via RegisterErrorHandler and injected into each handler through
// ServeHTTPWithErrorHandler. When unset (nil), handlers rely on their default
// error behavior. The generic [any] form allows a single global handler to be
// applied across different response body types.
var errorHandler ErrorHandlerType[any]

// GetHttpServer returns an *http.Server that uses the underlying chi router as its Handler.
func (r *RPCRouter) GetHttpServer() *http.Server {
	return &http.Server{
		Handler: r.router,
	}
}

// ListenAndServe starts an HTTP server on the provided address using the internal router.
func (r *RPCRouter) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, r.router)
}

// NewRPCRouter constructs a new RPCRouter with a fresh chi.Mux instance.
func NewRPCRouter() *RPCRouter {
	return &RPCRouter{
		router: chi.NewRouter(),
	}
}

// AddGlobalMiddlewares attaches the provided middlewares to the root router.
func AddGlobalMiddlewares(r *RPCRouter, middlewares ...MiddlewareType) {
	r.router.Use(middlewares...)
}

// AddHandler registers an RPC handler for the given HTTP method and path, applies optional middlewares,
// records its schema for TypeScript generation, and returns a BodyQueryParamType to allow parameter configuration.
func AddHandler[R any](r *RPCRouter, method HttpMethods, path string, handler RequestHandler[R], middlewares ...MiddlewareType) *rpcType.BodyQueryParamType {
	bodyQueryParam := &rpcType.BodyQueryParamType{}

	// register handler type to generate ts types
	schema, err := rpcType.RegisterHandler(string(method), path, handler)

	if err != nil {
		fmt.Fprintf(os.Stderr, "an error occurred while registering handler: \n %s", err)
		return bodyQueryParam
	}

	bodyQueryParam.Schema = schema

	// get slugs from path
	slugs := parseURLSlug(path)
	bodyQueryParam.Params(slugs)

	r.router.With(middlewares...).Method(string(method), path, handler.ServeHTTPWithErrorHandler(errorHandler))

	return bodyQueryParam
}

// Route creates a sub-route at the specified path, applies middlewares to it, and invokes the callback to populate it.
func Route(r *RPCRouter, path string, fn func(r *RPCRouter), middlewares ...MiddlewareType) {
	r.router.Route(path, func(r chi.Router) {
		subRouter := &RPCRouter{router: chi.NewRouter()}
		AddGlobalMiddlewares(subRouter, middlewares...)
		fn(subRouter)
		r.Mount("/", subRouter.router)
	})
}

// Mount mounts an existing RPCRouter at the specified path.
func Mount(r *RPCRouter, path string, subRouter *RPCRouter) {
	r.router.Mount(path, subRouter.router)
}

// Group creates an anonymous grouped sub-router, applies middlewares, and invokes the callback for registration.
func Group(r *RPCRouter, fn func(r *RPCRouter), middlewares ...MiddlewareType) {
	r.router.Group(func(r chi.Router) {
		subRouter := &RPCRouter{router: chi.NewRouter()}
		AddGlobalMiddlewares(subRouter, middlewares...)
		fn(subRouter)
		r.Mount("/", subRouter.router)
	})
}

// MethodNotAllowed sets a custom handler for HTTP 405 Method Not Allowed responses.
func MethodNotAllowed(r *RPCRouter, fn http.HandlerFunc) {
	r.router.MethodNotAllowed(fn)
}

// NotFound sets a custom handler for HTTP 404 Not Found responses.
func NotFound(r *RPCRouter, fn http.HandlerFunc) {
	r.router.NotFound(fn)
}

// RegisterErrorHandler sets a global error handler and registers its type information for generation.
func RegisterErrorHandler[R any](handler ErrorHandlerType[R]) {
	// register handler type to generate ts types
	_, err := rpcType.RegisterHandler("ERROR_HANDLER", "/", handler)

	if err != nil {
		fmt.Fprintf(os.Stderr, "an error occurred while  registering error handler: \n %s", err)
		return
	}

	var anyHandler ErrorHandlerType[any] = func(r *http.Request, err error) HttpResponse[any] {
		resp := handler(r, err)
		return HttpResponse[any]{
			StatusCode: resp.StatusCode,
			Body:       resp.Body,
			Headers:    resp.Headers,
		}
	}

	errorHandler = anyHandler
}

// RegisterMethod registers a custom HTTP method with chi so it can be used in routing.
func RegisterMethod(method string) {
	chi.RegisterMethod(method)
}

// GenerateRpcTypes generates TypeScript types for all registered RPC handlers and writes them to a file.
// By default, it writes to "apiSchema.ts", but a custom path can be provided.
func GenerateRpcTypes(paths ...string) error {
	path := "apiSchema.ts"

	if len(paths) > 0 {
		path = paths[0]
	}

	typeString, err := rpcType.ConvertToTs()

	if err != nil {
		return err
	}

	err = os.WriteFile(path, []byte(typeString), 0644)
	if err != nil {
		return errors.New("failed to write types to file: " + err.Error())
	}

	return nil
}
