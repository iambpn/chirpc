# Project Guidelines

## Code Style
- Go 1.25+ with generics; TypeScript 4.5+ for generated clients
- Use Go's standard formatting (`gofmt`) and idiomatic error handling
- Struct tags control TypeScript generation: `tsKey`, `tsType`, `tsOptional`, `tsOmit`
- JSON tags are respected for field naming when `tsKey` is not specified

## Architecture
- **v1/**: Public API - `RPCRouter`, `AddHandler`, `AddMiddlewares`, `GenerateRPCSchema`
- **internal/rpc/**: Handler schema extraction and router type management
- **internal/tsGen/**: Go-to-TypeScript conversion engine with struct tag support
- **internal/tsGen/tsInterface/**: TypeScript interface builder using ordered maps
- **cmd/**: Example servers and TypeScript generation utilities

## Build and Test
```bash
# Run all tests with coverage
make test

# Run tests with verbose output
make test-v

# Generate HTML coverage report
make html-coverage

# Run example server
make run-server
```

## Project Conventions
- **Handler Pattern**: `RequestHandler[T]` returns `(*HttpResponse[T], *ErrorResponse)`
- **Type Registration**: Chain methods on `AddHandler` - `.BodyType(T{}).QueryType(T{})`
- **Path Params**: Auto-extracted from chi-style paths (`/{id}` → TypeScript `params: { id: string }`)
- **Nested Types**: Flattened into separate TypeScript interfaces with `Package__TypeName` naming
- **Pointer Types**: Become nullable TypeScript types (`*string` → `string | null`)
- **Unexported Fields**: Ignored during TypeScript generation

## TypeScript Generation
- Generated schema written to `apiSchema.ts` as `ApiSchema` type
- Works with [ts-axios-wrapper](https://www.npmjs.com/package/ts-axios-wrapper) for typed API calls
- Example client usage in `cmd/js/main.ts`

## Key Files
- `v1/rpcRouter.go`: Core router with handler registration
- `v1/types.go`: `HttpResponse`, `ErrorResponse`, `RequestHandler` types
- `internal/tsGen/tsGen.go`: Type conversion logic
- `internal/tsGen/structTag.helper.go`: Struct tag parsing
- `cmd/example/server/main.go`: Full server example
