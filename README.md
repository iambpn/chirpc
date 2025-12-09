# chirpc

> End-to-end type-safe RPC toolkit for Go chi routers and TypeScript clients

## Introduction

**chirpc** is a lightweight, type-safe RPC framework that wraps the powerful [chi](https://github.com/go-chi/chi) router with automatic TypeScript code generation. It bridges the gap between Go backend handlers and TypeScript frontend clients, ensuring perfect type synchronization across your entire stack.

By leveraging Go generics and automatic schema extraction, chirpc eliminates manual type definitions, prevents API contract mismatches, and provides compile-time type safety from server to client. Each handler you register automatically produces a strongly typed schema that is exported to TypeScript, enabling IDE autocomplete, type checking, and refactoring support across your full-stack application.

No more hand-written DTOs, no more runtime surprises, no more API documentation drift.

## Features

- **🔒 End-to-End Type Safety**: Generic-based request handlers with typed request/response bodies, query parameters, and URL params that automatically sync with TypeScript clients
- **🚀 Automatic TypeScript Generation**: Converts Go structs to TypeScript interfaces with support for nested types, anonymous structs, pointers, maps, arrays, and custom struct tags
- **🔌 Drop-in chi Wrapper**: Seamlessly integrates with existing chi routers, middleware, and ecosystem
- **⚡ Zero Runtime Overhead**: Type generation happens at build time; production code runs as fast as standard chi routers
- **🛡️ Flexible Error Handling**: Configurable typed error handlers with structured error payloads exposed to both Go and TypeScript
- **🎯 Router Composition**: Full support for grouping, mounting, and nesting routers with middleware scoping
- **🔧 Custom HTTP Methods**: Register and use custom HTTP verbs beyond standard REST methods
- **📦 TypeScript Client Ready**: Generated `ApiSchema` works seamlessly with [ts-axios-wrapper](https://www.npmjs.com/package/ts-axios-wrapper) for fully typed API calls
- **🏷️ Struct Tag Support**: Customize TypeScript output using `tsKey`, `tsType`, `tsOptional`, and `tsOmit` tags
- **📝 Path Parameter Extraction**: Automatic URL parameter detection and type generation from chi-style path patterns (`/{id}`)
- **🔄 Nested Type Support**: Handles complex nested structs, pointers, maps, slices, and anonymous inline structs

## Installation

### Prerequisites

- **Go**: Version 1.21 or higher (generics support required)
- **Node.js**: Version 18 or higher (for consuming generated TypeScript types)
- **TypeScript**: Version 4.5 or higher (recommended)

### Backend (Go)

Add chirpc to your Go project:

```bash
go get github.com/iambpn/chirpc
```

### Frontend (TypeScript)

Install the TypeScript client wrapper for making type-safe API calls:

```bash
npm install ts-axios-wrapper
# or
yarn add ts-axios-wrapper
# or
pnpm add ts-axios-wrapper
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
      response: { message: string };
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
  baseURL: "http://localhost:8080",
});

// Make type-safe API calls
// TypeScript will enforce correct method, path, and payload structure

// GET request with body and query parameters
const response = await api.GET("/", {
  body: {
    name: "John Doe",
    age: 25, // age is optional due to tsOptional tag
  },
  query: {
    name: "John Doe",
  },
});

// TypeScript knows the response type automatically
console.log(response.body.message); // ✓ Type-safe!

// GET request with URL parameters
const userResponse = await api.GET("/{id}", {
  params: { id: "123" }, // TypeScript enforces the 'id' parameter
});

// Generic request method (alternative syntax)
const altResponse = await api.request("GET", "/", {
  body: { name: "Jane Doe" },
});

// Error responses are also typed
try {
  await api.GET("/error");
} catch (error) {
  // error.response.data matches ErrorResponse type
  console.error(error.response.data.message);
}
```

**Key Benefits:**

- **Autocomplete**: Your IDE suggests available endpoints, methods, and required fields
- **Type Safety**: Compile-time errors if you use wrong parameter types or miss required fields
- **Refactoring**: Changing Go types automatically shows TypeScript errors that need fixing
- **Self-Documenting**: No need for separate API documentation - types are the docs

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

Customize TypeScript generation with struct tags to control how Go types are exported:

```go
type User struct {
    ID        int    `json:"id" tsKey:"userId"`           // Rename field in TypeScript
    Name      string `json:"name"`                        // Standard mapping
    Age       int    `json:"age" tsOptional:"true"`       // Make optional in TypeScript
    Email     string `json:"email" tsType:"string"`       // Override TypeScript type
    Password  string `json:"password" tsOmit:"true"`      // Exclude from TypeScript
    CreatedAt time.Time `json:"created_at"`               // Mapped to string in TypeScript
    Metadata  map[string]interface{} `json:"metadata"`   // Mapped to { [key: string]: any }
    Tags      []string `json:"tags"`                      // Mapped to (string)[]
    Profile   *Profile `json:"profile"`                   // Mapped to Profile | null
}

type Profile struct {
    Bio       string `json:"bio"`
    AvatarURL string `json:"avatar_url"`
}
```

Generated TypeScript:

```typescript
interface Profile {
  bio: string;
  avatar_url: string;
}

interface User {
  userId: number; // Renamed via tsKey
  name: string;
  age?: number; // Optional via tsOptional
  email: string;
  created_at: string; // time.Time becomes string
  metadata: { [key: string]: any };
  tags: string[];
  profile: Profile | null; // Pointer becomes nullable
  // password is omitted via tsOmit
}
```

**Available Struct Tags:**

- `tsKey:"newName"` - Rename field in TypeScript interface
- `tsType:"customType"` - Override the generated TypeScript type
- `tsOptional:"true"` - Make the field optional (add `?` in TypeScript)
- `tsOmit:"true"` - Exclude the field from TypeScript generation

**Type Conversion Rules:**

- `bool` → `boolean`
- `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64` → `number`
- `string` → `string`
- `[]T` or `[N]T` → `(T)[]`
- `map[K]V` → `{ [key: K]: V }`
- `*T` → `T | null`
- `struct` → separate interface
- Anonymous struct → inline object type
- `time.Time` → `string`
- Unexported fields → ignored
- Anonymous fields → ignored

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

## Examples

- See the [Chirpc Examples](https://github.com/iambpn/Chirpc-examples) repository for complete server and client implementations.

## Exposed APIs

### Router Creation

- **`NewRPCRouter() *RPCRouter`** Create a new RPC router backed by chi.Mux. This is the main entry point for creating a chirpc application.

- **`NewRPCSubRouter() *RPCSubRouter`** Create a sub-router for mounting or grouping. Used with `Mount()` to organize routes.

### Handler Registration

- **`AddHandler[R any](router, method, path, handler, ...middlewares) *BodyQueryParamType`** Register a typed request handler and capture schema metadata for TypeScript generation. Returns a fluent builder for configuring body/query/param types.

  **Parameters:**

  - `router`: Either `*RPCRouter` or `*RPCSubRouter`
  - `method`: HTTP method (use constants like `MethodGet`, `MethodPost`, etc.)
  - `path`: URL path pattern (supports chi-style params like `/{id}`)
  - `handler`: `RequestHandler[R]` function that processes requests
  - `middlewares`: Optional middleware functions to apply to this specific handler

  **Example:**

  ```go
  chirpc.AddHandler(router, chirpc.MethodGet, "/users/{id}", GetUserHandler).
      QueryType(QueryParams{})
  ```

- **`RegisterErrorHandler[R any](router, handler)`** Define a global typed error handler invoked when handlers return `*ErrorResponse`. Must be registered before any route handlers.

  **Example:**

  ```go
  chirpc.RegisterErrorHandler(router, func(r *http.Request, err *ErrorResponse) *HttpResponse[ErrorResponse] {
      return &HttpResponse[ErrorResponse]{
          StatusCode: http.StatusInternalServerError,
          Body: ErrorResponse{Message: err.Errors[0]},
      }
  })
  ```

### Middleware Management

- **`AddMiddlewares(router, ...middlewares)`** Attach middlewares to the router that apply to all registered routes. Supports standard chi middleware.

  **Example:**

  ```go
  chirpc.AddMiddlewares(router, middleware.Logger, middleware.Recoverer)
  ```

### Router Composition

- **`Route(router, path, fn, ...middlewares)`** Create a sub-route at the specified path with scoped middlewares. The callback function receives a new router instance.

  **Example:**

  ```go
  chirpc.Route(router, "/api/v1", func(r *RPCRouter) {
      chirpc.AddHandler(r, chirpc.MethodGet, "/users", ListUsersHandler)
  }, middleware.Logger)
  ```

- **`Mount(router, path, subRouter)`** Mount an existing RPCSubRouter at the specified path. All routes in the sub-router are prefixed with the mount path.

  **Example:**

  ```go
  subRouter := chirpc.NewRPCSubRouter()
  chirpc.AddHandler(subRouter, chirpc.MethodGet, "/profile", ProfileHandler)
  chirpc.Mount(router, "/user", subRouter)  // Accessible at /user/profile
  ```

- **`Group(router, fn, ...middlewares)`** Create an anonymous grouped sub-router with scoped middlewares. Similar to `Route` but without a path prefix.

  **Example:**

  ```go
  chirpc.Group(router, func(r *RPCRouter) {
      chirpc.AddHandler(r, chirpc.MethodGet, "/protected", ProtectedHandler)
  }, AuthMiddleware)
  ```

### Request Type Configuration

Methods returned by `AddHandler` for configuring expected request types:

- **`.BodyType(body any)`** Specify the expected HTTP request body type for TypeScript generation.
- **`.QueryType(query any)`** Specify the expected URL query parameter type for TypeScript generation.
- **`.Params(slugs []string)`** Set expected URL path parameter slugs. Usually auto-detected from path, but can be set manually if needed.

**Example:**

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type QueryParams struct {
    Page  int `json:"page" tsOptional:"true"`
    Limit int `json:"limit" tsOptional:"true"`
}

chirpc.AddHandler(router, chirpc.MethodPost, "/users", CreateUserHandler).
    BodyType(CreateUserRequest{}).
    QueryType(QueryParams{})
```

### Type Generation

- **`GenerateRPCSchema(router, path...string) error`** Generate TypeScript types for all registered handlers and write to file. Default path is `apiSchema.ts` in the current directory.

  **Example:**

  ```go
  // Write to default location (./apiSchema.ts)
  err := chirpc.GenerateRPCSchema(router)

  // Write to custom location
  err := chirpc.GenerateRPCSchema(router, "frontend/src/types/api.ts")
  ```

### HTTP Method Constants

Pre-defined HTTP method constants for use with `AddHandler`:

- `MethodGet` - HTTP GET
- `MethodPost` - HTTP POST
- `MethodPut` - HTTP PUT
- `MethodDelete` - HTTP DELETE
- `MethodPatch` - HTTP PATCH
- `MethodOptions` - HTTP OPTIONS
- `MethodHead` - HTTP HEAD
- `MethodTrace` - HTTP TRACE
- `MethodConnect` - HTTP CONNECT

### Custom HTTP Verbs

- **`RegisterMethod(method string)`** Register a custom HTTP method with chi for routing. Call before using the method in handlers.

  **Example:**

  ```go
  chirpc.RegisterMethod("CUSTOM")
  chirpc.AddHandler(router, "CUSTOM", "/endpoint", CustomHandler)
  ```

### Fallback Handlers

- **`NotFound(router, handler)`** Set custom handler for HTTP 404 Not Found responses.

  **Example:**

  ```go
  chirpc.NotFound(router, func(w http.ResponseWriter, r *http.Request) {
      w.WriteHeader(http.StatusNotFound)
      json.NewEncoder(w).Encode(map[string]string{"error": "Not Found"})
  })
  ```

- **`MethodNotAllowed(router, handler)`** Set custom handler for HTTP 405 Method Not Allowed responses.

  **Example:**

  ```go
  chirpc.MethodNotAllowed(router, func(w http.ResponseWriter, r *http.Request) {
      w.WriteHeader(http.StatusMethodNotAllowed)
      json.NewEncoder(w).Encode(map[string]string{"error": "Method Not Allowed"})
  })
  ```

### Server Utilities

- **`GetHttpServer() *http.Server`** Get the underlying `http.Server` instance for advanced configuration.

  **Example:**

  ```go
  server := router.GetHttpServer()
  server.Addr = ":8080"
  server.ReadTimeout = 10 * time.Second
  server.WriteTimeout = 10 * time.Second
  ```

- **`ListenAndServe(addr string) error`** Start the HTTP server on the specified address. Convenience method that calls `http.ListenAndServe`.

  **Example:**

  ```go
  err := router.ListenAndServe(":8080")
  ```

### Core Types

- **`HttpResponse[T any]`** Generic HTTP response structure with `StatusCode`, `Body`, and `Headers` fields.

  ```go
  type HttpResponse[T any] struct {
      StatusCode int
      Body       T
      Headers    map[string]string
  }
  ```

- **`ErrorResponse`** Structured error response with status code, error messages, and field-level validation errors.

  ```go
  type ErrorResponse struct {
      StatusCode       int                 `json:"statusCode,omitempty"`
      Errors           []string            `json:"errors,omitempty"`
      ValidationErrors map[string][]string `json:"validationErrors,omitempty"`
  }
  ```

- **`RequestHandler[T any]`** Handler function type that processes requests and returns typed responses or errors.

  ```go
  type RequestHandler[T any] func(*http.Request) (*HttpResponse[T], *ErrorResponse)
  ```

- **`ErrorHandlerType[T any]`** Error handler function type for processing error responses.

  ```go
  type ErrorHandlerType[T any] func(*http.Request, *ErrorResponse) *HttpResponse[T]
  ```

- **`MiddlewareType`** Type alias for chi middleware functions.

  ```go
  type MiddlewareType = func(http.Handler) http.Handler
  ```

## Contributing

Contributions are welcome! Whether you want to fix a bug, add a feature, or improve documentation, your help is appreciated.

### How to Contribute

1. **Open an Issue First** Before starting work, open an issue describing the bug fix, improvement, or feature you'd like to work on. This helps avoid duplicate work and ensures your contribution aligns with the project's direction.

2. **Fork and Clone** Fork the repository and clone it locally:

   ```bash
   git clone https://github.com/YOUR_USERNAME/chirpc.git
   cd chirpc
   ```

3. **Create a Branch** Create a feature branch for your changes:

   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Make Your Changes**

   - Write clear, idiomatic Go code
   - Follow existing code style and conventions
   - Add or update tests for new functionality
   - Update documentation if your changes affect the public API

5. **Run Tests** Ensure all tests pass before submitting:

   ```bash
   # Run all tests
   go test ./...

   # Run tests with coverage
   go test -cover ./...

   # Run tests for specific packages
   go test ./v1
   go test ./internal/tsGen
   go test ./internal/rpc

   # Generate coverage report
   go test -coverprofile=coverage.out ./...
   go tool cover -html=coverage.out
   ```

6. **Format and Lint** Ensure your code follows Go standards:

   ```bash
   # Format code
   go fmt ./...

   # Run go vet
   go vet ./...

   # Run staticcheck (if installed)
   staticcheck ./...
   ```

7. **Commit and Push** Write clear, descriptive commit messages:

   ```bash
   git add .
   git commit -m "feat: add support for custom response headers"
   git push origin feature/your-feature-name
   ```

8. **Open a Pull Request**
   - Provide a clear description of what your PR does
   - Reference any related issues
   - Ensure CI checks pass
   - Be responsive to feedback and requests for changes

### Development Guidelines

- **Keep PRs Focused**: Each pull request should address a single feature or bug fix
- **Write Tests**: All new features and bug fixes should include tests
- **Maintain Coverage**: Aim to maintain or improve test coverage
- **Update Documentation**: Update README.md, code comments, or examples if needed
- **Follow Conventions**: Use Go idioms and follow the existing code style
- **Be Respectful**: Follow the code of conduct and be kind to other contributors

### Running the Example

To test your changes with the example application:

```bash
# Run the Go example server
cd cmd/example
go run main.go

# In another terminal, test the TypeScript client
cd cmd/js
npm install
npx tsx main.ts
```

### Reporting Issues

When reporting bugs, please include:

- Go version (`go version`)
- Minimal code example that reproduces the issue
- Expected vs actual behavior
- Any relevant error messages or stack traces

### Feature Requests

For feature requests, please describe:

- The problem you're trying to solve
- Your proposed solution
- Any alternative solutions you've considered
- How this would benefit other users of chirpc

## License

MIT License © 2025 Bipin Maharjan

See [LICENSE](LICENSE) for full details.
