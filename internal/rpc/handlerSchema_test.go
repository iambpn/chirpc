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

func TestHandlerSchema_SetBodyType_IgnoresNonStructTypes(t *testing.T) {
	r := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }

	schema, err := r.RegisterHandler("post", "/create", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	// attempt to set a non-struct body; should be ignored
	schema.SetBodyType(123)

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

func TestHandlerSchema_SetQueryType_IgnoresNonStructTypes(t *testing.T) {
	r := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }

	schema, err := r.RegisterHandler("get", "/search", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	// attempt to set a non-struct query; should be ignored
	schema.SetQueryType("not a struct")

	if schema.queryType != nil {
		t.Fatalf("expected queryType to remain nil when non-struct provided")
	}
}

func TestHandlerSchema_SetParamsType_GeneratesTypeScriptInterfaceFromStringSlice(t *testing.T) {
	s := &HandlerSchema{}
	s.SetParamsType([]string{"userId", "teamId"})

	if s.paramsType == "" {
		t.Fatalf("expected paramsType to be set")
	}

	expected := `{ "userId": string;"teamId": string; }`
	if s.paramsType != expected {
		t.Fatalf("expected paramsType %q, got %q", expected, s.paramsType)
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

	if schema.paramsType != "" {
		t.Fatalf("expected paramsType to be empty string, got %q", schema.paramsType)
	}

	if schema.StreamType() != "" {
		t.Fatalf("expected streamType to be empty string, got %q", schema.StreamType())
	}
}

func TestHandlerSchema_SetStreamType_SetsAndGetsStreamType(t *testing.T) {
	schema := NewHandlerSchema("GET", "/stream", reflect.TypeOf(testHttpResponse[string]{}))

	schema.SetStreamType("chunked")

	if schema.StreamType() != "chunked" {
		t.Fatalf("expected streamType to be %q, got %q", "chunked", schema.StreamType())
	}
}
