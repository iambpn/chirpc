package rpc

import (
	"net/http"
	"reflect"
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

func TestRouterRpcSchemas_ConvertToTs_GeneratesTypeScriptSchemaForSingleHandler(t *testing.T) {
	r := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) {
		return nil, nil
	}

	if _, err := r.RegisterHandler("get", "/status", handler); err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	out, err := r.ConvertToTs()
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}

	expectedString := `
		export type ApiSchema = { "GET": { "/status": { response: string; }; }; };
	`
	testVerifyTsTypes(t, out, expectedString)
}

func TestRouterRpcSchemas_ConvertToTs_GeneratesNestedTypeScriptInterfacesFromMultipleHandlers(t *testing.T) {
	r := NewRouterRpcSchemas()

	userHandler := func(*http.Request) (*testHttpResponse[testUserProfile], error) {
		return nil, nil
	}

	teamHandler := func(*http.Request) (*testHttpResponse[testTeamPayload], error) {
		return nil, nil
	}

	if _, err := r.RegisterHandler("get", "/users/{id}", userHandler); err != nil {
		t.Fatalf("registering user handler failed: %v", err)
	}

	if _, err := r.RegisterHandler("post", "/teams", teamHandler); err != nil {
		t.Fatalf("registering team handler failed: %v", err)
	}

	out, err := r.ConvertToTs()
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}

	expectedString := `
		interface Rpc__TestAddress {
			Line1:string;
			Zip:number;
		}
		interface Rpc__TestUserProfile {
			Name:string;
			Primary:Rpc__TestAddress;
		}
		interface Rpc__TestTeamPayload {
			Owner:Rpc__TestUserProfile;
			Members:(Rpc__TestUserProfile)[];
		}
		export type ApiSchema = {
			"GET": {
				"/users/:id": {
					response: Rpc__TestUserProfile;
				};
			};
			"POST": {
				"/teams": {
					response: Rpc__TestTeamPayload;
				};
			};
		};
	`

	testVerifyTsTypes(t, out, expectedString)
}

func TestRouterRpcSchemas_ConvertToTs_GeneratesTypeScriptSchemaForMultipleHandlers(t *testing.T) {
	r := NewRouterRpcSchemas()

	userHandler := func(*http.Request) (*testHttpResponse[string], error) {
		return nil, nil
	}
	teamHandler := func(*http.Request) (*testHttpResponse[int], error) {
		return nil, nil
	}

	if _, err := r.RegisterHandler("get", "/users", userHandler); err != nil {
		t.Fatalf("registering user handler failed: %v", err)
	}

	if _, err := r.RegisterHandler("post", "/teams", teamHandler); err != nil {
		t.Fatalf("registering team handler failed: %v", err)
	}

	out, err := r.ConvertToTs()
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}

	expectedString := `
		export type ApiSchema = {
			"GET": {
				"/users": {
					response: string;
				};
			};
			"POST": {
				"/teams": {
					response: number;
				};
			};
		};
	`
	testVerifyTsTypes(t, out, expectedString)
}

// Types moved to test_helpers_test.go for reuse across tests.

func TestRouterRpcSchemas_ConvertToTs_IncludesBodyQueryAndParamsInGeneratedSchema(t *testing.T) {
	r := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }

	schema, err := r.RegisterHandler("post", "/users/{userId}", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	schema.SetBodyType(testCreateReq{})
	schema.SetQueryType(testSearchQ{})
	schema.SetParamsType([]string{"userId"})

	out, err := r.ConvertToTs()
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}

	expected := `
		export type ApiSchema = {
			"POST": {
				"/users/:userId": {
					params: { "userId": string; };
					query?: { Filter:string; Limit:number; };
					body: { Name:string; TagIds:(number)[]; };
					response: string;
				};
			};
		};
	`
	testVerifyTsTypes(t, out, expected)
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

func TestRouterRpcSchemas_ConvertToTs_HandlesNoHandlers(t *testing.T) {
	r := NewRouterRpcSchemas()
	out, err := r.ConvertToTs()
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}
	expected := "\nexport type ApiSchema = { };"
	if out != expected {
		t.Fatalf("expected output %q for no handlers, got %q", expected, out)
	}
}

func TestRouterRpcSchemas_RegisterHandlers(t *testing.T) {
	t.Run("adds multiple schemas to collection", func(t *testing.T) {
		r := NewRouterRpcSchemas()

		schema1 := NewHandlerSchema("GET", "/test1", reflect.TypeOf(""))
		schema2 := NewHandlerSchema("POST", "/test2", reflect.TypeOf(""))
		schemas := []*HandlerSchema{schema1, schema2}

		r.RegisterHandlers(schemas)

		if len(r.schemas) != 2 {
			t.Errorf("expected 2 schemas, got %d", len(r.schemas))
		}

		if r.schemas[0] != schema1 {
			t.Error("first schema should match")
		}

		if r.schemas[1] != schema2 {
			t.Error("second schema should match")
		}
	})

	t.Run("does nothing when schemas slice is empty", func(t *testing.T) {
		r := NewRouterRpcSchemas()
		r.RegisterHandlers([]*HandlerSchema{})

		if len(r.schemas) != 0 {
			t.Errorf("expected 0 schemas, got %d", len(r.schemas))
		}
	})
}

func TestRouterRpcSchemas_RegisterHandlerFrom(t *testing.T) {
	r1 := NewRouterRpcSchemas()
	r2 := NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) {
		return nil, nil
	}

	// Register handlers in r2
	schema1, _ := r2.RegisterHandler("GET", "/test1", handler)
	schema2, _ := r2.RegisterHandler("POST", "/test2", handler)

	// Register r2's handlers into r1
	r1.RegisterHandlerFrom(r2)

	if len(r1.schemas) != 2 {
		t.Errorf("expected 2 schemas in r1, got %d", len(r1.schemas))
	}

	if r1.schemas[0] != schema1 {
		t.Error("first schema should match")
	}

	if r1.schemas[1] != schema2 {
		t.Error("second schema should match")
	}
}
