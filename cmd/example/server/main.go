// Command server runs the example task API.
//
// It does not import the tsgen package, so the binary does not include the TypeScript
// compiler. Run the gen-schema command to update the client types.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/iambpn/chirpc/cmd/example/api"
)

func main() {
	addr := flag.String("addr", ":8080", "The address to listen on.")
	flag.Parse()

	store := api.NewStore()
	seed(store)

	// RPCRouter is an http.Handler, so it can be wrapped like any other handler.
	server := &http.Server{
		Addr:              *addr,
		Handler:           middleware.Logger(api.NewRouter(store)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("The server is listening.", "addr", *addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("The server stopped with an error.", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("The server is shutting down.")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("The server did not shut down cleanly.", "error", err)
	}
}

// seed adds a few tasks, so the list endpoint has data before the client creates any.
func seed(store *api.Store) {
	dueAt := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)

	store.Create("u_alice", api.Task{
		Title:    "Write the release notes",
		Status:   api.StatusDoing,
		Priority: api.PriorityHigh,
		Tags:     []string{"docs", "release"},
		DueAt:    &dueAt,
		Checklist: []api.ChecklistItem{
			{Text: "List the new features", Done: true},
			{Text: "List the breaking changes"},
		},
	})
	store.Create("u_alice", api.Task{
		Title:    "Clean up old branches",
		Status:   api.StatusTodo,
		Priority: api.PriorityLow,
		Tags:     []string{"chore"},
	})
	store.Create("u_bob", api.Task{
		Title:    "Only Bob can see this task",
		Status:   api.StatusTodo,
		Priority: api.PriorityMedium,
	})
}
