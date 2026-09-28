// Package api is a small task board API built with chirpc. The server and the schema
// generator both build their router with NewRouter, so the generated TypeScript always
// matches the routes that are served.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	chirpc "github.com/iambpn/chirpc/v1"
)

// NewRouter builds the whole API on top of the given store.
//
// Routes:
//
//	GET    /health                      no auth, plain AddHandler
//	GET    /api/v1/account/me           the signed-in user, from a mounted router
//	GET    /api/v1/tasks                list with query filters and paging
//	POST   /api/v1/tasks                create, with nested body validation
//	GET    /api/v1/tasks/{taskId}       read one task
//	PATCH  /api/v1/tasks/{taskId}       partial update
//	DELETE /api/v1/tasks/{taskId}       delete, returns 204
func NewRouter(store *Store) *chirpc.RPCRouter {
	r := chirpc.NewRPCRouter()

	// One error handler for the whole tree, so ApiSchema has one error type.
	chirpc.RegisterErrorHandler(r, handleError)

	// These types have their own JSON form, so the TypeScript must not follow their Go kind.
	chirpc.RegisterTSType[TaskID](r, "string")
	chirpc.RegisterTSType[Status](r, `"todo" | "doing" | "done"`)
	chirpc.RegisterTSType[Priority](r, `"low" | "medium" | "high"`)

	// chi needs middlewares before routes. Request logging is added by the server,
	// so tests and the schema generator stay quiet.
	chirpc.AddMiddlewares(r, middleware.RequestID, middleware.Recoverer)

	chirpc.NotFound(r, func(w http.ResponseWriter, req *http.Request) {
		writeAPIError(w, req, http.StatusNotFound, "This route does not exist.")
	})
	chirpc.MethodNotAllowed(r, func(w http.ResponseWriter, req *http.Request) {
		writeAPIError(w, req, http.StatusMethodNotAllowed, "This route does not accept the "+req.Method+" method.")
	})

	chirpc.AddHandler(r, chirpc.MethodGet, "/health", health)

	tasks := &taskHandlers{store: store}
	chirpc.Route(r, "/api/v1", func(v1 *chirpc.RPCRouter) {
		// Every route in this group needs a bearer token.
		chirpc.Group(v1, func(authed *chirpc.RPCRouter) {
			chirpc.AddTypedHandler(authed, chirpc.MethodGet, "/tasks", tasks.list)
			chirpc.AddTypedHandler(authed, chirpc.MethodPost, "/tasks", tasks.create)
			chirpc.AddTypedHandler(authed, chirpc.MethodGet, "/tasks/{taskId}", tasks.get)
			chirpc.AddTypedHandler(authed, chirpc.MethodPatch, "/tasks/{taskId}", tasks.update)
			chirpc.AddTypedHandler(authed, chirpc.MethodDelete, "/tasks/{taskId}", tasks.delete)

			chirpc.Mount(authed, "/account", newAccountRouter())
		}, requireUser(DemoUsers))
	})

	return r
}

// newAccountRouter builds the account module as its own router. Any router can be
// mounted in another one, and its routes join the parent's schema.
func newAccountRouter() *chirpc.RPCRouter {
	r := chirpc.NewRPCRouter()
	chirpc.AddTypedHandler(r, chirpc.MethodGet, "/me", getMe)
	return r
}
