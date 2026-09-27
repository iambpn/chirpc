package chirpc

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/iambpn/chirpc/internal/rpc"
)

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

// RPCSubRouter is the same type as RPCRouter. Any router can be mounted in another.
//
// Deprecated: Use RPCRouter.
type RPCSubRouter = RPCRouter

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

// NewRPCRouter creates a new RPCRouter with a fresh chi.Mux instance. The router can
// serve requests directly or be mounted in another router with Mount.
func NewRPCRouter() *RPCRouter {
	router := newRPCRouter(chi.NewRouter())

	// This type describes the ErrorResponse body sent when no error handler is registered.
	// It is used only when this router is the root of the generated schema.
	router.routerTypes.SetDefaultErrorHandler(ErrorHandlerType[ErrorResponse](nil))
	return router
}

// NewRPCSubRouter creates a new router for mounting.
//
// Deprecated: Use NewRPCRouter. Any router can be mounted in another.
func NewRPCSubRouter() *RPCRouter {
	return NewRPCRouter()
}

// AddMiddlewares attaches the provided middlewares to the router.
func AddMiddlewares(r *RPCRouter, middlewares ...MiddlewareType) {
	r.router.Use(middlewares...)
}

// AddHandler registers an RPC handler for the given HTTP method and path, applies optional middlewares,
// records its schema for TypeScript generation, and returns a ParamsBuilder to add extra path parameters.
// Path params are read from the full route URL when the schema is generated. The schema has no request
// body or query type. Use AddTypedHandler for routes that take them.
// It panics when the handler cannot be registered, like chi does for invalid routes.
func AddHandler[R any](r *RPCRouter, method HttpMethods, path string, handler RequestHandler[R], middlewares ...MiddlewareType) *rpc.ParamsBuilder {
	// register handler type to generate ts types
	schema, err := r.routerTypes.RegisterHandler(method, path, handler)
	if err != nil {
		panic(fmt.Sprintf("chirpc: could not register the handler for %s %s: %s.", method, path, err))
	}

	paramsBuilder := rpc.NewParamsBuilder(schema)

	r.router.With(middlewares...).Method(method, path, handler.ServeHTTPWithErrorHandler(nil))

	return paramsBuilder
}

// Route creates a sub-route at the specified path, applies middlewares to it, and invokes the callback to populate it.
func Route(parent *RPCRouter, path string, fn func(r *RPCRouter), middlewares ...MiddlewareType) {
	router := newRPCRouter(chi.NewRouter())
	AddMiddlewares(router, middlewares...)
	fn(router)

	parent.router.Mount(path, router.router)
	parent.routerTypes.Mount(path, router.routerTypes)
}

// Mount mounts an existing router at the specified path.
// Handlers added to the mounted router later are also served and included in the schema.
func Mount(parent *RPCRouter, path string, subRouter *RPCRouter) {
	if subRouter == nil {
		return
	}

	parent.router.Mount(path, subRouter.router)
	parent.routerTypes.Mount(path, subRouter.routerTypes)
}

// Group creates an anonymous grouped sub-router, applies middlewares, and invokes the callback for registration.
func Group(parent *RPCRouter, fn func(r *RPCRouter), middlewares ...MiddlewareType) {
	parent.router.Group(func(chiR chi.Router) {
		// chi passes an inline *chi.Mux that shares the parent's routes.
		router := newRPCRouter(chiR.(*chi.Mux))
		AddMiddlewares(router, middlewares...)
		fn(router)

		parent.routerTypes.Mount("", router.routerTypes)
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

// RegisterErrorHandler sets the error handler for this router and its child routers,
// and registers its type information for generation. It applies to handlers added
// before and after the call. A child router can register its own handler, but it must
// return the same type as the root router's handler, because ApiSchema has one error type.
// It panics when the error handler cannot be registered.
func RegisterErrorHandler[R any](router *RPCRouter, handler ErrorHandlerType[R]) {

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

// RegisterTSType makes the generated TypeScript use tsType wherever the Go type T appears,
// for example chirpc.RegisterTSType[uuid.UUID](router, "string"). Use it for types with
// custom JSON encoding. A type registered on a mounted router applies to the whole schema.
// It panics when tsType is empty.
func RegisterTSType[T any](r *RPCRouter, tsType string) {
	if strings.TrimSpace(tsType) == "" {
		panic("chirpc: RegisterTSType needs a TypeScript type, but got an empty string.")
	}
	r.routerTypes.SetTypeOverride(reflect.TypeFor[T](), tsType)
}

// RegisterMethod registers a custom HTTP method with chi so it can be used in routing.
func RegisterMethod(method string) {
	chi.RegisterMethod(method)
}
