package chirpc

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRPCRouterCreatesChiMux(t *testing.T) {
	router := NewRPCRouter()

	if router == nil {
		t.Fatal("expected router to be created")
	}

	if router.router == nil {
		t.Fatal("expected underlying chi router to be initialized")
	}
}

func TestGetHttpServerSharesRouter(t *testing.T) {
	r := NewRPCRouter()

	AddHandler(r, MethodGet, "/ping", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "pong"}, nil
	}))

	server := r.GetHttpServer()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	recorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestListenAndServePropagatesErrors(t *testing.T) {
	r := NewRPCRouter()
	if err := r.ListenAndServe("invalid-address"); err == nil {
		t.Fatal("expected error for invalid listen address")
	}
}

func TestAddGlobalMiddlewaresAreApplied(t *testing.T) {
	r := NewRPCRouter()

	AddMiddlewares(r, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Global", "hit")
			next.ServeHTTP(w, req)
		})
	})

	AddHandler(r, MethodGet, "/global", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "ok"}, nil
	}))

	req := httptest.NewRequest(http.MethodGet, "/global", nil)
	recorder := httptest.NewRecorder()
	r.router.ServeHTTP(recorder, req)

	if recorder.Header().Get("X-Global") != "hit" {
		t.Fatal("expected global middleware to set header")
	}
}

func TestAddHandlerSpecificMiddlewareRuns(t *testing.T) {
	r := NewRPCRouter()

	AddHandler(r, MethodGet, "/middle", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "ok"}, nil
	}), func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Route", "1")
			next.ServeHTTP(w, req)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/middle", nil)
	recorder := httptest.NewRecorder()
	r.router.ServeHTTP(recorder, req)

	if recorder.Header().Get("X-Route") != "1" {
		t.Fatal("expected route middleware to set header")
	}
}

func TestRegisterErrorHandlerHandlesErrors(t *testing.T) {
	defer func() { errorHandler = nil }()

	router := NewRPCRouter()
	RegisterErrorHandler(router, func(r *http.Request, er *ErrorResponse) *HttpResponse[string] {
		return &HttpResponse[string]{
			StatusCode: http.StatusBadRequest,
			Body:       "handled",
		}
	})

	AddHandler(router, MethodGet, "/fail", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return nil, &ErrorResponse{
			StatusCode: http.StatusInternalServerError,
			Errors:     []string{"original error"},
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/fail", nil)
	recorder := httptest.NewRecorder()
	router.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	if body := strings.TrimSpace(recorder.Body.String()); body != "\"handled\"" {
		t.Fatalf("expected body to be %q, got %q", "\"handled\"", body)
	}
}

func TestRouteMountsSubRouterWithMiddlewares(t *testing.T) {
	r := NewRPCRouter()
	hits := 0

	Route(r, "/api", func(sub *RPCRouter) {
		AddHandler(sub, MethodGet, "/ping", func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
			return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "pong"}, nil
		})
	}, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			hits++
			next.ServeHTTP(w, req)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	recorder := httptest.NewRecorder()
	r.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if hits != 1 {
		t.Fatalf("expected middleware to run once, ran %d times", hits)
	}
}

func TestMountAttachesSubRouter(t *testing.T) {
	root := NewRPCRouter()
	sub := NewRPCSubRouter()

	AddHandler(sub, MethodGet, "/child", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "child"}, nil
	}))

	Mount(root, "/prefix", sub)

	req := httptest.NewRequest(http.MethodGet, "/prefix/child", nil)
	recorder := httptest.NewRecorder()
	root.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if recorder.Body.String() != "\"child\"" {
		t.Fatalf("expected body to be %q, got %q", "\"child\"", recorder.Body.String())
	}
}

func TestMountOnRouteWithMiddlewares(t *testing.T) {
	r := NewRPCRouter()
	hits := 0

	sub := NewRPCSubRouter()
	AddHandler(sub, MethodGet, "/child", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "child"}, nil
	}))

	Route(r, "/api", func(subRouter *RPCRouter) {
		Mount(subRouter, "/prefix", sub)
	}, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			hits++
			next.ServeHTTP(w, req)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/prefix/child", nil)
	recorder := httptest.NewRecorder()
	r.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if hits != 1 {
		t.Fatalf("expected middleware to run once, ran %d times", hits)
	}

	// Verify generated ts types
	path := "apiSchemaMountOnRoute.ts"
	t.Cleanup(func() { _ = os.Remove(path) })

	if err := GenerateRPCSchema(r, path); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type ApiSchema") {
		t.Fatalf("expected generated schema to contain type definition")
	}

	if !strings.Contains(content, "/api/prefix/child") {
		t.Fatalf("expected schema to contain /api/prefix/child route")
	}
}

func TestGroupAppliesMiddlewaresToNestedHandlers(t *testing.T) {
	r := NewRPCRouter()
	hits := 0

	Group(r, func(sub *RPCRouter) {
		AddHandler(sub, MethodGet, "/group", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
			return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "group"}, nil
		}))
	}, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			hits++
			next.ServeHTTP(w, req)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/group", nil)
	recorder := httptest.NewRecorder()
	r.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if hits != 1 {
		t.Fatalf("expected middleware to run once, ran %d times", hits)
	}
}

func TestMethodNotAllowedHandlerOverridesDefault(t *testing.T) {
	r := NewRPCRouter()

	MethodNotAllowed(r, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	AddHandler(r, MethodGet, "/only-get", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "done"}, nil
	}))

	req := httptest.NewRequest(http.MethodPost, "/only-get", nil)
	recorder := httptest.NewRecorder()
	r.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, recorder.Code)
	}
}

func TestNotFoundHandlerOverridesDefault(t *testing.T) {
	r := NewRPCRouter()

	NotFound(r, func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "gone", http.StatusGone)
	})

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	recorder := httptest.NewRecorder()
	r.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusGone {
		t.Fatalf("expected status %d, got %d", http.StatusGone, recorder.Code)
	}
}

func TestRegisterMethodSupportsCustomVerb(t *testing.T) {
	r := NewRPCRouter()
	const customMethod = "CUSTOM"
	RegisterMethod(customMethod)

	AddHandler(r, HttpMethods(customMethod), "/custom", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusAccepted, Body: "ok"}, nil
	}))

	req := httptest.NewRequest(customMethod, "/custom", nil)
	recorder := httptest.NewRecorder()
	r.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, recorder.Code)
	}
}

func TestBuildRpcTypesWritesDefaultFile(t *testing.T) {
	path := "apiSchema.ts"
	t.Cleanup(func() { _ = os.Remove(path) })

	r := NewRPCRouter()
	if err := GenerateRPCSchema(r, path); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	if !strings.Contains(string(data), "type ApiSchema") {
		t.Fatalf("expected generated schema to contain type definition")
	}
}

func TestBuildRpcTypesWritesToCustomPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.ts")

	r := NewRPCRouter()
	if err := GenerateRPCSchema(r, path); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	if !strings.Contains(string(data), "type ApiSchema") {
		t.Fatalf("expected generated schema to contain type definition")
	}
}

func TestBuildRpcTypesReturnsErrorWhenWriteFails(t *testing.T) {
	dir := t.TempDir()
	unreachable := filepath.Join(dir, "nested", "schema.ts")

	r := NewRPCRouter()
	err := GenerateRPCSchema(r, unreachable)
	if err == nil {
		t.Fatal("expected error when parent directories are missing")
	}

	if !strings.Contains(err.Error(), "failed to write types to file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterErrorHandlerWrapsTypedResponse(t *testing.T) {
	defer func() { errorHandler = nil }()

	router := NewRPCRouter()
	RegisterErrorHandler(router, func(r *http.Request, err *ErrorResponse) *HttpResponse[map[string]string] {
		return &HttpResponse[map[string]string]{
			StatusCode: http.StatusInternalServerError,
			Body:       map[string]string{"error": err.Errors[0]},
		}
	})

	AddHandler(router, MethodGet, "/err", RequestHandler[map[string]string](func(req *http.Request) (*HttpResponse[map[string]string], *ErrorResponse) {
		return nil, &ErrorResponse{
			StatusCode: http.StatusInternalServerError,
			Errors:     []string{"failed"},
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	recorder := httptest.NewRecorder()
	router.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	bodyBytes, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if !strings.Contains(string(bodyBytes), "failed") {
		t.Fatalf("expected body to contain error message, got %q", string(bodyBytes))
	}
}

func TestAddHandlerReturnsBodyQueryParamWithSchema(t *testing.T) {
	r := NewRPCRouter()
	bqp := AddHandler(r, MethodGet, "/items/{id}", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "ok"}, nil
	}))
	if bqp == nil {
		t.Fatal("expected BodyQueryParamType pointer, got nil")
	}
	if bqp.Schema == nil {
		t.Fatal("expected Schema to be populated")
	}
}

func TestMiddlewareOrderGlobalThenRoute(t *testing.T) {
	r := NewRPCRouter()

	AddMiddlewares(r, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Order", "global")
			next.ServeHTTP(w, req)
		})
	})

	AddHandler(r, MethodGet, "/mw-order", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "ok"}, nil
	}), func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("X-Order", w.Header().Get("X-Order")+"-route")
			next.ServeHTTP(w, req)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/mw-order", nil)
	rec := httptest.NewRecorder()
	r.router.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Order"); got != "global-route" {
		t.Fatalf("expected header 'global-route', got %q", got)
	}
}

func TestRegisterErrorHandlerSetsGlobalHandler(t *testing.T) {
	defer func() { errorHandler = nil }()
	if errorHandler != nil {
		t.Fatal("expected initial global errorHandler to be nil")
	}

	r := NewRPCRouter()
	RegisterErrorHandler(r, func(r *http.Request, err *ErrorResponse) *HttpResponse[string] {
		return &HttpResponse[string]{StatusCode: http.StatusBadRequest, Body: "handled"}
	})

	if errorHandler == nil {
		t.Fatal("expected global errorHandler to be set")
	}
}

func TestGenerateRpcTypesWithRouteMountGroup(t *testing.T) {
	path := "apiSchemaRouteMountGroup.ts"
	t.Cleanup(func() { _ = os.Remove(path) })

	r := NewRPCRouter()

	// Using Route
	Route(r, "/api", func(sub *RPCRouter) {
		AddHandler(sub, MethodGet, "/ping", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
			return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "pong"}, nil
		}))
	})

	// Using Mount
	sub := NewRPCSubRouter()
	AddHandler(sub, MethodGet, "/child", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "child"}, nil
	}))
	Mount(r, "/prefix", sub)

	// Using Group
	Group(r, func(sub *RPCRouter) {
		AddHandler(sub, MethodGet, "/group", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
			return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "group"}, nil
		}))
	})

	if err := GenerateRPCSchema(r, path); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type ApiSchema") {
		t.Fatalf("expected generated schema to contain type definition")
	}
	if !strings.Contains(content, "/api/ping") {
		t.Fatalf("expected schema to contain /api/ping route")
	}
	if !strings.Contains(content, "/prefix/child") {
		t.Fatalf("expected schema to contain /prefix/child route")
	}

	if !strings.Contains(content, "/group") {
		t.Fatalf("expected schema to contain /group route")
	}
}

type fakeRouter struct{}

func (f *fakeRouter) isRpcRouter() bool { return true }

func TestAddHandlerWithUnknownRouterType(t *testing.T) {
	fr := &fakeRouter{}
	bqp := AddHandler(fr, MethodGet, "/unknown", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusNoContent}, nil
	}))

	if bqp == nil {
		t.Fatal("expected BodyQueryParamType pointer, got nil")
	}
	if bqp.Schema != nil {
		t.Fatalf("expected Schema to be nil for unknown router type, got %v", bqp.Schema)
	}
}

func TestAddHandlerOnSubRouterRecordsSchema(t *testing.T) {
	sub := NewRPCSubRouter()
	bqp := AddHandler(sub, MethodGet, "/child", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "child"}, nil
	}))

	if len(sub.subRoutes) != 1 {
		t.Fatalf("expected one schema recorded, got %d", len(sub.subRoutes))
	}
	schema := sub.subRoutes[0]
	if schema.URL() != "/child" {
		t.Fatalf("expected schema URL %q, got %q", "/child", schema.URL())
	}
	if bqp == nil || bqp.Schema != schema {
		t.Fatalf("expected BodyQueryParamType to reference recorded schema, got %v", bqp.Schema)
	}
}
