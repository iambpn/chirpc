package api

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

// DemoUsers maps each demo bearer token to its user. A real application would check
// a session or a signed token instead.
var DemoUsers = map[string]User{
	"alice-token": {ID: "u_alice", Name: "Alice", Email: "alice@example.com"},
	"bob-token":   {ID: "u_bob", Name: "Bob", Email: "bob@example.com"},
}

type userKey struct{}

// requireUser is a plain chi middleware. It rejects requests without a known bearer
// token and stores the user in the request context for the handlers.
func requireUser(users map[string]User) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			user, known := users[token]
			if !ok || !known {
				writeAPIError(w, r, http.StatusUnauthorized, "Send a valid bearer token in the Authorization header.")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
		})
	}
}

// currentUser returns the user that requireUser stored in the context.
func currentUser(ctx context.Context) User {
	user, _ := ctx.Value(userKey{}).(User)
	return user
}

// writeAPIError writes an APIError from code that runs outside a chirpc handler, such as
// middleware and the 404 handler, so every error has the same JSON shape.
func writeAPIError(w http.ResponseWriter, r *http.Request, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.MarshalWrite(w, APIError{
		Status:    status,
		Message:   message,
		RequestID: middleware.GetReqID(r.Context()),
	})
}
