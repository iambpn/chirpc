package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/iambpn/chirpc/v1"
)

const addr = ":8080"

type ErrorResponse struct {
	Message string `json:"message"`
}

type body struct {
	Name string `json:"name"`
	Age  int    `json:"age" tsOptional:"true"`
}

func (b *body) Validate() error {
	return errors.New("test error")
}

func main() {
	rpcRouter := chirpc.NewRPCRouter()

	chirpc.AddGlobalMiddlewares(rpcRouter, middleware.Logger)
	chirpc.AddHandler(rpcRouter, chirpc.GET, "/", GetHandler).BodyType(body{}).QueryType(body{})
	chirpc.AddHandler(rpcRouter, chirpc.GET, "/{test}", GetHandler)

	chirpc.RegisterErrorHandler(ErrorHandler)

	err := chirpc.BuildRpcTypes()

	if err != nil {
		fmt.Println("Error generating types:", err.Error())
		return
	}

	server := rpcRouter.GetHttpServer()
	server.Addr = addr

	println("Starting server on", addr)
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

func ErrorHandler(r *http.Request, err error) chirpc.HttpResponse[ErrorResponse] {
	return chirpc.HttpResponse[ErrorResponse]{
		StatusCode: http.StatusInternalServerError,
		Body:       ErrorResponse{Message: err.Error()},
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

func GetHandler(r *http.Request) (*chirpc.HttpResponse[map[string]string], error) {
	return &chirpc.HttpResponse[map[string]string]{
		StatusCode: http.StatusOK,
		Body: map[string]string{
			"message": "Hello, World!",
		},
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}
