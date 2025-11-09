package chirpc

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/iambpn/chirpc/internal/rpcType"
)

var errorHandler ErrorHandlerType[any]

type RPCRouter struct {
	router *chi.Mux
}

func (r *RPCRouter) GetHttpServer() *http.Server {
	return &http.Server{
		Handler: r.router,
	}
}

func (r *RPCRouter) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, r.router)
}

func NewRPCRouter() *RPCRouter {
	return &RPCRouter{
		router: chi.NewRouter(),
	}
}

func AddGlobalMiddlewares(r *RPCRouter, middlewares ...MiddlewareType) {
	r.router.Use(middlewares...)
}

func AddHandler[R any](r *RPCRouter, method HttpMethods, path string, handler RequestHandler[R], middlewares ...MiddlewareType) *rpcType.BodyQueryParamType {
	bodyQueryParam := &rpcType.BodyQueryParamType{}

	// register handler type to generate ts types
	schema, err := rpcType.RegisterHandler(string(method), path, handler)

	if err != nil {
		fmt.Println("Error while registering handler:", err)
		return bodyQueryParam
	}

	bodyQueryParam.Schema = schema

	// get slugs from path
	slugs := parseURLSlug(path)
	bodyQueryParam.Params(slugs)

	r.router.With(middlewares...).Method(string(method), path, handler.ServeHTTPWithErrorHandler(errorHandler))

	return bodyQueryParam
}

func Route(r *RPCRouter, path string, fn func(r *RPCRouter), middlewares ...MiddlewareType) {
	r.router.Route(path, func(r chi.Router) {
		subRouter := &RPCRouter{router: chi.NewRouter()}
		AddGlobalMiddlewares(subRouter, middlewares...)
		fn(subRouter)
		r.Mount("/", subRouter.router)
	})
}

func Mount(r *RPCRouter, path string, subRouter *RPCRouter) {
	r.router.Mount(path, subRouter.router)
}

func Group(r *RPCRouter, fn func(r *RPCRouter), middlewares ...MiddlewareType) {
	r.router.Group(func(r chi.Router) {
		subRouter := &RPCRouter{router: chi.NewRouter()}
		AddGlobalMiddlewares(subRouter, middlewares...)
		fn(subRouter)
		r.Mount("/", subRouter.router)
	})
}

func MethodNotAllowed(r *RPCRouter, fn http.HandlerFunc) {
	r.router.MethodNotAllowed(fn)
}

func NotFound(r *RPCRouter, fn http.HandlerFunc) {
	r.router.NotFound(fn)
}

func RegisterErrorHandler[R any](handler ErrorHandlerType[R]) {
	// register handler type to generate ts types
	_, err := rpcType.RegisterHandler("ERROR_HANDLER", "/", handler)

	if err != nil {
		fmt.Println("Error registering error handler:", err)
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

func RegisterMethod(method string) {
	chi.RegisterMethod(method)
}

func BuildRpcTypes(paths ...string) error {
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
