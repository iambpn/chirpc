package rpcType

import (
	"strings"
	"testing"
)

func TestNewRpcTypeInitializesCorrectly(t *testing.T) {
	rt := NewRpcType(true)

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

func TestAddTypeInitializesMethodBucket(t *testing.T) {
	rt := NewRpcType(false)

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

func TestAddTypeOverridesExistingEntry(t *testing.T) {
	rt := NewRpcType(false)

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

func TestStringWithExportAndNoEntries(t *testing.T) {
	rt := NewRpcType(true)

	expected := "export type ApiSchema = { };"
	if got := rt.String(); got != expected {
		t.Fatalf("unexpected string output.\nexpected: %q\n     got: %q", expected, got)
	}
}

func TestStringWithFullSchema(t *testing.T) {
	rt := NewRpcType(false)
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

func TestStringOmitsEmptyFieldsAndFallsBackToVoidResponse(t *testing.T) {
	rt := NewRpcType(false)
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

func TestConvertURLPattern_SimpleSlug(t *testing.T) {
	in := "/users/{id}"
	out := convertURLPattern(in)
	if out != "/users/:id" {
		t.Fatalf("unexpected conversion: %q -> %q", in, out)
	}
}

func TestConvertURLPattern_MultipleSlugs(t *testing.T) {
	in := "/a/{id}/b/{slug}"
	out := convertURLPattern(in)
	if out != "/a/:id/b/:slug" {
		t.Fatalf("unexpected conversion: %q -> %q", in, out)
	}
}

func TestConvertURLPattern_NestedBracesInsideSlug(t *testing.T) {
	in := "/x/{a{b}c}/y"
	out := convertURLPattern(in)
	if out != "/x/:a{b}c/y" {
		t.Fatalf("unexpected conversion with nested braces: %q -> %q", in, out)
	}
}

func TestStringWithExportAndEntries(t *testing.T) {
	rt := NewRpcType(true)
	rt.AddRpcSchema("get", "/ping", RpcSchema{Response: "Pong"})
	got := rt.String()
	if !strings.HasPrefix(got, "export type ApiSchema") {
		t.Fatalf("expected export prefix, got: %q", got)
	}
}

func TestStringConvertsURLPatterns(t *testing.T) {
	rt := NewRpcType(false)
	rt.AddRpcSchema("get", "/foo/{id}/bar", RpcSchema{Response: "R"})
	got := rt.String()
	if !strings.Contains(got, "\"/foo/:id/bar\"") {
		t.Fatalf("expected converted url pattern in output, got: %q", got)
	}
}

func TestStringPreservesInsertionOrder(t *testing.T) {
	rt := NewRpcType(false)
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


