package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/iambpn/chirpc/v1"
	"github.com/iambpn/chirpc/v1/tsgen"
)

const addr = ":8080"

type ErrorResponse struct {
	Message string `json:"message"`
}

type body struct {
	Name string `json:"name"`
	Age  int    `json:"age" tsOptional:"true"`
}

type createUser struct {
	Name string `json:"name"`
}

type user struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (b *body) Validate() error {
	return errors.New("test error")
}

func main() {
	startServer()
}

func startServer() {
	rpcRouter := chirpc.NewRPCRouter()

	chirpc.RegisterErrorHandler(rpcRouter, ErrorHandler)

	chirpc.AddMiddlewares(rpcRouter, middleware.Logger)
	chirpc.AddTypedHandler(rpcRouter, chirpc.MethodGet, "/", GreetHandler)
	chirpc.AddHandler(rpcRouter, chirpc.MethodGet, "/error", GetErrorHandler)
	chirpc.AddHandler(rpcRouter, chirpc.MethodGet, "/{test}", GetHandler)
	chirpc.AddTypedHandler(rpcRouter, chirpc.MethodPost, "/users", CreateUserHandler)

	// This example generates the schema at startup for convenience. A production server
	// should generate it in a separate program, so it does not link the TypeScript compiler.
	err := tsgen.GenerateRPCSchema(rpcRouter)

	if err != nil {
		fmt.Println("Error generating types:", err.Error())
		return
	}
	fmt.Println("Generated the RPC schema at apiSchema.ts.")

	server := rpcRouter.GetHttpServer()
	server.Addr = addr

	println("Starting server on", addr)
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

func ErrorHandler(r *http.Request, err *chirpc.ErrorResponse) *chirpc.HttpResponse[ErrorResponse] {
	// StatusCode is left unset, so the status code of the ErrorResponse is used.
	return &chirpc.HttpResponse[ErrorResponse]{
		Body: ErrorResponse{Message: strings.Join(err.Errors, ", ")},
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

func GetHandler(r *http.Request) (*chirpc.HttpResponse[map[string]string], *chirpc.ErrorResponse) {
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

// GreetHandler shows a typed handler that reads both a JSON body and query parameters.
func GreetHandler(req *chirpc.Request[body, body]) (*chirpc.HttpResponse[map[string]string], error) {
	return &chirpc.HttpResponse[map[string]string]{
		Body: map[string]string{
			"message": "Hello, " + req.Body.Name + " from " + req.Query.Name + "!",
		},
	}, nil
}

func GetErrorHandler(r *http.Request) (*chirpc.HttpResponse[map[string]string], *chirpc.ErrorResponse) {
	return nil, &chirpc.ErrorResponse{
		Errors: []string{"this is a test error"},
	}
}

// CreateUserHandler shows a typed handler. chirpc decodes the JSON body into createUser
// and rejects a body without "name" before the handler runs. The handler only checks
// rules that chirpc cannot know, such as a name that is present but empty.
func CreateUserHandler(req *chirpc.Request[createUser, chirpc.NoQuery]) (*chirpc.HttpResponse[user], error) {
	if req.Body.Name == "" {
		return nil, &chirpc.ErrorResponse{
			StatusCode:       http.StatusBadRequest,
			Errors:           []string{"The name must not be empty."},
			ValidationErrors: map[string][]string{"name": {"This field must not be empty."}},
		}
	}

	return &chirpc.HttpResponse[user]{
		StatusCode: http.StatusCreated,
		Body:       user{ID: "1", Name: req.Body.Name},
	}, nil
}
