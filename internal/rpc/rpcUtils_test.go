package rpc

import (
	"reflect"
	"strings"
	"testing"
)

func TestConvertURLPattern_ConvertsSingleBracedParameterToColonSyntax(t *testing.T) {
	in := "/users/{id}"
	out := convertURLPattern(in)
	if out != "/users/:id" {
		t.Fatalf("unexpected conversion: %q -> %q", in, out)
	}
}

func TestConvertURLPattern_ConvertsMultipleBracedParametersToColonSyntax(t *testing.T) {
	in := "/a/{id}/b/{slug}"
	out := convertURLPattern(in)
	if out != "/a/:id/b/:slug" {
		t.Fatalf("unexpected conversion: %q -> %q", in, out)
	}
}

func TestConvertURLPattern_HandlesNestedBracesWithinParameter(t *testing.T) {
	in := "/x/{a{b}c}/y"
	out := convertURLPattern(in)
	if out != "/x/:a{b}c/y" {
		t.Fatalf("unexpected conversion with nested braces: %q -> %q", in, out)
	}
}

func TestExtractReturnType_ExtractsStructTypeFromFunctionSignature(t *testing.T) {
	handler := func() (*testHttpResponse[string], error) { return nil, nil }

	retType, err := extractReturnType(reflect.TypeOf(handler))
	if err != nil {
		t.Fatalf("extractReturnType returned error: %v", err)
	}

	if retType.Kind() != reflect.Struct {
		t.Fatalf("expected struct kind, got %s", retType.Kind())
	}

	expectedType := reflect.TypeOf(testHttpResponse[string]{})
	if retType != expectedType {
		t.Fatalf("expected return type %v, got %v", expectedType, retType)
	}
}

func TestExtractReturnType_ReturnsErrorWhenInputIsNotAFunction(t *testing.T) {
	_, err := extractReturnType(reflect.TypeOf(42))
	if err == nil || !strings.Contains(err.Error(), "not a function") {
		t.Fatalf("expected not a function error, got %v", err)
	}
}

func TestExtractReturnType_ReturnsErrorWhenFunctionHasNoReturnValues(t *testing.T) {
	noReturnFn := func() {}

	_, err := extractReturnType(reflect.TypeOf(noReturnFn))
	if err == nil || !strings.Contains(err.Error(), "at least one return") {
		t.Fatalf("expected missing return values error, got %v", err)
	}
}

func TestSliceToTsInf_GeneratesTypeScriptInterfaceFromStringSlice(t *testing.T) {
	t.Run("returns never for empty slice", func(t *testing.T) {
		got := sliceToTsInf([]string{})
		if got != "never" {
			t.Fatalf("expected never, got %q", got)
		}
	})

	t.Run("generates interface with string properties for non-empty slice", func(t *testing.T) {
		got := sliceToTsInf([]string{"id", "postId"})
		expected := `{ "id": string;"postId": string; }`
		if got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})

	t.Run("generates interface with single property", func(t *testing.T) {
		got := sliceToTsInf([]string{"userId"})
		expected := `{ "userId": string; }`
		if got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})
}

func TestConvertURLPattern_HandlesNoParameters(t *testing.T) {
	in := "/users/list"
	out := convertURLPattern(in)
	if out != "/users/list" {
		t.Fatalf("expected no change for URL without parameters: %q -> %q", in, out)
	}
}

func TestConvertURLPattern_HandlesEmptyString(t *testing.T) {
	in := ""
	out := convertURLPattern(in)
	if out != "" {
		t.Fatalf("expected empty string, got %q", out)
	}
}

func TestConvertURLPattern_HandlesRootPath(t *testing.T) {
	in := "/"
	out := convertURLPattern(in)
	if out != "/" {
		t.Fatalf("expected root path unchanged: %q -> %q", in, out)
	}
}

func TestConvertURLPattern_HandlesConsecutiveParameters(t *testing.T) {
	in := "/api/{category}/{id}"
	out := convertURLPattern(in)
	if out != "/api/:category/:id" {
		t.Fatalf("unexpected conversion: %q -> %q", in, out)
	}
}

func TestExtractReturnType_HandlesNonPointerReturnType(t *testing.T) {
	handler := func() (testHttpResponse[string], error) { return testHttpResponse[string]{}, nil }

	retType, err := extractReturnType(reflect.TypeOf(handler))
	if err != nil {
		t.Fatalf("extractReturnType returned error: %v", err)
	}

	if retType.Kind() != reflect.Struct {
		t.Fatalf("expected struct kind, got %s", retType.Kind())
	}

	expectedType := reflect.TypeOf(testHttpResponse[string]{})
	if retType != expectedType {
		t.Fatalf("expected return type %v, got %v", expectedType, retType)
	}
}

func TestExtractReturnType_HandlesPointerToFunction(t *testing.T) {
	handler := func() (*testHttpResponse[string], error) { return nil, nil }
	handlerPtr := &handler

	retType, err := extractReturnType(reflect.TypeOf(handlerPtr))
	if err != nil {
		t.Fatalf("extractReturnType returned error: %v", err)
	}

	expectedType := reflect.TypeOf(testHttpResponse[string]{})
	if retType != expectedType {
		t.Fatalf("expected return type %v, got %v", expectedType, retType)
	}
}

func TestExtractReturnType_HandlesMultipleReturnValues(t *testing.T) {
	handler := func() (*testHttpResponse[string], error, bool) { return nil, nil, false }

	retType, err := extractReturnType(reflect.TypeOf(handler))
	if err != nil {
		t.Fatalf("extractReturnType returned error: %v", err)
	}

	expectedType := reflect.TypeOf(testHttpResponse[string]{})
	if retType != expectedType {
		t.Fatalf("expected first return type %v, got %v", expectedType, retType)
	}
}

func TestBuildGoToTsSchema_CreatesSchemaWithValidHandler(t *testing.T) {
	handler := func() (*testHttpResponse[testUserProfile], error) { return nil, nil }

	schema, err := BuildGoToTsSchema("GET", "/users/{id}", handler)
	if err != nil {
		t.Fatalf("BuildGoToTsSchema returned error: %v", err)
	}

	if schema == nil {
		t.Fatal("expected non-nil schema")
	}

	if schema.method != "GET" {
		t.Fatalf("expected method GET, got %s", schema.method)
	}

	if schema.url != "/users/{id}" {
		t.Fatalf("expected URL /users/{id}, got %s", schema.url)
	}

	expectedType := reflect.TypeOf(testHttpResponse[testUserProfile]{})
	if schema.returnType != expectedType {
		t.Fatalf("expected return type %v, got %v", expectedType, schema.returnType)
	}
}

func TestBuildGoToTsSchema_ReturnsErrorForInvalidHandler(t *testing.T) {
	_, err := BuildGoToTsSchema("POST", "/users", "not a function")
	if err == nil || !strings.Contains(err.Error(), "not a function") {
		t.Fatalf("expected not a function error, got %v", err)
	}
}

func TestBuildGoToTsSchema_ReturnsErrorForHandlerWithNoReturnValue(t *testing.T) {
	handler := func() {}

	_, err := BuildGoToTsSchema("DELETE", "/users/{id}", handler)
	if err == nil || !strings.Contains(err.Error(), "at least one return") {
		t.Fatalf("expected missing return values error, got %v", err)
	}
}

func TestBuildGoToTsSchema_HandlesPointerHandler(t *testing.T) {
	handler := func() (*testHttpResponse[testCreateReq], error) { return nil, nil }
	handlerPtr := &handler

	schema, err := BuildGoToTsSchema("POST", "/api/create", handlerPtr)
	if err != nil {
		t.Fatalf("BuildGoToTsSchema returned error: %v", err)
	}

	if schema == nil {
		t.Fatal("expected non-nil schema")
	}

	expectedType := reflect.TypeOf(testHttpResponse[testCreateReq]{})
	if schema.returnType != expectedType {
		t.Fatalf("expected return type %v, got %v", expectedType, schema.returnType)
	}
}
