package chirpc

import (
	"context"
	"fmt"
	"net/http"

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

	// errorHandler handles errors from this router's handlers and from its child routers
	// that have no error handler of their own. It is read when a request is served.
	errorHandler ErrorHandlerType[any]
}

// isRpcRouter implements the IsRPCRouter interface for RPCRouter.
func (r *RPCRouter) isRpcRouter() bool {
	return true
}

// RPCSubRouter represents a sub-router within the chirpc routing system.
// Its handlers are served and added to the schema at the path given to Mount.
// This type is used for modular route grouping and mounting within the main router.
type RPCSubRouter struct {
	rpcRouter *RPCRouter
}

// isRpcRouter implements the IsRPCRouter interface for RPCSubRouter.
func (r *RPCSubRouter) isRpcRouter() bool {
	return true
}

// errorHandlerKey is the request context key for the error handler of the router serving the request.
type errorHandlerKey struct{}

// errorHandlerFromContext returns the error handler stored by the nearest router that has one, or nil.
func errorHandlerFromContext(ctx context.Context) ErrorHandlerType[any] {
	handler, _ := ctx.Value(errorHandlerKey{}).(ErrorHandlerType[any])
	return handler
}

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

// ServeHTTP serves the request with the underlying chi router, so an RPCRouter can be
// used anywhere an http.Handler is expected.
func (r *RPCRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.router.ServeHTTP(w, req)
}

// RPCSchemas returns the handler schemas of this router and its child routers.
// The tsgen package uses it to generate TypeScript.
func (r *RPCRouter) RPCSchemas() *rpc.RouterRpcSchemas {
	return r.routerTypes
}

// rpcRouterOf returns the RPCRouter behind r. It panics for any other router type.
func rpcRouterOf(r IsRPCRouter, caller string) *RPCRouter {
	switch rt := r.(type) {
	case *RPCRouter:
		return rt
	case *RPCSubRouter:
		return rt.rpcRouter
	default:
		panic(fmt.Sprintf("chirpc: %s needs an *RPCRouter or *RPCSubRouter, but got %T.", caller, r))
	}
}

// newRPCRouter wraps mux in an RPCRouter with no error handler.
func newRPCRouter(mux *chi.Mux) *RPCRouter {
	router := &RPCRouter{
		router:      mux,
		routerTypes: rpc.NewRouterRpcSchemas(),
	}

	// Routers run from the outside in, so the innermost router with an error handler sets the final value.
	mux.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if router.errorHandler != nil {
				req = req.WithContext(context.WithValue(req.Context(), errorHandlerKey{}, router.errorHandler))
			}
			next.ServeHTTP(w, req)
		})
	})

	return router
}

// NewRPCRouter creates a new RPCRouter with a fresh chi.Mux instance and registers
// a default error handler for TypeScript type generation.
func NewRPCRouter() *RPCRouter {
	router := newRPCRouter(chi.NewRouter())

	// This type describes the ErrorResponse body sent when no error handler is registered.
	router.routerTypes.RegisterErrorHandler(ErrorHandlerType[ErrorResponse](nil))
	return router
}

// NewRPCSubRouter creates a new RPCSubRouter with an empty route collection.
func NewRPCSubRouter() *RPCSubRouter {
	return &RPCSubRouter{
		rpcRouter: newRPCRouter(chi.NewRouter()),
	}
}

// AddMiddlewares attaches the provided middlewares to the router.
func AddMiddlewares(r IsRPCRouter, middlewares ...MiddlewareType) {
	rpcRouterOf(r, "AddMiddlewares").router.Use(middlewares...)
}

// AddHandler registers an RPC handler for the given HTTP method and path, applies optional middlewares,
// records its schema for TypeScript generation, and returns a BodyQueryParamType to allow parameter configuration.
// Path params are read from the full route URL when the schema is generated.
// It panics when the handler cannot be registered, like chi does for invalid routes.
func AddHandler[R any](r IsRPCRouter, method HttpMethods, path string, handler RequestHandler[R], middlewares ...MiddlewareType) *rpc.BodyQueryParamType {
	rpcRouter := rpcRouterOf(r, "AddHandler")

	// register handler type to generate ts types
	schema, err := rpcRouter.routerTypes.RegisterHandler(method, path, handler)
	if err != nil {
		panic(fmt.Sprintf("chirpc: could not register the handler for %s %s: %s.", method, path, err))
	}

	bodyQueryParam := rpc.NewBodyQueryParamType(schema)

	rpcRouter.router.With(middlewares...).Method(method, path, handler.ServeHTTPWithErrorHandler(nil))

	return bodyQueryParam
}

// Route creates a sub-route at the specified path, applies middlewares to it, and invokes the callback to populate it.
func Route(r IsRPCRouter, path string, fn func(r *RPCRouter), middlewares ...MiddlewareType) {
	parent := rpcRouterOf(r, "Route")
	router := newRPCRouter(chi.NewRouter())
	AddMiddlewares(router, middlewares...)
	fn(router)

	parent.router.Mount(path, router.router)
	parent.routerTypes.Mount(path, router.routerTypes)
}

// Mount mounts an existing RPCSubRouter at the specified path.
// Handlers added to the sub-router after mounting are also served and included in the schema.
func Mount(r IsRPCRouter, path string, subRouter *RPCSubRouter) {
	if subRouter == nil {
		return
	}

	parent := rpcRouterOf(r, "Mount")
	parent.router.Mount(path, subRouter.rpcRouter.router)
	parent.routerTypes.Mount(path, subRouter.rpcRouter.routerTypes)
}

// Group creates an anonymous grouped sub-router, applies middlewares, and invokes the callback for registration.
func Group(r IsRPCRouter, fn func(r *RPCRouter), middlewares ...MiddlewareType) {
	parent := rpcRouterOf(r, "Group")
	parent.router.Group(func(chiR chi.Router) {
		// chi passes an inline *chi.Mux that shares the parent's routes.
		router := newRPCRouter(chiR.(*chi.Mux))
		AddMiddlewares(router, middlewares...)
		fn(router)

		parent.routerTypes.Mount("", router.routerTypes)
	})
}

// MethodNotAllowed sets a custom handler for HTTP 405 Method Not Allowed responses.
func MethodNotAllowed(r IsRPCRouter, fn http.HandlerFunc) {
	rpcRouterOf(r, "MethodNotAllowed").router.MethodNotAllowed(fn)
}

// NotFound sets a custom handler for HTTP 404 Not Found responses.
func NotFound(r IsRPCRouter, fn http.HandlerFunc) {
	rpcRouterOf(r, "NotFound").router.NotFound(fn)
}

// RegisterErrorHandler sets the error handler for this router or sub-router and its child routers,
// and registers its type information for generation. It applies to handlers added
// before and after the call. A child router can register its own handler, but it must
// return the same type as the root router's handler, because ApiSchema has one error type.
// It panics when the error handler cannot be registered.
func RegisterErrorHandler[R any](r IsRPCRouter, handler ErrorHandlerType[R]) {
	router := rpcRouterOf(r, "RegisterErrorHandler")

	// register handler type to generate ts types
	if err := router.routerTypes.RegisterErrorHandler(handler); err != nil {
		panic(fmt.Sprintf("chirpc: could not register the error handler: %s.", err))
	}

	router.errorHandler = func(r *http.Request, err *ErrorResponse) *HttpResponse[any] {
		resp := handler(r, err)
		if resp == nil {
			return nil
		}
		return &HttpResponse[any]{
			StatusCode: resp.StatusCode,
			Body:       resp.Body,
			Headers:    resp.Headers,
		}
	}
}

// RegisterMethod registers a custom HTTP method with chi so it can be used in routing.
func RegisterMethod(method string) {
	chi.RegisterMethod(method)
}
