package rpc

import (
	"strings"
	"testing"
)

func TestNewEndpointSchema_InitializesWithExportFlagAndEmptyTypesMap(t *testing.T) {
	rt := NewEndpointSchema(true)

	if !rt.shouldExport {
		t.Fatalf("expected shouldExport to be true")
	}

	if rt.types == nil {
		t.Fatalf("expected types map to be initialized")
	}

	if rt.types.Len() != 0 {
		t.Fatalf("expected types map to be empty, got %d entries", rt.types.Len())
	}
}

func TestAddRpcSchema_CreatesMethodBucketAndStoresSchemaByURL(t *testing.T) {
	rt := NewEndpointSchema(false)

	schema := RpcSchema{Param: "FooParam"}
	rt.AddRpcSchema("get", "/items", schema)

	methodGroup, ok := rt.types.Get("get")
	if !ok {
		t.Fatalf("expected method bucket to be initialized")
	}

	stored, ok := methodGroup.Get("/items")
	if !ok {
		t.Fatalf("expected url entry to be stored")
	}

	if stored != schema {
		t.Fatalf("stored schema mismatch: %#v", stored)
	}
}

func TestAddRpcSchema_OverwritesExistingSchemaForSameMethodAndURL(t *testing.T) {
	rt := NewEndpointSchema(false)

	first := RpcSchema{Param: "First"}
	second := RpcSchema{Param: "Second", Response: "SecondResponse"}

	rt.AddRpcSchema("get", "/items", first)
	rt.AddRpcSchema("get", "/items", second)

	methodGroup, _ := rt.types.Get("get")
	stored, _ := methodGroup.Get("/items")
	if stored != second {
		t.Fatalf("expected latest schema to be stored, got %#v", stored)
	}
}

func TestEndpointSchema_String_GeneratesExportedEmptyApiSchema(t *testing.T) {
	rt := NewEndpointSchema(true)

	expected := "export type ApiSchema = { };"
	if got := rt.String(); got != expected {
		t.Fatalf("unexpected string output.\nexpected: %q\n     got: %q", expected, got)
	}
}

func TestEndpointSchema_String_GeneratesCompleteSchemaWithAllFields(t *testing.T) {
	rt := NewEndpointSchema(false)
	schema := RpcSchema{
		Param:    "SomeParam",
		Query:    "SomeQuery",
		Body:     "SomeBody",
		Response: "SomeResponse",
	}

	rt.AddRpcSchema("post", "/users", schema)

	expected := "type ApiSchema = { \"POST\": { \"/users\": { params: SomeParam; query?: SomeQuery; body: SomeBody; response: SomeResponse; }; }; };"
	if got := rt.String(); got != expected {
		t.Fatalf("unexpected string output.\nexpected: %q\n     got: %q", expected, got)
	}
}

func TestEndpointSchema_String_OmitsEmptyFieldsAndDefaultsToVoidResponse(t *testing.T) {
	rt := NewEndpointSchema(false)
	rt.AddRpcSchema("get", "/empty", RpcSchema{})

	got := rt.String()

	if strings.Contains(got, "param:") {
		t.Fatalf("expected param field to be omitted, got %q", got)
	}

	if strings.Contains(got, "query?:") {
		t.Fatalf("expected query field to be omitted, got %q", got)
	}

	if strings.Contains(got, "body:") {
		t.Fatalf("expected body field to be omitted, got %q", got)
	}

	if !strings.Contains(got, "response: void;") {
		t.Fatalf("expected response void fallback, got %q", got)
	}

	if !strings.Contains(got, "\"GET\": {") {
		t.Fatalf("expected method to be uppercased, got %q", got)
	}
}

func TestEndpointSchema_String_IncludesExportKeywordWhenExportFlagIsTrue(t *testing.T) {
	rt := NewEndpointSchema(true)
	rt.AddRpcSchema("get", "/ping", RpcSchema{Response: "Pong"})
	got := rt.String()
	if !strings.HasPrefix(got, "export type ApiSchema") {
		t.Fatalf("expected export prefix, got: %q", got)
	}
}

func TestEndpointSchema_String_ConvertsBracedURLParametersToColonSyntax(t *testing.T) {
	rt := NewEndpointSchema(false)
	rt.AddRpcSchema("get", "/foo/{id}/bar", RpcSchema{Response: "R"})
	got := rt.String()
	if !strings.Contains(got, "\"/foo/:id/bar\"") {
		t.Fatalf("expected converted url pattern in output, got: %q", got)
	}
}

func TestEndpointSchema_String_PreservesMethodAndURLInsertionOrder(t *testing.T) {
	rt := NewEndpointSchema(false)
	rt.AddRpcSchema("get", "/one", RpcSchema{Response: "R1"})
	rt.AddRpcSchema("get", "/two", RpcSchema{Response: "R2"})
	rt.AddRpcSchema("post", "/p1", RpcSchema{Response: "R3"})
	rt.AddRpcSchema("post", "/p2", RpcSchema{Response: "R4"})

	got := rt.String()

	idxGET := strings.Index(got, "\"GET\": {")
	idxPOST := strings.Index(got, "\"POST\": {")
	if idxGET == -1 || idxPOST == -1 || idxGET > idxPOST {
		t.Fatalf("expected GET to appear before POST, got: %q", got)
	}

	idxOne := strings.Index(got, "\"/one\": {")
	idxTwo := strings.Index(got, "\"/two\": {")
	if idxOne == -1 || idxTwo == -1 || idxOne > idxTwo {
		t.Fatalf("expected /one to appear before /two, got: %q", got)
	}

	idxP1 := strings.Index(got, "\"/p1\": {")
	idxP2 := strings.Index(got, "\"/p2\": {")
	if idxP1 == -1 || idxP2 == -1 || idxP1 > idxP2 {
		t.Fatalf("expected /p1 to appear before /p2, got: %q", got)
	}
}

func TestAddRpcSchema_HandlesMultipleMethodsForSameURL(t *testing.T) {
	rt := NewEndpointSchema(false)

	getSchema := RpcSchema{Response: "GetResponse"}
	postSchema := RpcSchema{Body: "PostBody", Response: "PostResponse"}

	rt.AddRpcSchema("get", "/resource", getSchema)
	rt.AddRpcSchema("post", "/resource", postSchema)

	getMethodGroup, ok := rt.types.Get("get")
	if !ok {
		t.Fatalf("expected GET method bucket to exist")
	}
	storedGet, ok := getMethodGroup.Get("/resource")
	if !ok || storedGet != getSchema {
		t.Fatalf("GET schema not stored correctly")
	}

	postMethodGroup, ok := rt.types.Get("post")
	if !ok {
		t.Fatalf("expected POST method bucket to exist")
	}
	storedPost, ok := postMethodGroup.Get("/resource")
	if !ok || storedPost != postSchema {
		t.Fatalf("POST schema not stored correctly")
	}
}

func TestEndpointSchema_String_HandlesMultipleMethods(t *testing.T) {
	rt := NewEndpointSchema(false)

	rt.AddRpcSchema("get", "/items", RpcSchema{Response: "ItemList"})
	rt.AddRpcSchema("post", "/items", RpcSchema{Body: "CreateItem", Response: "Item"})
	rt.AddRpcSchema("delete", "/items/{id}", RpcSchema{Param: "ItemId", Response: "void"})

	got := rt.String()

	if !strings.Contains(got, "\"GET\": {") {
		t.Fatalf("expected GET method in output, got: %q", got)
	}
	if !strings.Contains(got, "\"POST\": {") {
		t.Fatalf("expected POST method in output, got: %q", got)
	}
	if !strings.Contains(got, "\"DELETE\": {") {
		t.Fatalf("expected DELETE method in output, got: %q", got)
	}
}

func TestEndpointSchema_String_HandlesOnlyParam(t *testing.T) {
	rt := NewEndpointSchema(false)
	rt.AddRpcSchema("get", "/users/{id}", RpcSchema{Param: "UserId"})

	got := rt.String()

	if !strings.Contains(got, "params: UserId;") {
		t.Fatalf("expected params field, got: %q", got)
	}
	if strings.Contains(got, "query?:") {
		t.Fatalf("expected no query field, got: %q", got)
	}
	if strings.Contains(got, "body:") {
		t.Fatalf("expected no body field, got: %q", got)
	}
	if !strings.Contains(got, "response: void;") {
		t.Fatalf("expected void response, got: %q", got)
	}
}

func TestEndpointSchema_String_HandlesOnlyQuery(t *testing.T) {
	rt := NewEndpointSchema(false)
	rt.AddRpcSchema("get", "/search", RpcSchema{Query: "SearchQuery"})

	got := rt.String()

	if strings.Contains(got, "params:") {
		t.Fatalf("expected no params field, got: %q", got)
	}
	if !strings.Contains(got, "query?: SearchQuery;") {
		t.Fatalf("expected query field, got: %q", got)
	}
	if strings.Contains(got, "body:") {
		t.Fatalf("expected no body field, got: %q", got)
	}
}

func TestEndpointSchema_String_HandlesOnlyBody(t *testing.T) {
	rt := NewEndpointSchema(false)
	rt.AddRpcSchema("post", "/data", RpcSchema{Body: "DataPayload"})

	got := rt.String()

	if strings.Contains(got, "params:") {
		t.Fatalf("expected no params field, got: %q", got)
	}
	if strings.Contains(got, "query?:") {
		t.Fatalf("expected no query field, got: %q", got)
	}
	if !strings.Contains(got, "body: DataPayload;") {
		t.Fatalf("expected body field, got: %q", got)
	}
}

func TestEndpointSchema_String_HandlesParamQueryAndBody(t *testing.T) {
	rt := NewEndpointSchema(false)
	schema := RpcSchema{
		Param:    "RouteParam",
		Query:    "QueryParam",
		Body:     "BodyPayload",
		Response: "Result",
	}
	rt.AddRpcSchema("put", "/items/{id}", schema)

	got := rt.String()

	if !strings.Contains(got, "params: RouteParam;") {
		t.Fatalf("expected params field, got: %q", got)
	}
	if !strings.Contains(got, "query?: QueryParam;") {
		t.Fatalf("expected query field, got: %q", got)
	}
	if !strings.Contains(got, "body: BodyPayload;") {
		t.Fatalf("expected body field, got: %q", got)
	}
	if !strings.Contains(got, "response: Result;") {
		t.Fatalf("expected response field, got: %q", got)
	}
}

func TestEndpointSchema_String_HandlesLowercaseMethodNames(t *testing.T) {
	rt := NewEndpointSchema(false)
	rt.AddRpcSchema("patch", "/resource", RpcSchema{Response: "R"})

	got := rt.String()

	if !strings.Contains(got, "\"PATCH\": {") {
		t.Fatalf("expected method to be uppercased to PATCH, got: %q", got)
	}
	if strings.Contains(got, "\"patch\": {") {
		t.Fatalf("method should not remain lowercase, got: %q", got)
	}
}

func TestEndpointSchema_String_HandlesMultipleBracedParameters(t *testing.T) {
	rt := NewEndpointSchema(false)
	rt.AddRpcSchema("get", "/org/{orgId}/team/{teamId}", RpcSchema{Response: "Team"})

	got := rt.String()

	if !strings.Contains(got, "\"/org/:orgId/team/:teamId\"") {
		t.Fatalf("expected multiple braced parameters converted to colon syntax, got: %q", got)
	}
}

func TestEndpointSchema_String_GeneratesNonExportedTypeByDefault(t *testing.T) {
	rt := NewEndpointSchema(false)
	rt.AddRpcSchema("get", "/test", RpcSchema{Response: "Test"})

	got := rt.String()

	if strings.HasPrefix(got, "export type") {
		t.Fatalf("expected non-exported type, got: %q", got)
	}
	if !strings.HasPrefix(got, "type ApiSchema") {
		t.Fatalf("expected type to start with 'type ApiSchema', got: %q", got)
	}
}

func TestAddRpcSchema_HandlesEmptyMethodString(t *testing.T) {
	rt := NewEndpointSchema(false)
	rt.AddRpcSchema("", "/test", RpcSchema{Response: "Test"})

	methodGroup, ok := rt.types.Get("")
	if !ok {
		t.Fatalf("expected empty method key to be stored")
	}

	stored, ok := methodGroup.Get("/test")
	if !ok {
		t.Fatalf("expected schema to be stored under empty method")
	}
	if stored.Response != "Test" {
		t.Fatalf("stored schema mismatch")
	}
}

func TestAddRpcSchema_HandlesEmptyURLString(t *testing.T) {
	rt := NewEndpointSchema(false)
	rt.AddRpcSchema("get", "", RpcSchema{Response: "Test"})

	methodGroup, ok := rt.types.Get("get")
	if !ok {
		t.Fatalf("expected method bucket to exist")
	}

	stored, ok := methodGroup.Get("")
	if !ok {
		t.Fatalf("expected schema to be stored under empty URL")
	}
	if stored.Response != "Test" {
		t.Fatalf("stored schema mismatch")
	}
}

func TestEndpointSchema_String_HandlesComplexTypeNames(t *testing.T) {
	rt := NewEndpointSchema(false)
	schema := RpcSchema{
		Param:    "Array<string>",
		Query:    "Record<string, number>",
		Body:     "Partial<User>",
		Response: "Promise<Data>",
	}
	rt.AddRpcSchema("post", "/complex", schema)

	got := rt.String()

	if !strings.Contains(got, "params: Array<string>;") {
		t.Fatalf("expected complex param type preserved, got: %q", got)
	}
	if !strings.Contains(got, "query?: Record<string, number>;") {
		t.Fatalf("expected complex query type preserved, got: %q", got)
	}
	if !strings.Contains(got, "body: Partial<User>;") {
		t.Fatalf("expected complex body type preserved, got: %q", got)
	}
	if !strings.Contains(got, "response: Promise<Data>;") {
		t.Fatalf("expected complex response type preserved, got: %q", got)
	}
}

func TestAddRpcSchema_AllowsMultipleURLsUnderSameMethod(t *testing.T) {
	rt := NewEndpointSchema(false)

	rt.AddRpcSchema("get", "/first", RpcSchema{Response: "First"})
	rt.AddRpcSchema("get", "/second", RpcSchema{Response: "Second"})
	rt.AddRpcSchema("get", "/third", RpcSchema{Response: "Third"})

	methodGroup, ok := rt.types.Get("get")
	if !ok {
		t.Fatalf("expected method bucket to exist")
	}

	if methodGroup.Len() != 3 {
		t.Fatalf("expected 3 URLs under GET method, got %d", methodGroup.Len())
	}

	first, _ := methodGroup.Get("/first")
	if first.Response != "First" {
		t.Fatalf("first schema mismatch")
	}

	second, _ := methodGroup.Get("/second")
	if second.Response != "Second" {
		t.Fatalf("second schema mismatch")
	}

	third, _ := methodGroup.Get("/third")
	if third.Response != "Third" {
		t.Fatalf("third schema mismatch")
	}
}
