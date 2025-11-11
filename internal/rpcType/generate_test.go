package rpcType

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func resetTypeRegistry() {
	types = nil
}

type httpResponse[T any] struct {
	StatusCode int
	Body       T
	Headers    map[string]string
}

func VerifyTsTypes(t *testing.T, types string, expectedTypes string) {
	t.Helper()

	replacer := strings.NewReplacer("\n", " ", "\t", "")

	types = strings.TrimSpace(replacer.Replace(types))
	expectedTypes = strings.TrimSpace(replacer.Replace(expectedTypes))

	if types != expectedTypes {
		t.Fatalf("expected output \n%s\n, got \n%s\n", expectedTypes, types)
	}
}

func TestExtractReturnType_ReturnsUnderlyingStruct(t *testing.T) {
	handler := func() (*httpResponse[string], error) { return nil, nil }

	retType, err := extractReturnType(reflect.TypeOf(handler))
	if err != nil {
		t.Fatalf("extractReturnType returned error: %v", err)
	}

	if retType.Kind() != reflect.Struct {
		t.Fatalf("expected struct kind, got %s", retType.Kind())
	}

	expectedType := reflect.TypeOf(httpResponse[string]{})
	if retType != expectedType {
		t.Fatalf("expected return type %v, got %v", expectedType, retType)
	}
}

func TestExtractReturnType_ErrorsOnNonFunctionInput(t *testing.T) {
	_, err := extractReturnType(reflect.TypeOf(42))
	if err == nil || !strings.Contains(err.Error(), "not a function") {
		t.Fatalf("expected not a function error, got %v", err)
	}
}

func TestExtractReturnType_ErrorsOnFunctionWithoutReturns(t *testing.T) {
	noReturnFn := func() {}

	_, err := extractReturnType(reflect.TypeOf(noReturnFn))
	if err == nil || !strings.Contains(err.Error(), "at least one return") {
		t.Fatalf("expected missing return values error, got %v", err)
	}
}

func TestRegisterHandler_StoresSchemaInformation(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	handler := func(*http.Request) (*httpResponse[string], error) {
		return nil, nil
	}

	if _, err := RegisterHandler("get", "/users", handler); err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	if len(types) != 1 {
		t.Fatalf("expected one registered type, got %d", len(types))
	}

	schema := types[0]
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

func TestRegisterHandler_ErrorsOnNonFunction(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	if _, err := RegisterHandler("post", "/invalid", 123); err == nil {
		t.Fatalf("expected error when registering non-function handler")
	}
}

func TestConvertToTs_GeneratesSingleHandlerSchema(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	handler := func(*http.Request) (*httpResponse[string], error) {
		return nil, nil
	}

	if _, err := RegisterHandler("get", "/status", handler); err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	out, err := ConvertToTs()
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}

	expectedString := `
		export type ApiSchema = { "GET": { "/status": { response: string; }; }; };
	`
	VerifyTsTypes(t, out, expectedString)
}

type Address struct {
	Line1 string
	Zip   int
}

type UserProfile struct {
	Name    string
	Primary Address
}

type TeamPayload struct {
	Owner   UserProfile
	Members []UserProfile
}

func TestConvertToTs_HandlesNestedTypesAcrossHandlers(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	userHandler := func(*http.Request) (*httpResponse[UserProfile], error) {
		return nil, nil
	}

	teamHandler := func(*http.Request) (*httpResponse[TeamPayload], error) {
		return nil, nil
	}

	if _, err := RegisterHandler("get", "/users/{id}", userHandler); err != nil {
		t.Fatalf("registering user handler failed: %v", err)
	}

	if _, err := RegisterHandler("post", "/teams", teamHandler); err != nil {
		t.Fatalf("registering team handler failed: %v", err)
	}

	out, err := ConvertToTs()
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}

	expectedString := `
		interface RpcType__Address {
			Line1:string;
			Zip:number;
		}
		interface RpcType__UserProfile {
			Name:string;
			Primary:RpcType__Address;
		}
		interface RpcType__TeamPayload {
			Owner:RpcType__UserProfile;
			Members:RpcType__UserProfile[];
		}
		export type ApiSchema = {
			"GET": {
				"/users/:id": {
					response: RpcType__UserProfile;
				};
			};
			"POST": {
				"/teams": {
					response: RpcType__TeamPayload;
				};
			};
		};
	`

	VerifyTsTypes(t, out, expectedString)
}

func TestSetBodyType_SetsStructType(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	handler := func(*http.Request) (*httpResponse[string], error) {
		return nil, nil
	}

	schema, err := RegisterHandler("post", "/create", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	// set body type using a value
	SetBodyType(schema, Address{})

	if schema.bodyType == nil {
		t.Fatalf("expected bodyType to be set")
	}

	expected := reflect.TypeOf(Address{})
	if schema.bodyType != expected {
		t.Fatalf("expected bodyType %v, got %v", expected, schema.bodyType)
	}
}

func TestSetBodyType_AcceptsPointerToStruct(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	handler := func(*http.Request) (*httpResponse[string], error) { return nil, nil }

	schema, err := RegisterHandler("post", "/create", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	// set body type using a pointer
	SetBodyType(schema, &Address{})

	if schema.bodyType == nil {
		t.Fatalf("expected bodyType to be set from pointer")
	}

	expected := reflect.TypeOf(Address{})
	if schema.bodyType != expected {
		t.Fatalf("expected bodyType %v, got %v", expected, schema.bodyType)
	}
}

func TestSetBodyType_IgnoresNonStruct(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	handler := func(*http.Request) (*httpResponse[string], error) { return nil, nil }

	schema, err := RegisterHandler("post", "/create", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	// attempt to set a non-struct body; should be ignored
	SetBodyType(schema, 123)

	if schema.bodyType != nil {
		t.Fatalf("expected bodyType to remain nil when non-struct provided")
	}
}

func TestSetQueryType_SetsStructType(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	handler := func(*http.Request) (*httpResponse[string], error) { return nil, nil }

	schema, err := RegisterHandler("get", "/search", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	// set query type using a value
	SetQueryType(schema, UserProfile{})

	if schema.queryType == nil {
		t.Fatalf("expected queryType to be set")
	}

	expected := reflect.TypeOf(UserProfile{})
	if schema.queryType != expected {
		t.Fatalf("expected queryType %v, got %v", expected, schema.queryType)
	}
}

func TestSetQueryType_AcceptsPointerToStruct(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	handler := func(*http.Request) (*httpResponse[string], error) { return nil, nil }

	schema, err := RegisterHandler("get", "/search", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	// set query type using a pointer
	SetQueryType(schema, &UserProfile{})

	if schema.queryType == nil {
		t.Fatalf("expected queryType to be set from pointer")
	}

	expected := reflect.TypeOf(UserProfile{})
	if schema.queryType != expected {
		t.Fatalf("expected queryType %v, got %v", expected, schema.queryType)
	}
}

func TestSetQueryType_IgnoresNonStruct(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	handler := func(*http.Request) (*httpResponse[string], error) { return nil, nil }

	schema, err := RegisterHandler("get", "/search", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	// attempt to set a non-struct query; should be ignored
	SetQueryType(schema, "not a struct")

	if schema.queryType != nil {
		t.Fatalf("expected queryType to remain nil when non-struct provided")
	}
}

func TestSliceToTsInf_EmptyAndNonEmpty(t *testing.T) {
	t.Run("empty returns never", func(t *testing.T) {
		got := sliceToTsInf([]string{})
		if got != "never" {
			t.Fatalf("expected never, got %q", got)
		}
	})

	t.Run("multiple slugs forms interface", func(t *testing.T) {
		got := sliceToTsInf([]string{"id", "postId"})
		expected := `{ "id": string;"postId": string; }`
		if got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})
}

func TestSetParamsType_SetsParamsInterfaceString(t *testing.T) {
	s := &TsGoSchema{}
	SetParamsType(s, []string{"userId", "teamId"})

	if s.paramsType == "" {
		t.Fatalf("expected paramsType to be set")
	}

	expected := `{ "userId": string;"teamId": string; }`
	if s.paramsType != expected {
		t.Fatalf("expected paramsType %q, got %q", expected, s.paramsType)
	}
}

type CreateReq struct {
	Name   string
	TagIds []int
}
type SearchQ struct {
	Filter string
	Limit  int
}

func TestConvertToTs_IncludesBodyQueryAndParam(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	handler := func(*http.Request) (*httpResponse[string], error) { return nil, nil }

	schema, err := RegisterHandler("post", "/users/{userId}", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	SetBodyType(schema, CreateReq{})
	SetQueryType(schema, SearchQ{})
	SetParamsType(schema, []string{"userId"})

	out, err := ConvertToTs()
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}

	expected := `
		export type ApiSchema = {
			"POST": {
				"/users/:userId": {
					params: { "userId": string; };
					query?: { Filter:string; Limit:number; };
					body: { Name:string; TagIds:number[]; };
					response: string;
				};
			};
		};
	`
	VerifyTsTypes(t, out, expected)
}

func TestRegisterHandler_AppendsMultipleSchemas(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	handlerA := func(*http.Request) (*httpResponse[string], error) { return nil, nil }
	handlerB := func(*http.Request) (*httpResponse[int], error) { return nil, nil }

	if _, err := RegisterHandler("get", "/alpha", handlerA); err != nil {
		t.Fatalf("unexpected error registering handlerA: %v", err)
	}
	if _, err := RegisterHandler("post", "/beta", handlerB); err != nil {
		t.Fatalf("unexpected error registering handlerB: %v", err)
	}

	if len(types) != 2 {
		t.Fatalf("expected 2 registered handlers, got %d", len(types))
	}

	if types[0].method != "get" || types[0].url != "/alpha" {
		t.Fatalf("unexpected first schema contents: method=%s url=%s", types[0].method, types[0].url)
	}
	if types[1].method != "post" || types[1].url != "/beta" {
		t.Fatalf("unexpected second schema contents: method=%s url=%s", types[1].method, types[1].url)
	}
}

func TestRegisterHandler_ReturnsStoredSchemaReference(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	handler := func(*http.Request) (*httpResponse[string], error) { return nil, nil }

	schema, err := RegisterHandler("put", "/gamma", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	if len(types) != 1 {
		t.Fatalf("expected 1 registered handler, got %d", len(types))
	}

	if schema != types[0] {
		t.Fatalf("returned schema pointer should match stored schema")
	}

	SetBodyType(schema, Address{})

	if types[0].bodyType == nil || types[0].bodyType != reflect.TypeOf(Address{}) {
		t.Fatalf("expected bodyType to propagate to stored schema")
	}
}

func TestRegisterHandler_PropagatesBuildErrors(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	invalid := func() {}

	if _, err := RegisterHandler("delete", "/delta", invalid); err == nil {
		t.Fatalf("expected error registering handler without return value")
	}

	if len(types) != 0 {
		t.Fatalf("expected no handlers registered after error, got %d", len(types))
	}
}
