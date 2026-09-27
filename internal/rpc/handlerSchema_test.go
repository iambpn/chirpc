package rpc

import (
	"net/http"
	"reflect"
	"testing"
)

func TestHandlerSchema_SetBodyType_AcceptsStructValue(t *testing.T) {
	r := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) {
		return nil, nil
	}

	schema, err := r.RegisterHandler("post", "/create", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	// set body type using a value
	schema.SetBodyType(testAddress{})

	if schema.bodyType == nil {
		t.Fatalf("expected bodyType to be set")
	}

	expected := reflect.TypeOf(testAddress{})
	if schema.bodyType != expected {
		t.Fatalf("expected bodyType %v, got %v", expected, schema.bodyType)
	}
}

func TestHandlerSchema_SetBodyType_AcceptsStructPointer(t *testing.T) {
	r := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }

	schema, err := r.RegisterHandler("post", "/create", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	// set body type using a pointer
	schema.SetBodyType(&testAddress{})

	if schema.bodyType == nil {
		t.Fatalf("expected bodyType to be set from pointer")
	}

	expected := reflect.TypeOf(testAddress{})
	if schema.bodyType != expected {
		t.Fatalf("expected bodyType %v, got %v", expected, schema.bodyType)
	}
}

func TestHandlerSchema_SetBodyType_PanicsForNonStructTypes(t *testing.T) {
	r := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }

	schema, err := r.RegisterHandler("post", "/create", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	testExpectPanic(t, "body type must be a struct or a pointer to a struct, but got int", func() {
		schema.SetBodyType(123)
	})
	testExpectPanic(t, "but got <nil>", func() {
		schema.SetBodyType(nil)
	})

	if schema.bodyType != nil {
		t.Fatalf("expected bodyType to remain nil when non-struct provided")
	}
}

func TestHandlerSchema_SetQueryType_AcceptsStructValue(t *testing.T) {
	r := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }

	schema, err := r.RegisterHandler("get", "/search", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	// set query type using a value
	schema.SetQueryType(testUserProfile{})

	if schema.queryType == nil {
		t.Fatalf("expected queryType to be set")
	}

	expected := reflect.TypeOf(testUserProfile{})
	if schema.queryType != expected {
		t.Fatalf("expected queryType %v, got %v", expected, schema.queryType)
	}
}

func TestHandlerSchema_SetQueryType_AcceptsStructPointer(t *testing.T) {
	r := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }

	schema, err := r.RegisterHandler("get", "/search", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	// set query type using a pointer
	schema.SetQueryType(&testUserProfile{})

	if schema.queryType == nil {
		t.Fatalf("expected queryType to be set from pointer")
	}

	expected := reflect.TypeOf(testUserProfile{})
	if schema.queryType != expected {
		t.Fatalf("expected queryType %v, got %v", expected, schema.queryType)
	}
}

func TestHandlerSchema_SetQueryType_PanicsForNonStructTypes(t *testing.T) {
	r := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }

	schema, err := r.RegisterHandler("get", "/search", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	testExpectPanic(t, "query type must be a struct or a pointer to a struct, but got string", func() {
		schema.SetQueryType("not a struct")
	})

	if schema.queryType != nil {
		t.Fatalf("expected queryType to remain nil when non-struct provided")
	}
}

func TestHandlerSchema_SetParamsType_StoresParamNames(t *testing.T) {
	s := &HandlerSchema{}
	s.SetParamsType([]string{"userId", "teamId"})

	expected := []string{"userId", "teamId"}
	if !reflect.DeepEqual(s.params, expected) {
		t.Fatalf("expected params %v, got %v", expected, s.params)
	}
}

func TestHandlerSchema_ParamNames_CombinesURLAndExtraParams(t *testing.T) {
	s := &HandlerSchema{}
	s.SetParamsType([]string{"teamId", "extra"})

	got := s.paramNames("/teams/{teamId}/users/{userId:[0-9]+}")
	expected := []string{"teamId", "userId", "extra"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected params %v, got %v", expected, got)
	}
}

func TestHandlerSchema_SetUrl_SetsUrlCorrectly(t *testing.T) {
	s := &HandlerSchema{}
	testUrl := "/api/users/:id"

	s.SetUrl(testUrl)

	if s.url != testUrl {
		t.Fatalf("expected url to be %q, got %q", testUrl, s.url)
	}
}

func TestHandlerSchema_SetUrl_UpdatesExistingUrl(t *testing.T) {
	s := &HandlerSchema{url: "/old/path"}
	newUrl := "/new/path"

	s.SetUrl(newUrl)

	if s.url != newUrl {
		t.Fatalf("expected url to be updated to %q, got %q", newUrl, s.url)
	}
}

func TestHandlerSchema_URL_ReturnsCorrectUrl(t *testing.T) {
	testUrl := "/api/posts/:postId"
	s := &HandlerSchema{url: testUrl}

	result := s.URL()

	if result != testUrl {
		t.Fatalf("expected URL() to return %q, got %q", testUrl, result)
	}
}

func TestHandlerSchema_URL_ReturnsEmptyStringWhenNotSet(t *testing.T) {
	s := &HandlerSchema{}

	result := s.URL()

	if result != "" {
		t.Fatalf("expected URL() to return empty string, got %q", result)
	}
}

func TestNewHandlerSchema_CreatesInstanceWithCorrectValues(t *testing.T) {
	method := "POST"
	url := "/api/create"
	returnType := reflect.TypeOf(testHttpResponse[string]{})

	schema := NewHandlerSchema(method, url, returnType)

	if schema == nil {
		t.Fatalf("expected NewHandlerSchema to return non-nil instance")
	}

	if schema.method != method {
		t.Fatalf("expected method to be %q, got %q", method, schema.method)
	}

	if schema.url != url {
		t.Fatalf("expected url to be %q, got %q", url, schema.url)
	}

	if schema.returnType != returnType {
		t.Fatalf("expected returnType to be %v, got %v", returnType, schema.returnType)
	}
}

func TestNewHandlerSchema_InitializesOtherFieldsAsZeroValues(t *testing.T) {
	method := "GET"
	url := "/api/list"
	returnType := reflect.TypeOf("")

	schema := NewHandlerSchema(method, url, returnType)

	if schema.bodyType != nil {
		t.Fatalf("expected bodyType to be nil, got %v", schema.bodyType)
	}

	if schema.queryType != nil {
		t.Fatalf("expected queryType to be nil, got %v", schema.queryType)
	}

	if schema.params != nil {
		t.Fatalf("expected params to be nil, got %v", schema.params)
	}
}
