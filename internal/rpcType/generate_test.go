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

	if err := RegisterHandler("get", "/users", handler); err != nil {
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

	if err := RegisterHandler("post", "/invalid", 123); err == nil {
		t.Fatalf("expected error when registering non-function handler")
	}
}

func TestConvertToTs_GeneratesSingleHandlerSchema(t *testing.T) {
	resetTypeRegistry()
	t.Cleanup(resetTypeRegistry)

	handler := func(*http.Request) (*httpResponse[string], error) {
		return nil, nil
	}

	if err := RegisterHandler("get", "/status", handler); err != nil {
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

	if err := RegisterHandler("get", "/users/{id}", userHandler); err != nil {
		t.Fatalf("registering user handler failed: %v", err)
	}

	if err := RegisterHandler("post", "/teams", teamHandler); err != nil {
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
				"/users/{id}": {
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
