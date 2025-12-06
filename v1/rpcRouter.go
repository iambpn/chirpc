package chirpc

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/iambpn/chirpc/internal/rpc"
)

// IsRPCRouter is an interface used to identify types that act as RPC routers within the chirpc package.
// It provides a single method isRpcRouter for type assertion and internal routing logic.
type IsRPCRouter interface {
	isRpcRouter() bool
}

// RPCRouter provides a thin wrapper around a chi.Mux router and exposes
// helper methods for registering RPC-style handlers, mounting sub-routers,
// and producing an http.Server. It centralizes middleware registration and
// error handler wiring for the chirpc package.
type RPCRouter struct {
	router      *chi.Mux
	routerTypes *rpc.RouterRpcSchemas
	prefixPath  string
}

// isRpcRouter implements the IsRPCRouter interface for RPCRouter.
func (r *RPCRouter) isRpcRouter() bool {
	return true
}

// RPCSubRouter represents a sub-router within the chirpc routing system.
// It holds a reference to its parent RPCRouter and a slice of HandlerSchema objects
// that describe the registered RPC routes for TypeScript type generation and internal routing.
// This type is used for modular route grouping and mounting within the main router.
type RPCSubRouter struct {
	rpcRouter *RPCRouter
	subRoutes []*rpc.HandlerSchema
}

// isRpcRouter implements the IsRPCRouter interface for RPCSubRouter.
func (r *RPCSubRouter) isRpcRouter() bool {
	return true
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

// NewRPCRouter creates a new RPCRouter with a fresh chi.Mux instance and registers
// a default error handler for TypeScript type generation.
func NewRPCRouter() *RPCRouter {
	router := &RPCRouter{
		router:      chi.NewRouter(),
		routerTypes: rpc.NewRouterRpcSchemas(),
	}

	// register default error handler
	router.routerTypes.RegisterHandler("ERROR_HANDLER", "/", ErrorHandlerType[ErrorResponse](nil))
	return router
}

// NewRPCSubRouter creates a new RPCSubRouter with an empty route collection.
func NewRPCSubRouter() *RPCSubRouter {
	router := NewRPCRouter()
	return &RPCSubRouter{
		rpcRouter: router,
		subRoutes: []*rpc.HandlerSchema{},
	}
}

// AddMiddlewares attaches the provided middlewares to the root router.
func AddMiddlewares(r *RPCRouter, middlewares ...MiddlewareType) {
	r.router.Use(middlewares...)
}

// AddHandler registers an RPC handler for the given HTTP method and path, applies optional middlewares,
// records its schema for TypeScript generation, and returns a BodyQueryParamType to allow parameter configuration.
func AddHandler[R any](r IsRPCRouter, method HttpMethods, path string, handler RequestHandler[R], middlewares ...MiddlewareType) *rpc.BodyQueryParamType {
	bodyQueryParam := rpc.NewBodyQueryParamType(nil)

	// get specific rpc router
	var rpcRouter *RPCRouter = nil
	var rpcSubRouter *RPCSubRouter = nil
	var mergedPath string

	switch rt := r.(type) {
	case *RPCRouter:
		rpcRouter = rt
		mergedPath = mergePaths(rt.prefixPath, path)
	case *RPCSubRouter:
		rpcSubRouter = rt
		mergedPath = mergePaths(rt.rpcRouter.prefixPath, path)
		rpcRouter = rt.rpcRouter
	default:
		fmt.Fprintf(os.Stderr, "invalid router type provided to AddHandler. Must be oneof RPCRouter or RPCSubRouter type\n")
		return bodyQueryParam
	}

	var schema *rpc.HandlerSchema
	var err error

	// register handler type to generate ts types
	if rpcSubRouter != nil {
		schema, err = rpc.BuildGoToTsSchema(string(method), mergedPath, handler)
		rpcSubRouter.subRoutes = append(rpcSubRouter.subRoutes, schema)
	} else {
		schema, err = rpcRouter.routerTypes.RegisterHandler(string(method), mergedPath, handler)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "an error occurred while registering handler: \n %s", err)
		return bodyQueryParam
	}

	bodyQueryParam.Schema = schema

	// get slugs from path
	slugs := parseURLSlug(path)
	bodyQueryParam.Params(slugs)

	rpcRouter.router.With(middlewares...).Method(string(method), path, handler.ServeHTTPWithErrorHandler(errorHandler))

	return bodyQueryParam
}

// Route creates a sub-route at the specified path, applies middlewares to it, and invokes the callback to populate it.
func Route(r *RPCRouter, path string, fn func(r *RPCRouter), middlewares ...MiddlewareType) {
	r.router.Route(path, func(chiR chi.Router) {
		router := NewRPCRouter()
		router.prefixPath = path

		AddMiddlewares(router, middlewares...)
		fn(router)

		// mount sub-router at path
		chiR.Mount("/", router.router)

		// register sub-router schemas to parent router
		r.routerTypes.RegisterHandlerFrom(router.routerTypes)
	})
}

// Mount mounts an existing RPCSubRouter at the specified path.
func Mount(r *RPCRouter, path string, subRouter *RPCSubRouter) {
	if subRouter == nil {
		return
	}

	for _, schema := range subRouter.subRoutes {
		// adjust path to include mount point
		schema.SetUrl(mergePaths(mergePaths(r.prefixPath, path), schema.URL()))
	}

	r.routerTypes.RegisterHandlers(subRouter.subRoutes)

	r.router.Mount(path, subRouter.rpcRouter.router)
}

// Group creates an anonymous grouped sub-router, applies middlewares, and invokes the callback for registration.
func Group(r *RPCRouter, fn func(r *RPCRouter), middlewares ...MiddlewareType) {
	r.router.Group(func(chiR chi.Router) {
		router := NewRPCRouter()
		AddMiddlewares(router, middlewares...)
		fn(router)

		// mount sub-router at path
		chiR.Mount("/", router.router)

		// register sub-router schemas to parent router
		r.routerTypes.RegisterHandlerFrom(router.routerTypes)
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
// This handler must be registered before any handlers are registered.
func RegisterErrorHandler[R any](r *RPCRouter, handler ErrorHandlerType[R]) {
	// register handler type to generate ts types
	_, err := r.routerTypes.RegisterHandler("ERROR_HANDLER", "/", handler)

	if err != nil {
		fmt.Fprintf(os.Stderr, "an error occurred while  registering error handler: \n %s", err)
		return
	}

	var anyHandler ErrorHandlerType[any] = func(r *http.Request, err *ErrorResponse) *HttpResponse[any] {
		resp := handler(r, err)
		return &HttpResponse[any]{
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

// GenerateRPCSchema generates TypeScript types for all registered RPC handlers and writes them to a file.
// By default, it writes to "apiSchema.ts", but a custom path can be provided.
func GenerateRPCSchema(r *RPCRouter, paths ...string) error {
	path := "apiSchema.ts"

	if len(paths) > 0 {
		path = paths[0]
	}

	typeString, err := r.routerTypes.ConvertToTs()

	if err != nil {
		return err
	}

	err = os.WriteFile(path, []byte(typeString), 0644)
	if err != nil {
		return errors.New("failed to write types to file: " + err.Error())
	}

	fmt.Printf("Successfully generated RPC schema at %s\n", path)
	return nil
}
