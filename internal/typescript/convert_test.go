package typescript

import (
	"net/http"
	"strings"
	"testing"

	"github.com/iambpn/chirpc/internal/rpc"
)

func TestRouterRpcSchemas_ConvertToTs_GeneratesTypeScriptSchemaForSingleHandler(t *testing.T) {
	r := rpc.NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) {
		return nil, nil
	}

	if _, err := r.RegisterHandler("get", "/status", handler); err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	out, err := Convert(r)
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}

	expectedString := `
		export type ApiSchema = { "GET": { "/status": { response: string; }; }; };
	`
	testVerifyTsTypes(t, out, expectedString)
}

func TestRouterRpcSchemas_ConvertToTs_GeneratesNestedTypeScriptInterfacesFromMultipleHandlers(t *testing.T) {
	r := rpc.NewRouterRpcSchemas()

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

	out, err := Convert(r)
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}

	expectedString := `
		interface Typescript__TestUserProfile {
			Name:string;
			Primary:Typescript__TestAddress;
		}
		interface Typescript__TestAddress {
			Line1:string;
			Zip:number;
		}
		interface Typescript__TestTeamPayload {
			Owner:Typescript__TestUserProfile;
			Members:Typescript__TestUserProfile[];
		}
		export type ApiSchema = {
			"GET": {
				"/users/:id": {
					params: { "id": string; };
					response: Typescript__TestUserProfile;
				};
			};
			"POST": {
				"/teams": {
					response: Typescript__TestTeamPayload;
				};
			};
		};
	`

	testVerifyTsTypes(t, out, expectedString)
}

func TestRouterRpcSchemas_ConvertToTs_GeneratesTypeScriptSchemaForMultipleHandlers(t *testing.T) {
	r := rpc.NewRouterRpcSchemas()

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

	out, err := Convert(r)
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
	r := rpc.NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }

	schema, err := r.RegisterHandler("post", "/users/{userId}", handler)
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	schema.SetBodyType(testCreateReq{})
	schema.SetQueryType(testSearchQ{})
	schema.SetParamsType([]string{"userId"})

	out, err := Convert(r)
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}

	expected := `
		export type ApiSchema = {
			"POST": {
				"/users/:userId": {
					params: { "userId": string; };
					query: { Filter:string; Limit:number; };
					body: { Name:string; TagIds:number[]; };
					response: string;
				};
			};
		};
	`
	testVerifyTsTypes(t, out, expected)
}

func TestRouterRpcSchemas_ConvertToTs_HandlesNoHandlers(t *testing.T) {
	r := rpc.NewRouterRpcSchemas()
	out, err := Convert(r)
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}
	expected := "export type ApiSchema = {\n};\n"
	if out != expected {
		t.Fatalf("expected output %q for no handlers, got %q", expected, out)
	}
}

func TestRouterRpcSchemas_ConvertToTs_PrefixesMountedRoutesAndParams(t *testing.T) {
	root := rpc.NewRouterRpcSchemas()
	outer := rpc.NewRouterRpcSchemas()
	inner := rpc.NewRouterRpcSchemas()
	group := rpc.NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }

	if _, err := inner.RegisterHandler("GET", "/posts/{postId:[0-9]+}", handler); err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}
	if _, err := group.RegisterHandler("GET", "/grouped", handler); err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	root.Mount("/users/{userId}", outer)
	outer.Mount("/inner", inner)
	outer.Mount("", group)

	// A handler added after mounting is still included.
	if _, err := inner.RegisterHandler("GET", "/late", handler); err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	out, err := Convert(root)
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}

	expected := `
		export type ApiSchema = {
			"GET": {
				"/users/:userId/inner/posts/:postId": {
					params: { "userId": string;"postId": string; };
					response: string;
				};
				"/users/:userId/inner/late": {
					params: { "userId": string; };
					response: string;
				};
				"/users/:userId/grouped": {
					params: { "userId": string; };
					response: string;
				};
			};
		};
	`
	testVerifyTsTypes(t, out, expected)
}

func TestRouterRpcSchemas_ConvertToTs_MountsSameChildAtTwoPaths(t *testing.T) {
	root := rpc.NewRouterRpcSchemas()
	child := rpc.NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }
	if _, err := child.RegisterHandler("GET", "/x", handler); err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	root.Mount("/a", child)
	root.Mount("/b", child)

	out, err := Convert(root)
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}

	expected := `export type ApiSchema = { "GET": { "/a/x": { response: string; }; "/b/x": { response: string; }; }; };`
	testVerifyTsTypes(t, out, expected)
}

func TestRouterRpcSchemas_ConvertToTs_ErrorHandlerComesFirst(t *testing.T) {
	r := rpc.NewRouterRpcSchemas()

	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }
	if _, err := r.RegisterHandler("GET", "/x", handler); err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	errorHandler := func(*http.Request, error) *testHttpResponse[int] { return nil }
	if err := r.RegisterErrorHandler(errorHandler); err != nil {
		t.Fatalf("unexpected error registering error handler: %v", err)
	}

	out, err := Convert(r)
	if err != nil {
		t.Fatalf("ConvertToTs returned error: %v", err)
	}

	expected := `export type ApiSchema = { "ERROR_HANDLER": { "/": { response: number; }; }; "GET": { "/x": { response: string; }; }; };`
	testVerifyTsTypes(t, out, expected)
}

func TestRouterRpcSchemas_ConvertToTs_ChildErrorHandlerTypes(t *testing.T) {
	sameType := func(*http.Request, error) *testHttpResponse[int] { return nil }
	otherType := func(*http.Request, error) *testHttpResponse[string] { return nil }

	t.Run("allows a child error handler with the root's type", func(t *testing.T) {
		root := rpc.NewRouterRpcSchemas()
		child := rpc.NewRouterRpcSchemas()
		_ = root.RegisterErrorHandler(sameType)
		_ = child.RegisterErrorHandler(sameType)
		root.Mount("/child", child)

		if _, err := Convert(root); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("rejects a child error handler with a different type", func(t *testing.T) {
		root := rpc.NewRouterRpcSchemas()
		child := rpc.NewRouterRpcSchemas()
		_ = root.RegisterErrorHandler(sameType)
		_ = child.RegisterErrorHandler(otherType)
		root.Mount("/child", child)

		_, err := Convert(root)
		if err == nil || !strings.Contains(err.Error(), `router at "/child"`) {
			t.Fatalf("expected error naming the child router, got %v", err)
		}
	})
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

func TestConvert_QueryIsOptionalOnlyWhenAllFieldsAreOptional(t *testing.T) {
	r := rpc.NewRouterRpcSchemas()
	handler := func(*http.Request) (*testHttpResponse[string], error) { return nil, nil }

	optional, _ := r.RegisterHandler("GET", "/optional", handler)
	optional.SetQueryType(testOptionalQ{})
	required, _ := r.RegisterHandler("GET", "/required", handler)
	required.SetQueryType(testSearchQ{})

	out, err := Convert(r)
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	expected := `
		export type ApiSchema = {
			"GET": {
				"/optional": {
					query?: { filter?: string; Limit?: number; };
					response: string;
				};
				"/required": {
					query: { Filter: string; Limit: number; };
					response: string;
				};
			};
		};
	`
	testVerifyTsTypes(t, out, expected)
}

func TestConvert_WritesOneMemberPerLine(t *testing.T) {
	r := rpc.NewRouterRpcSchemas()
	handler := func(*http.Request) (*testHttpResponse[testAddress], error) { return nil, nil }
	if _, err := r.RegisterHandler("GET", "/users/{id}", handler); err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	out, err := Convert(r)
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}

	expected := `interface Typescript__TestAddress {
    Line1: string;
    Zip: number;
}

export type ApiSchema = {
  "GET": {
    "/users/:id": {
      params: { "id": string; };
      response: Typescript__TestAddress;
    };
  };
};
`
	if out != expected {
		t.Fatalf("expected output\n%s\ngot\n%s", expected, out)
	}
}
