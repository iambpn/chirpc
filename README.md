# chirpc

> End-to-end type-safe RPC toolkit for Go chi routers and TypeScript clients.

## Introduction

chirpc wraps the excellent [chi](https://github.com/go-chi/chi) router with a thin RPC layer that keeps Go handlers and TypeScript consumers in lockstep. Each handler you register produces a strongly typed schema that can be consumed on the frontend, eliminating hand-written DTOs and mismatched payloads.

## Features

- Drop-in `chi` router wrapper with generics-based request handlers and global middlewares.
- Automatic TypeScript interface generation for handler responses, nested structs, and custom tags.
- Configurable error handler that surfaces typed error payloads to both Go and TypeScript clients.
- Custom HTTP verb support plus helpers for grouping, mounting, and composing routers.
- Codegen pipeline that emits an `ApiSchema` consumed by `ts-axios-wrapper` for typed API calls.

## Installation

- Prerequisites: Go 1.21+ and Node.js 18+ (for consuming generated TypeScript types).
- Add the module to your project: `go get github.com/iambpn/chirpc/v1`.
- (Optional) Install TypeScript dependencies: `npm install ts-axios-wrapper` in your frontend workspace.

## Usage

```go
package main

import (
    "net/http"

    "github.com/go-chi/chi/v5/middleware"
    "github.com/iambpn/chirpc/v1"
    "os"
)

type ErrorBody struct {
    Message string `json:"message"`
}

type HelloResponse struct {
    Message string `json:"message"`
}

func main() {
    router := chirpc.NewRPCRouter()

    chirpc.AddGlobalMiddlewares(router, middleware.Logger)

    chirpc.AddHandler(router, chirpc.GET, "/", func(r *http.Request) (*chirpc.HttpResponse[HelloResponse], error) {
        return &chirpc.HttpResponse[HelloResponse]{
            StatusCode: http.StatusOK,
            Body:       HelloResponse{Message: "Hello, world!"},
            Headers: map[string]string{
                "Content-Type": "application/json",
            },
        }, nil
    })

    chirpc.RegisterErrorHandler(func(r *http.Request, err error) chirpc.HttpResponse[ErrorBody] {
        return chirpc.HttpResponse[ErrorBody]{
            StatusCode: http.StatusInternalServerError,
            Body:       ErrorBody{Message: err.Error()},
            Headers: map[string]string{
                "Content-Type": "application/json",
            },
        }
    })

    // only run during development to generate apiSchema.ts
    if os.Getenv("ENV") == "development" {
      if err := chirpc.BuildRpcTypes(); err != nil {
          panic("failed to generate apiSchema: " + err.Error())
      }
    }

    server := router.GetHttpServer()
    server.Addr = ":8080"
    if err := server.ListenAndServe(); err != nil {
        panic(err)
    }
}
```

- `BuildRpcTypes()` writes the generated TypeScript schema to `apiSchema.ts` (override the destination via `BuildRpcTypes("web/src/apiSchema.ts")`).
- Annotate struct fields with `tsKey`, `tsType`, or `tsOptional` to customize generated interfaces.
- Each registered handler contributes to the generated `ApiSchema`, including global error handlers (under the synthetic `ERROR_HANDLER` method).

```typescript
// frontend/client.ts
import { TypedAxios } from "ts-axios-wrapper";
import type { ApiSchema } from "../apiSchema.js";

const api = new TypedAxios<ApiSchema>({ baseURL: "http://localhost:8080" });

const response = await api.GET("/", {});
console.log(response.body.message);
```

- `TypedAxios<ApiSchema>` infers valid methods, paths, and payload shapes from the generated schema.
- Optional request metadata (params, query, body) are enforced at compile time, ensuring end-to-end safety.

## Exposed APIs

- `NewRPCRouter()` – create a router backed by `chi.Mux`.
- `AddHandler(router, method, path, RequestHandler[T], ...middlewares)` – register typed handlers and capture schema metadata.
- `AddGlobalMiddlewares(router, ...middlewares)` – attach middlewares to every route.
- `Route`, `Group`, `Mount` – compose sub-routers with scoped middleware stacks.
- `RegisterErrorHandler(handler)` – define a typed fallback invoked when handlers return errors.
- `RegisterMethod(method)` – declare custom HTTP verbs for a router.
- `MethodNotAllowed`, `NotFound` – override default fallback handlers.
- `BuildRpcTypes(path...)` – emit the aggregated TypeScript schema to disk. Should only run while in development.

## Contributing

- Open an issue describing the improvement or bug before submitting a pull request.
- Ensure `go test ./...` passes and add coverage for new behavior.
- Follow Go formatting conventions and limit PR scope to focused changes.

## License

- MIT License © 2025 Bipin Maharjan (see `LICENSE`).
