package rpc

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestRouterRpcSchemas_RegisterHandler_StoresMethodURLAndReturnType(t *testing.T) {
	r := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) {
		return nil, nil
	}

	if _, err := r.RegisterHandler("get", "/users", handler); err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	if len(r.schemas) != 1 {
		t.Fatalf("expected one registered type, got %d", len(r.schemas))
	}

	schema := r.schemas[0]
	if schema.method != "get" {
		t.Fatalf("expected method to be stored as get, got %s", schema.method)
	}

	if schema.url != "/users" {
		t.Fatalf("expected url /users, got %s", schema.url)
	}

	if schema.returnType == nil {
		t.Fatalf("expected return type to be captured")
	}
}

func TestRouterRpcSchemas_RegisterHandler_ReturnsErrorForNonFunctionHandler(t *testing.T) {
	r := NewRouterRpcSchemas()

	if _, err := r.RegisterHandler("post", "/invalid", 123); err == nil {
		t.Fatalf("expected error when registering non-function handler")
	}
}

func TestRouterRpcSchemas_RegisterHandler_AccumulatesMultipleHandlerSchemas(t *testing.T) {
	r := NewRouterRpcSchemas()

	handlerA := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }
	handlerB := func(*http.Request) (*testHttpResponse[int], error) { return nil, nil }

	if _, err := r.RegisterHandler("get", "/alpha", handlerA); err != nil {
		t.Fatalf("unexpected error registering handlerA: %v", err)
	}
	if _, err := r.RegisterHandler("post", "/beta", handlerB); err != nil {
		t.Fatalf("unexpected error registering handlerB: %v", err)
	}

	if len(r.schemas) != 2 {
		t.Fatalf("expected 2 registered handlers, got %d", len(r.schemas))
	}

	if r.schemas[0].method != "get" || r.schemas[0].url != "/alpha" {
		t.Fatalf("unexpected first schema contents: method=%s url=%s", r.schemas[0].method, r.schemas[0].url)
	}
	if r.schemas[1].method != "post" || r.schemas[1].url != "/beta" {
		t.Fatalf("unexpected second schema contents: method=%s url=%s", r.schemas[1].method, r.schemas[1].url)
	}
}

func TestRouterRpcSchemas_RegisterHandler_ReturnsModifiableSchemaReference(t *testing.T) {
	r := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }

	schema, err := r.RegisterHandler("put", "/gamma", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	if len(r.schemas) != 1 {
		t.Fatalf("expected 1 registered handler, got %d", len(r.schemas))
	}

	if schema != r.schemas[0] {
		t.Fatalf("returned schema pointer should match stored schema")
	}

	schema.SetBodyType(testAddress{})

	if r.schemas[0].bodyType == nil || r.schemas[0].bodyType != reflect.TypeOf(testAddress{}) {
		t.Fatalf("expected bodyType to propagate to stored schema")
	}
}

func TestRouterRpcSchemas_RegisterHandler_ReturnsErrorForInvalidHandlerSignature(t *testing.T) {
	r := NewRouterRpcSchemas()

	invalid := func() {}

	if _, err := r.RegisterHandler("delete", "/delta", invalid); err == nil {
		t.Fatalf("expected error registering handler without return value")
	}

	if len(r.schemas) != 0 {
		t.Fatalf("expected no handlers registered after error, got %d", len(r.schemas))
	}
}

func TestRouterRpcSchemas_Routes_ResolvesMountedURLsAndParams(t *testing.T) {
	root := NewRouterRpcSchemas()
	child := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }
	errorHandler := func(*http.Request, error) *testHttpResponse[int] { return nil }

	if err := root.RegisterErrorHandler(errorHandler); err != nil {
		t.Fatalf("unexpected error registering error handler: %v", err)
	}
	schema, err := child.RegisterHandler("GET", "/posts/{postId:[0-9]+}", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}
	schema.SetBodyType(testCreateReq{})
	root.Mount("/users/{userId}", child)

	routes, err := root.Routes()
	if err != nil {
		t.Fatalf("Routes returned error: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected the error handler and one route, got %d", len(routes))
	}

	if routes[0].Method != "ERROR_HANDLER" || routes[0].URL != "/" {
		t.Fatalf("expected the error handler first, got %+v", routes[0])
	}

	route := routes[1]
	if route.URL != "/users/{userId}/posts/{postId:[0-9]+}" {
		t.Fatalf("unexpected URL %q", route.URL)
	}
	if !reflect.DeepEqual(route.Params, []string{"userId", "postId"}) {
		t.Fatalf("unexpected params %v", route.Params)
	}
	if route.Body != reflect.TypeOf(testCreateReq{}) {
		t.Fatalf("unexpected body type %v", route.Body)
	}
}

func TestRouterRpcSchemas_Routes_RejectsMountCycles(t *testing.T) {
	a := NewRouterRpcSchemas()
	b := NewRouterRpcSchemas()
	a.Mount("/b", b)
	b.Mount("/a", a)

	_, err := a.Routes()
	if err == nil || !strings.Contains(err.Error(), "mounted inside itself") {
		t.Fatalf("expected a mount cycle error, got %v", err)
	}
}

func TestRouterRpcSchemas_Routes_UsesDefaultErrorHandlerOnlyForRoot(t *testing.T) {
	defaultType := func(*http.Request, error) *testHttpResponse[int] { return nil }
	customType := func(*http.Request, error) *testHttpResponse[string] { return nil }

	root := NewRouterRpcSchemas()
	child := NewRouterRpcSchemas()
	_ = root.SetDefaultErrorHandler(defaultType)
	_ = child.SetDefaultErrorHandler(defaultType)
	_ = root.RegisterErrorHandler(customType)
	root.Mount("/child", child)

	routes, err := root.Routes()
	if err != nil {
		t.Fatalf("expected the child's default error type to be ignored, got %v", err)
	}
	if routes[0].Response != reflect.TypeOf(testHttpResponse[string]{}) {
		t.Fatalf("expected the registered error type to win over the default, got %v", routes[0].Response)
	}
}

func TestRouterRpcSchemas_TypeOverrides(t *testing.T) {
	type id struct{}

	root := NewRouterRpcSchemas()
	child := NewRouterRpcSchemas()
	root.Mount("/child", child)
	child.SetTypeOverride(reflect.TypeOf(id{}), "string")

	overrides, err := root.TypeOverrides()
	if err != nil {
		t.Fatalf("TypeOverrides returned error: %v", err)
	}
	if overrides[reflect.TypeOf(id{})] != "string" {
		t.Fatalf("expected the child's override, got %v", overrides)
	}

	root.SetTypeOverride(reflect.TypeOf(id{}), "number")
	if _, err := root.TypeOverrides(); err == nil || !strings.Contains(err.Error(), "registered as both") {
		t.Fatalf("expected a conflict error, got %v", err)
	}
}
