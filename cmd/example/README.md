# chirpc example: task board API

A small but complete API that shows how the pieces of chirpc fit together: a Go server,
a generated TypeScript schema, and a typed client.

```
cmd/example
├── api/          The API: types, handlers, in-memory store, and NewRouter
│   └── router_test.go   Tests that call the router in memory with httptest
├── server/       Runs the API. It does not import tsgen, so the binary stays small
├── gen-schema/   Writes the TypeScript ApiSchema from the same NewRouter
└── client/       A TypeScript script that calls every route with ts-axios-wrapper
```

## Run it

From the repository root:

```bash
# 1. Start the server on :8080. It adds a few demo tasks.
make run-server

# 2. In another terminal, generate the schema and run the client.
make gen-schema
cd cmd/example/client
pnpm install
pnpm start
```

The client prints each step:

```
1. Health: ok at 2026-09-28T16:17:36Z
2. Signed in as Alice <alice@example.com>
3. Created task_4 [todo/high] Try the chirpc example (1/2 steps, tags: demo, docs)
4. Found 1 todo task(s) tagged "demo":
   - task_4 [todo/high] Try the chirpc example (1/2 steps, tags: demo, docs)
5. Updated task_4 [doing/high] Try the chirpc example (1/2 steps, tags: demo)
6. Bob gets 404: This task does not exist.
7. Invalid body gets 400: The request body is not valid.
   - checklist[0].text: This field is required.
8. After delete, the task gets 404: This task does not exist.
```

Run the Go tests with `go test ./cmd/example/...`. Set `API_URL` to point the client at another address.

You can also call the API with curl. Use the token `alice-token` or `bob-token`:

```bash
curl -H 'Authorization: Bearer alice-token' 'localhost:8080/api/v1/tasks?tag=docs&limit=5'
```

## Routes

| Method | Path | Shows |
| --- | --- | --- |
| GET | `/health` | A plain `AddHandler` handler with no auth |
| GET | `/api/v1/account/me` | A module router attached with `Mount` |
| GET | `/api/v1/tasks` | Typed query: enum filter, repeated `?tag=`, `time.Time`, paging |
| POST | `/api/v1/tasks` | Typed body with nested required fields, 201 with a `Location` header |
| GET | `/api/v1/tasks/{taskId}` | Path parameters and 404 errors |
| PATCH | `/api/v1/tasks/{taskId}` | Partial update with pointer fields |
| DELETE | `/api/v1/tasks/{taskId}` | Returning no response, which sends 204 |

All `/api/v1` routes are inside a `Group` with a bearer token middleware. Each user sees only their own tasks.

## What to look at

- **One router for the server and the generator.** [api/router.go](api/router.go) builds everything in `NewRouter`. [server/main.go](server/main.go) serves it, and [gen-schema/main.go](gen-schema/main.go) turns it into TypeScript. They cannot drift apart.
- **Two layers of validation.** chirpc rejects a missing field, a wrong JSON type, or a bad query value before the handler runs. The handlers in [api/handlers.go](api/handlers.go) check only the rules chirpc cannot know, such as an empty title or an unknown priority. Both return the same `validationErrors` shape.
- **One error shape.** `handleError` turns every `ErrorResponse` into `APIError`, and it logs `Cause` for 500 errors. The middleware and the 404 and 405 handlers write the same shape with `writeAPIError`. The client reads it as `ApiSchema["ERROR_HANDLER"]["/"]["response"]`.
- **Custom TypeScript types.** `RegisterTSType` makes `TaskID` a `string` (it has its own JSON form, `"task_7"`) and makes `Status` and `Priority` string unions.
- **Plain errors stay private.** A handler can return any Go error. chirpc sends it as a general 500 error, and only the error handler sees the details.
- **Testing without a network.** `RPCRouter` is an `http.Handler`, so [api/router_test.go](api/router_test.go) calls `router.ServeHTTP` with `httptest`.

## Notes on the client

- `paramsSerializer: { indexes: null }` makes axios send arrays as `?tag=a&tag=b`. The default `?tag[]=a` form does not match the Go query field.
- `ts-axios-wrapper` is published as CommonJS, so TypeScript sees its axios types as a different copy from the ESM `axios` import. [client/src/main.ts](client/src/main.ts) casts the instance once to join them. It is the same object at runtime.
- `apiSchema.ts` is generated and ignored by Git. Run `make gen-schema` or `pnpm gen` after you change a Go type.
