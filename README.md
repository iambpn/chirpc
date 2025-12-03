# chirpc

> End-to-end type-safe RPC toolkit for Go chi routers and TypeScript clients.

## Introduction

**chirpc** is a lightweight RPC framework that wraps the excellent [chi](https://github.com/go-chi/chi) router with a type-safe layer, ensuring Go backend handlers and TypeScript frontend clients stay perfectly synchronized. By leveraging Go generics and automatic TypeScript code generation, chirpc eliminates manual type definitions and prevents API contract mismatches.

Each handler you register automatically produces a strongly typed schema that is exported to TypeScript, enabling compile-time type checking across your entire stack. No more hand-written DTOs, no more runtime surprises.

## Features

- **Type-Safe Handlers**: Generic-based request handlers with typed request/response bodies, query parameters, and URL params
- **Automatic TypeScript Generation**: Converts Go structs to TypeScript interfaces with support for nested types, custom tags, and optional fields
- **Drop-in chi Wrapper**: Works seamlessly with existing chi routers and middleware
- **Flexible Error Handling**: Configurable error handlers with typed error payloads exposed to both Go and TypeScript
- **Router Composition**: Built-in support for grouping, mounting, and nesting routers with middleware scoping
- **Custom HTTP Methods**: Register and use custom HTTP verbs beyond the standard REST methods
- **TypeScript Client Integration**: Generated `ApiSchema` works seamlessly with [ts-axios-wrapper](https://www.npmjs.com/package/ts-axios-wrapper) for fully typed API calls
- **Struct Tag Support**: Customize TypeScript output using `tsKey`, `tsType`, `tsOptional`, and `tsOmit` tags

## Installation

### Prerequisites

- **Go**: Version 1.21 or higher
- **Node.js**: Version 18 or higher (for consuming generated TypeScript types)

### Backend (Go)

Add chirpc to your Go project:

```bash
go get github.com/iambpn/chirpc
```

### Frontend (TypeScript)

Install the TypeScript client wrapper in your frontend project:

```bash
npm install ts-axios-wrapper
```

## Usage

### Basic Server Setup

Here's a complete example of setting up a chirpc server:

```go
package main

import (
    "net/http"

    "github.com/go-chi/chi/v5/middleware"
    "github.com/iambpn/chirpc/v1"
)

const addr = ":8080"

// Define your response types
type ErrorResponse struct {
    Message string `json:"message"`
}

type HelloResponse struct {
    Message string `json:"message"`
}

// Define request body/query types
type RequestBody struct {
    Name string `json:"name"`
    Age  int    `json:"age" tsOptional:"true"`
}

func main() {
    // Create a new RPC router
    router := chirpc.NewRPCRouter()

    // Register global error handler
    chirpc.RegisterErrorHandler(router, ErrorHandler)

    // Add global middlewares
    chirpc.AddMiddlewares(router, middleware.Logger)

    // Register handlers with typed responses
    chirpc.AddHandler(router, chirpc.MethodGet, "/", HelloHandler).
        BodyType(RequestBody{}).
        QueryType(RequestBody{})

    chirpc.AddHandler(router, chirpc.MethodGet, "/{id}", GetByIdHandler)

    // Generate TypeScript schema (run this during development)
    if err := chirpc.GenerateRPCSchema(router); err != nil {
        panic("failed to generate apiSchema: " + err.Error())
    }

    // Start the server
    server := router.GetHttpServer()
    server.Addr = addr

    println("Starting server on", addr)
    if err := server.ListenAndServe(); err != nil {
        panic(err)
    }
}

// Error handler with typed error response
func ErrorHandler(r *http.Request, err *chirpc.ErrorResponse) *chirpc.HttpResponse[ErrorResponse] {
    return &chirpc.HttpResponse[ErrorResponse]{
        StatusCode: http.StatusInternalServerError,
        Body:       ErrorResponse{Message: "An error occurred"},
        Headers: map[string]string{
            "Content-Type": "application/json",
        },
    }
}

// Request handler with typed response
func HelloHandler(r *http.Request) (*chirpc.HttpResponse[HelloResponse], *chirpc.ErrorResponse) {
    return &chirpc.HttpResponse[HelloResponse]{
        StatusCode: http.StatusOK,
        Body:       HelloResponse{Message: "Hello, World!"},
        Headers: map[string]string{
            "Content-Type": "application/json",
        },
    }, nil
}

func GetByIdHandler(r *http.Request) (*chirpc.HttpResponse[HelloResponse], *chirpc.ErrorResponse) {
    // Access URL params via chi's context
    // id := chi.URLParam(r, "id")
    
    return &chirpc.HttpResponse[HelloResponse]{
        StatusCode: http.StatusOK,
        Body:       HelloResponse{Message: "Handler with URL params"},
    }, nil
}
```

### TypeScript Schema Generation

When you run `chirpc.GenerateRPCSchema(router)`, it generates an `apiSchema.ts` file:

```typescript
interface V1__ErrorResponse {
  statusCode?: number;
  errors?: string[];
  validationErrors?: { [key: string]: string[] };
}

export type ApiSchema = {
  ERROR_HANDLER: { "/": { response: V1__ErrorResponse } };
  GET: {
    "/": {
      query?: { name: string; age?: number };
      body: { name: string; age?: number };
      response: { message: string };
    };
    "/{id}": { 
      params: { id: string }; 
      response: { message: string } 
    };
  };
};
```

By default, the schema is written to `apiSchema.ts` in the project root. You can specify a custom path:

```go
chirpc.GenerateRPCSchema(router, "frontend/src/types/apiSchema.ts")
```

### Frontend Client Usage

Use the generated schema with `ts-axios-wrapper` for fully typed API calls:

```typescript
import { TypedAxios } from "ts-axios-wrapper";
import type { ApiSchema } from "./apiSchema.js";

// Create a typed API client
const api = new TypedAxios<ApiSchema>({ 
  baseURL: "http://localhost:8080" 
});

// Make type-safe API calls
// TypeScript will enforce correct method, path, and payload structure

// GET request with body and query parameters
const response = await api.GET("/", {
  body: {
    name: "John Doe",
    age: 25,
  },
  query: {
    name: "John Doe",
  }
});

// TypeScript knows the response type
console.log(response.body.message); // ✓ Type-safe!

// GET request with URL parameters
const userResponse = await api.GET("/{id}", {
  params: { id: "123" }
});

// Generic request method
api.request("GET", "/", {
  body: { name: "Jane Doe" },
});
```

### Advanced Usage

#### Router Grouping and Mounting

```go
// Create sub-router
subRouter := chirpc.NewRPCSubRouter()
chirpc.AddHandler(subRouter, chirpc.MethodGet, "/profile", ProfileHandler)
chirpc.AddHandler(subRouter, chirpc.MethodPost, "/settings", SettingsHandler)

// Mount at a prefix
chirpc.Mount(router, "/api/v1", subRouter)

// Use Route for scoped middleware
chirpc.Route(router, "/admin", func(r *chirpc.RPCRouter) {
    chirpc.AddMiddlewares(r, AdminAuthMiddleware)
    chirpc.AddHandler(r, chirpc.MethodGet, "/dashboard", DashboardHandler)
}, middleware.Logger)

// Group with middleware
chirpc.Group(router, func(r *chirpc.RPCRouter) {
    chirpc.AddHandler(r, chirpc.MethodGet, "/protected", ProtectedHandler)
}, AuthMiddleware)
```

#### Custom Struct Tags

Customize TypeScript generation with struct tags:

```go
type User struct {
    ID        int    `json:"id" tsKey:"userId"`           // Rename field in TypeScript
    Name      string `json:"name"`
    Age       int    `json:"age" tsOptional:"true"`       // Make optional in TypeScript
    Email     string `json:"email" tsType:"string"`       // Override TypeScript type
    Password  string `json:"password" tsOmit:"true"`      // Exclude from TypeScript
    CreatedAt time.Time `json:"created_at"`
}
```

Generated TypeScript:

```typescript
interface User {
  userId: number;
  name: string;
  age?: number;
  email: string;
  created_at: string;
  // password is omitted
}
```

#### Custom HTTP Methods

```go
// Register custom HTTP method
chirpc.RegisterMethod("CUSTOM")

// Use it in handlers
chirpc.AddHandler(router, "CUSTOM", "/custom-endpoint", CustomHandler)
```

#### Custom 404 and 405 Handlers

```go
chirpc.NotFound(router, func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotFound)
    w.Write([]byte("Custom 404 page"))
})

chirpc.MethodNotAllowed(router, func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusMethodNotAllowed)
    w.Write([]byte("Method not allowed"))
})
```

## Exposed APIs

### Router Creation

- **`NewRPCRouter() *RPCRouter`** - Create a new RPC router backed by chi.Mux
- **`NewRPCSubRouter() *RPCSubRouter`** - Create a sub-router for mounting or grouping

### Handler Registration

- **`AddHandler[R any](router, method, path, handler, ...middlewares) *BodyQueryParamType`** - Register a typed request handler and capture schema metadata. Returns a fluent builder for configuring body/query/param types
- **`RegisterErrorHandler[R any](router, handler)`** - Define a global typed error handler invoked when handlers return errors

### Middleware Management

- **`AddMiddlewares(router, ...middlewares)`** - Attach middlewares to the router that apply to all registered routes

### Router Composition

- **`Route(router, path, fn, ...middlewares)`** - Create a sub-route at the specified path with scoped middlewares
- **`Mount(router, path, subRouter)`** - Mount an existing RPCSubRouter at the specified path
- **`Group(router, fn, ...middlewares)`** - Create an anonymous grouped sub-router with scoped middlewares

### Request Type Configuration

Returned by `AddHandler`, these methods configure the expected request types:

- **`.BodyType(body any)`** - Specify the expected HTTP request body type
- **`.QueryType(query any)`** - Specify the expected URL query parameter type
- **`.Params(slugs []string)`** - Set expected URL path parameter slugs (auto-detected from path)

### Type Generation

- **`GenerateRPCSchema(router, path...string) error`** - Generate TypeScript types for all registered handlers and write to file (default: `apiSchema.ts`)

### HTTP Method Constants

- `MethodGet`, `MethodPost`, `MethodPut`, `MethodDelete`, `MethodPatch`
- `MethodOptions`, `MethodHead`, `MethodTrace`, `MethodConnect`

### Custom HTTP Verbs

- **`RegisterMethod(method string)`** - Register a custom HTTP method with chi for routing

### Fallback Handlers

- **`NotFound(router, handler)`** - Set custom handler for HTTP 404 Not Found responses
- **`MethodNotAllowed(router, handler)`** - Set custom handler for HTTP 405 Method Not Allowed responses

### Server Utilities

- **`GetHttpServer() *http.Server`** - Get the underlying http.Server instance
- **`ListenAndServe(addr string) error`** - Start the HTTP server on the specified address

### Core Types

- **`HttpResponse[T any]`** - Generic HTTP response with StatusCode, Body, and Headers
- **`ErrorResponse`** - Structured error response with StatusCode, Errors, and ValidationErrors
- **`RequestHandler[T any]`** - Handler function type that processes requests and returns typed responses
- **`ErrorHandlerType[T any]`** - Error handler function type
- **`MiddlewareType`** - Type alias for chi middleware functions

## Contributing

Contributions are welcome! Here's how you can help:

1. **Open an Issue**: Before submitting a pull request, open an issue describing the improvement, bug fix, or feature you'd like to work on
2. **Run Tests**: Ensure all tests pass with `go test ./...`
3. **Add Coverage**: Include tests for new functionality or bug fixes
4. **Follow Conventions**: 
   - Use Go formatting conventions (`gofmt`, `go vet`)
   - Keep PRs focused on a single feature or fix
   - Write clear commit messages
5. **Update Documentation**: Update README or code comments if your changes affect the public API

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./v1
go test ./internal/tsGen
```

## License

MIT License © 2025 Bipin Maharjan

See [LICENSE](LICENSE) for full details.
