package chirpc

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/iambpn/chirpc/internal/typescript"
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

func TestDefaultErrorResponseWhenNoErrorHandler(t *testing.T) {
	router := NewRPCRouter()

	AddHandler(router, MethodGet, "/error-no-handler", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return nil, &ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Errors:     []string{"something went wrong"},
			ValidationErrors: map[string][]string{
				"field1": {"error1", "error2"},
			},
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/error-no-handler", nil)
	recorder := httptest.NewRecorder()
	router.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected the ErrorResponse status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("expected Content-Type to be %q, got %q", "application/json", contentType)
	}

	body := strings.TrimSpace(recorder.Body.String())
	expectedFields := []string{"statusCode", "errors", "validationErrors"}
	for _, field := range expectedFields {
		if !strings.Contains(body, field) {
			t.Errorf("expected response body to contain field %q, got: %s", field, body)
		}
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

	content := generateSchema(t, r)
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

func TestRegisterErrorHandlerWrapsTypedResponse(t *testing.T) {
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
		t.Fatal("expected ParamsBuilder pointer, got nil")
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

func TestRegisterErrorHandlerSetsRouterHandler(t *testing.T) {
	r := NewRPCRouter()
	other := NewRPCRouter()
	RegisterErrorHandler(r, func(r *http.Request, err *ErrorResponse) *HttpResponse[string] {
		return &HttpResponse[string]{StatusCode: http.StatusBadRequest, Body: "handled"}
	})

	if r.errorHandler == nil {
		t.Fatal("expected router errorHandler to be set")
	}
	if other.errorHandler != nil {
		t.Fatal("expected other router errorHandler to stay nil")
	}
}

func TestGenerateRpcTypesWithRouteMountGroup(t *testing.T) {

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

	content := generateSchema(t, r)
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

func TestAddHandlerOnSubRouterRecordsSchema(t *testing.T) {
	sub := NewRPCSubRouter()
	bqp := AddHandler(sub, MethodGet, "/child", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "child"}, nil
	}))

	if bqp == nil || bqp.Schema == nil {
		t.Fatal("expected ParamsBuilder to reference a recorded schema")
	}
	if bqp.Schema.URL() != "/child" {
		t.Fatalf("expected schema URL %q, got %q", "/child", bqp.Schema.URL())
	}
}

func TestDefaultStatusCodeForSuccessResponse(t *testing.T) {
	r := NewRPCRouter()

	AddHandler(r, MethodGet, "/default-status", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		// Return response with StatusCode = 0
		return &HttpResponse[string]{StatusCode: 0, Body: "ok"}, nil
	}))

	req := httptest.NewRequest(http.MethodGet, "/default-status", nil)
	recorder := httptest.NewRecorder()
	r.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected default status %d (OK), got %d", http.StatusOK, recorder.Code)
	}
}

func TestDefaultStatusCodeForErrorHandler(t *testing.T) {
	router := NewRPCRouter()
	RegisterErrorHandler(router, func(r *http.Request, err *ErrorResponse) *HttpResponse[string] {
		// Return error response with StatusCode = 0
		return &HttpResponse[string]{
			StatusCode: 0,
			Body:       "handled",
		}
	})

	AddHandler(router, MethodGet, "/error-default", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return nil, &ErrorResponse{
			StatusCode: http.StatusBadRequest,
			Errors:     []string{"error occurred"},
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/error-default", nil)
	recorder := httptest.NewRecorder()
	router.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected the ErrorResponse status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestErrorHandlerWithoutStatusCodesUses500(t *testing.T) {
	router := NewRPCRouter()
	RegisterErrorHandler(router, func(r *http.Request, err *ErrorResponse) *HttpResponse[string] {
		return &HttpResponse[string]{Body: "handled"}
	})
	AddHandler(router, MethodGet, "/fail", failHandler)

	if code := serve(router, http.MethodGet, "/fail").Code; code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, code)
	}
}

func TestMount_WithNilSubRouter(t *testing.T) {
	router := NewRPCRouter()

	// This should return early and not panic
	Mount(router, "/test", nil)

	// If it didn't panic, the test passes
}

func TestMount_AdjustsSchemaURLs(t *testing.T) {
	router := NewRPCRouter()
	subRouter := NewRPCSubRouter()

	AddHandler(subRouter, MethodGet, "/users", RequestHandler[string](func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return &HttpResponse[string]{StatusCode: http.StatusOK, Body: "users"}, nil
	}))

	Mount(router, "/api", subRouter)

	// Verify the mounted route is accessible at the adjusted path
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	recorder := httptest.NewRecorder()
	router.router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200 at /api/users, got %d", recorder.Code)
	}
}

func TestGenerateRPCSchema_CircularDependencies(t *testing.T) {

	// Define circular dependency types
	// Node has a self-reference through Parent field and Children slice
	type Node struct {
		ID       int     `json:"id"`
		Value    string  `json:"value"`
		Parent   *Node   `json:"parent,omitempty"`
		Children []*Node `json:"children,omitempty"`
	}

	type TreeResponse struct {
		Root *Node `json:"root"`
	}

	r := NewRPCRouter()

	// Add handler with circular dependency in response type
	AddHandler(r, MethodGet, "/tree", RequestHandler[TreeResponse](func(req *http.Request) (*HttpResponse[TreeResponse], *ErrorResponse) {
		return &HttpResponse[TreeResponse]{
			StatusCode: http.StatusOK,
			Body: TreeResponse{
				Root: &Node{
					ID:    1,
					Value: "root",
					Children: []*Node{
						{ID: 2, Value: "child1"},
						{ID: 3, Value: "child2"},
					},
				},
			},
		}, nil
	}))

	content := generateSchema(t, r)

	// Verify the schema was generated
	if !strings.Contains(content, "type ApiSchema") {
		t.Fatalf("expected generated schema to contain type definition")
	}

	// Verify the route is included
	if !strings.Contains(content, "/tree") {
		t.Fatalf("expected schema to contain /tree route")
	}

	// Verify Node interface is generated only once (not duplicated due to circular ref)
	nodeCount := strings.Count(content, "interface Chirpc__Node")
	if nodeCount != 1 {
		t.Fatalf("expected Node interface to be defined exactly once, found %d occurrences", nodeCount)
	}

	// Verify TreeResponse interface exists
	if !strings.Contains(content, "Chirpc__TreeResponse") {
		t.Fatalf("expected schema to contain TreeResponse interface")
	}

	// Verify circular reference fields are present
	if !strings.Contains(content, "parent") || !strings.Contains(content, "children") {
		t.Fatalf("expected schema to contain circular reference fields (parent, children)")
	}
}

func okHandler(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
	return &HttpResponse[string]{Body: req.URL.Path}, nil
}

func failHandler(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
	return nil, &ErrorResponse{Errors: []string{"failed"}}
}

func serve(r *RPCRouter, method, url string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	r.router.ServeHTTP(recorder, httptest.NewRequest(method, url, nil))
	return recorder
}

func generateSchema(t *testing.T, r *RPCRouter) string {
	t.Helper()
	content, err := typescript.Convert(r.routerTypes)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	return content
}

func TestNestedRoutesAndGroupsUseFullPathInSchema(t *testing.T) {
	r := NewRPCRouter()
	Route(r, "/a", func(a *RPCRouter) {
		Route(a, "/b", func(b *RPCRouter) {
			AddHandler(b, MethodGet, "/nested", okHandler)
		})
		Group(a, func(g *RPCRouter) {
			AddHandler(g, MethodGet, "/grouped", okHandler)
		})
	})

	for _, url := range []string{"/a/b/nested", "/a/grouped"} {
		if code := serve(r, http.MethodGet, url).Code; code != http.StatusOK {
			t.Fatalf("expected %s to be served, got status %d", url, code)
		}
		if !strings.Contains(generateSchema(t, r), `"`+url+`"`) {
			t.Fatalf("expected schema to contain %s", url)
		}
	}
}

func TestTwoGroupsOnSameRouter(t *testing.T) {
	r := NewRPCRouter()
	Group(r, func(g *RPCRouter) { AddHandler(g, MethodGet, "/g1", okHandler) })
	Group(r, func(g *RPCRouter) { AddHandler(g, MethodGet, "/g2", okHandler) })
	AddHandler(r, MethodGet, "/top", okHandler)

	for _, url := range []string{"/g1", "/g2", "/top"} {
		if code := serve(r, http.MethodGet, url).Code; code != http.StatusOK {
			t.Fatalf("expected %s to be served, got status %d", url, code)
		}
	}
}

func TestParamsFromRoutePrefixAndRegexInSchema(t *testing.T) {
	r := NewRPCRouter()
	Route(r, "/users/{uid}", func(u *RPCRouter) {
		AddHandler(u, MethodGet, "/posts/{postId:[0-9]+}", okHandler)
	})
	sub := NewRPCSubRouter()
	AddHandler(sub, MethodGet, "/{code:[a-z]{3}}/{x}", okHandler)
	Mount(r, "/teams/{teamId}", sub)

	if code := serve(r, http.MethodGet, "/users/7/posts/42").Code; code != http.StatusOK {
		t.Fatalf("expected regex route to be served, got status %d", code)
	}

	content := generateSchema(t, r)
	for _, want := range []string{
		`"/users/:uid/posts/:postId": {
      params: { "uid": string;"postId": string; };`,
		`"/teams/:teamId/:code/:x": {
      params: { "teamId": string;"code": string;"x": string; };`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected schema to contain %s, got %s", want, content)
		}
	}
}

func TestMountIncludesLaterHandlersAndSupportsTwoPaths(t *testing.T) {
	r := NewRPCRouter()
	sub := NewRPCSubRouter()
	AddHandler(sub, MethodGet, "/early", okHandler)
	Mount(r, "/m1", sub)
	Mount(r, "/m2", sub)
	AddHandler(sub, MethodGet, "/late", okHandler)

	content := generateSchema(t, r)
	for _, url := range []string{"/m1/early", "/m1/late", "/m2/early", "/m2/late"} {
		if code := serve(r, http.MethodGet, url).Code; code != http.StatusOK {
			t.Fatalf("expected %s to be served, got status %d", url, code)
		}
		if !strings.Contains(content, `"`+url+`"`) {
			t.Fatalf("expected schema to contain %s, got %s", url, content)
		}
	}
}

func TestDefaultErrorResponseWithoutStatusCodeUses500(t *testing.T) {
	r := NewRPCRouter()
	AddHandler(r, MethodGet, "/fail", failHandler)

	if code := serve(r, http.MethodGet, "/fail").Code; code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, code)
	}
}

func TestNilResponsesDoNotPanic(t *testing.T) {
	r := NewRPCRouter()
	AddHandler(r, MethodGet, "/nil", func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return nil, nil
	})
	AddHandler(r, MethodGet, "/nil-body", func(req *http.Request) (*HttpResponse[any], *ErrorResponse) {
		return &HttpResponse[any]{}, nil
	})

	recorder := serve(r, http.MethodGet, "/nil")
	if recorder.Code != http.StatusNoContent || recorder.Body.Len() != 0 {
		t.Fatalf("expected 204 with no body, got %d %q", recorder.Code, recorder.Body.String())
	}

	recorder = serve(r, http.MethodGet, "/nil-body")
	if recorder.Code != http.StatusOK || recorder.Body.String() != "null" {
		t.Fatalf("expected 200 with null body, got %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestErrorHandlerReturningNilFallsBackToDefault(t *testing.T) {
	r := NewRPCRouter()
	RegisterErrorHandler(r, func(req *http.Request, err *ErrorResponse) *HttpResponse[ErrorResponse] {
		return nil
	})
	AddHandler(r, MethodGet, "/fail", func(req *http.Request) (*HttpResponse[string], *ErrorResponse) {
		return nil, &ErrorResponse{StatusCode: http.StatusNotFound, Errors: []string{"missing"}}
	})

	recorder := serve(r, http.MethodGet, "/fail")
	if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), "missing") {
		t.Fatalf("expected default 404 error response, got %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestSuccessResponseSetsJSONContentType(t *testing.T) {
	r := NewRPCRouter()
	AddHandler(r, MethodGet, "/ok", okHandler)

	if contentType := serve(r, http.MethodGet, "/ok").Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", contentType)
	}
}

func TestErrorHandlerIsScopedPerRouterAndReadAtRequestTime(t *testing.T) {
	type customError struct {
		Message string `json:"message"`
	}
	custom := func(req *http.Request, err *ErrorResponse) *HttpResponse[customError] {
		return &HttpResponse[customError]{StatusCode: http.StatusTeapot, Body: customError{Message: "custom"}}
	}

	r := NewRPCRouter()
	AddHandler(r, MethodGet, "/before", failHandler)
	RegisterErrorHandler(r, custom)
	AddHandler(r, MethodGet, "/after", failHandler)

	other := NewRPCRouter()
	AddHandler(other, MethodGet, "/other", failHandler)

	for _, url := range []string{"/before", "/after"} {
		if code := serve(r, http.MethodGet, url).Code; code != http.StatusTeapot {
			t.Fatalf("expected %s to use the registered error handler, got status %d", url, code)
		}
	}
	if code := serve(other, http.MethodGet, "/other").Code; code != http.StatusInternalServerError {
		t.Fatalf("expected other router to keep the default error response, got status %d", code)
	}
}

func TestChildRouterErrorHandler(t *testing.T) {
	rootHandler := func(req *http.Request, err *ErrorResponse) *HttpResponse[ErrorResponse] {
		return &HttpResponse[ErrorResponse]{StatusCode: http.StatusBadRequest, Body: ErrorResponse{Errors: []string{"root"}}}
	}
	childHandler := func(req *http.Request, err *ErrorResponse) *HttpResponse[ErrorResponse] {
		return &HttpResponse[ErrorResponse]{StatusCode: http.StatusConflict, Body: ErrorResponse{Errors: []string{"child"}}}
	}

	r := NewRPCRouter()
	RegisterErrorHandler(r, rootHandler)
	AddHandler(r, MethodGet, "/root", failHandler)
	Route(r, "/child", func(c *RPCRouter) {
		RegisterErrorHandler(c, childHandler)
		AddHandler(c, MethodGet, "/fail", failHandler)
	})
	Group(r, func(g *RPCRouter) {
		AddHandler(g, MethodGet, "/grouped", failHandler)
	})

	cases := map[string]int{"/root": http.StatusBadRequest, "/child/fail": http.StatusConflict, "/grouped": http.StatusBadRequest}
	for url, want := range cases {
		if code := serve(r, http.MethodGet, url).Code; code != want {
			t.Fatalf("expected %s to return %d, got %d", url, want, code)
		}
	}

	// The child's error type matches the root's, so the schema can be generated.
	generateSchema(t, r)
}

func TestChildRouterErrorHandlerWithDifferentTypeFailsGeneration(t *testing.T) {
	r := NewRPCRouter()
	Route(r, "/child", func(c *RPCRouter) {
		RegisterErrorHandler(c, func(req *http.Request, err *ErrorResponse) *HttpResponse[string] {
			return &HttpResponse[string]{Body: "child"}
		})
		AddHandler(c, MethodGet, "/fail", failHandler)
	})

	_, err := typescript.Convert(r.routerTypes)
	if err == nil || !strings.Contains(err.Error(), "only one error type") {
		t.Fatalf("expected an error about the error type, got %v", err)
	}
}

func TestRPCRouterIsHTTPHandler(t *testing.T) {
	r := NewRPCRouter()
	AddHandler(r, MethodGet, "/ok", okHandler)

	var handler http.Handler = r
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ok", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestSubRouterSupportsRouterFunctions(t *testing.T) {
	hits := 0
	sub := NewRPCSubRouter()
	AddMiddlewares(sub, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			hits++
			next.ServeHTTP(w, req)
		})
	})
	RegisterErrorHandler(sub, func(r *http.Request, err *ErrorResponse) *HttpResponse[ErrorResponse] {
		return &HttpResponse[ErrorResponse]{StatusCode: http.StatusConflict}
	})
	NotFound(sub, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusGone)
	})
	Route(sub, "/nested", func(n *RPCRouter) {
		AddHandler(n, MethodGet, "/{id}", okHandler)
	})
	Group(sub, func(g *RPCRouter) {
		AddHandler(g, MethodGet, "/fail", failHandler)
	})
	inner := NewRPCSubRouter()
	AddHandler(inner, MethodGet, "/deep", okHandler)
	Mount(sub, "/inner", inner)

	r := NewRPCRouter()
	Mount(r, "/sub", sub)

	cases := map[string]int{
		"/sub/nested/7":   http.StatusOK,
		"/sub/fail":       http.StatusConflict,
		"/sub/inner/deep": http.StatusOK,
		"/sub/missing":    http.StatusGone,
	}
	for url, want := range cases {
		if code := serve(r, http.MethodGet, url).Code; code != want {
			t.Fatalf("expected %s to return %d, got %d", url, want, code)
		}
	}
	if hits != len(cases) {
		t.Fatalf("expected the sub-router middleware to run %d times, ran %d times", len(cases), hits)
	}

	content := generateSchema(t, r)
	for _, want := range []string{`"/sub/nested/:id"`, `"/sub/fail"`, `"/sub/inner/deep"`} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected schema to contain %s, got %s", want, content)
		}
	}
}

func TestMountCycleFailsGeneration(t *testing.T) {
	a := NewRPCSubRouter()
	b := NewRPCSubRouter()
	Mount(a, "/b", b)
	Mount(b, "/a", a)

	r := NewRPCRouter()
	Mount(r, "/a", a)

	if _, err := typescript.Convert(r.routerTypes); err == nil || !strings.Contains(err.Error(), "mounted inside itself") {
		t.Fatalf("expected a mount cycle error, got %v", err)
	}
}

func TestSchemaNamesChirpcTypesByPackageName(t *testing.T) {
	content := generateSchema(t, NewRPCRouter())
	if !strings.Contains(content, "interface Chirpc__ErrorResponse") {
		t.Fatalf("expected the error type to be named Chirpc__ErrorResponse, got %s", content)
	}
}

func TestAnyRouterCanBeMounted(t *testing.T) {
	root := NewRPCRouter()
	RegisterErrorHandler(root, func(req *http.Request, err *ErrorResponse) *HttpResponse[string] {
		return &HttpResponse[string]{StatusCode: http.StatusTeapot, Body: "root"}
	})

	// This router has its own default error type, which must not conflict with the root's.
	child := NewRPCRouter()
	AddHandler(child, MethodGet, "/fail", failHandler)
	Mount(root, "/child", child)

	if code := serve(root, http.MethodGet, "/child/fail").Code; code != http.StatusTeapot {
		t.Fatalf("expected the root error handler to be used, got %d", code)
	}
	if content := generateSchema(t, root); !strings.Contains(content, `"/child/fail"`) {
		t.Fatalf("expected the mounted route in the schema, got %s", content)
	}
}

type customID struct {
	value [16]byte
}

func (id customID) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%x", id.value)), nil
}

func TestRegisterTSType(t *testing.T) {
	type withID struct {
		ID  customID   `json:"id"`
		IDs []customID `json:"ids"`
	}

	r := NewRPCRouter()
	sub := NewRPCRouter()
	RegisterTSType[customID](sub, "string")
	AddHandler(sub, MethodGet, "/item", func(req *http.Request) (*HttpResponse[withID], *ErrorResponse) {
		return nil, nil
	})
	Mount(r, "/api", sub)

	content := generateSchema(t, r)
	for _, want := range []string{"id: string;", "ids: string[];"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected schema to contain %q, got:\n%s", want, content)
		}
	}
	if strings.Contains(content, "CustomID") {
		t.Fatalf("expected no declaration for the overridden type, got:\n%s", content)
	}

	defer func() {
		if recovered := fmt.Sprint(recover()); !strings.Contains(recovered, "needs a TypeScript type") {
			t.Fatalf("expected a panic for an empty type, got %q", recovered)
		}
	}()
	RegisterTSType[customID](r, " ")
}
