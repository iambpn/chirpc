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
	rt.AddType("get", "/items", schema)

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

	rt.AddType("get", "/items", first)
	rt.AddType("get", "/items", second)

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

	rt.AddType("post", "/users", schema)

	expected := "type ApiSchema = { \"POST\": { \"/users\": { param: SomeParam; query?: SomeQuery; body: SomeBody; response: SomeResponse; }; }; };"
	if got := rt.String(); got != expected {
		t.Fatalf("unexpected string output.\nexpected: %q\n     got: %q", expected, got)
	}
}

func TestStringOmitsEmptyFieldsAndFallsBackToVoidResponse(t *testing.T) {
	rt := NewRpcType(false)
	rt.AddType("get", "/empty", RpcSchema{})

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
