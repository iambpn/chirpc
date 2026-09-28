package chirpc

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

type createUserBody struct {
	Name string `json:"name"`
	Age  int    `json:"age,omitempty"`
}

type userQuery struct {
	Notify bool     `json:"notify"`
	Tags   []string `json:"tag,omitempty"`
}

type createdUser struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Notify bool     `json:"notify"`
	Tags   []string `json:"tags"`
}

func serveBody(r *RPCRouter, method, url, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest(method, url, strings.NewReader(body)))
	return recorder
}

func decodeErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder) ErrorResponse {
	t.Helper()
	var errResp ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("expected an ErrorResponse body, got %q", recorder.Body.String())
	}
	return errResp
}

func newTypedRouter() *RPCRouter {
	r := NewRPCRouter()
	AddTypedHandler(r, MethodPost, "/teams/{teamId}/users", func(req *Request[createUserBody, userQuery]) (*HttpResponse[createdUser], error) {
		return &HttpResponse[createdUser]{
			StatusCode: http.StatusCreated,
			Body: createdUser{
				ID:     req.Param("teamId") + "-1",
				Name:   req.Body.Name,
				Notify: req.Query.Notify,
				Tags:   req.Query.Tags,
			},
		}, nil
	})
	return r
}

func TestTypedHandlerDecodesBodyQueryAndParams(t *testing.T) {
	recorder := serveBody(newTypedRouter(), http.MethodPost, "/teams/t7/users?notify=true&tag=a&tag=b", `{"name":"Ada"}`)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d with %q", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	var got createdUser
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := createdUser{ID: "t7-1", Name: "Ada", Notify: true, Tags: []string{"a", "b"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %+v, got %+v", want, got)
	}
}

func TestTypedHandlerSchemaComesFromTypeParameters(t *testing.T) {
	content := generateSchema(t, newTypedRouter())

	for _, want := range []string{
		`"/teams/:teamId/users": {`,
		`params: { "teamId": string; };`,
		"query: {",
		"notify: boolean;",
		"tag?: string[];",
		"body: {",
		"age?: number;",
		"response: Chirpc__CreatedUser;",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected schema to contain %q, got:\n%s", want, content)
		}
	}
}

func TestTypedHandlerRejectsBadRequests(t *testing.T) {
	r := newTypedRouter()
	cases := []struct {
		name       string
		url        string
		body       string
		message    string
		validation map[string][]string
	}{
		{"missing body", "/teams/1/users?notify=true", "", "The request body is missing.", nil},
		{"invalid JSON", "/teams/1/users?notify=true", "{", "The request body is not valid JSON.", nil},
		{"wrong JSON type", "/teams/1/users?notify=true", `{"name":"Ada","age":"old"}`, "The request body is not valid.",
			map[string][]string{"age": {"This value must be a number."}}},
		{"missing query field", "/teams/1/users", `{"name":"Ada"}`, "The query parameters are not valid.",
			map[string][]string{"notify": {"This field is required."}}},
		{"missing body field", "/teams/1/users?notify=true", `{"age":30}`, "The request body is not valid.",
			map[string][]string{"name": {"This field is required."}}},
		{"null body", "/teams/1/users?notify=true", "null", "The request body is missing.", nil},
		{"duplicate key", "/teams/1/users?notify=true", `{"name":"Ada","name":"Bob"}`, "The request body is not valid JSON.", nil},
		{"key in the wrong case", "/teams/1/users?notify=true", `{"Name":"Ada"}`, "The request body is not valid.",
			map[string][]string{"name": {"This field is required."}}},
	}

	for _, c := range cases {
		recorder := serveBody(r, http.MethodPost, c.url, c.body)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected status 400, got %d", c.name, recorder.Code)
		}
		errResp := decodeErrorResponse(t, recorder)
		if len(errResp.Errors) != 1 || errResp.Errors[0] != c.message {
			t.Fatalf("%s: expected message %q, got %v", c.name, c.message, errResp.Errors)
		}
		if !reflect.DeepEqual(errResp.ValidationErrors, c.validation) {
			t.Fatalf("%s: expected validation errors %v, got %v", c.name, c.validation, errResp.ValidationErrors)
		}
	}
}

var errDatabase = errors.New("database is down")

func TestTypedHandlerErrors(t *testing.T) {
	var seenCause error
	r := NewRPCRouter()
	RegisterErrorHandler(r, func(req *http.Request, errResp *ErrorResponse) *HttpResponse[ErrorResponse] {
		seenCause = errResp.Cause
		return &HttpResponse[ErrorResponse]{Body: *errResp}
	})

	handlers := map[string]error{
		"/plain":    errDatabase,
		"/response": &ErrorResponse{StatusCode: http.StatusNotFound, Errors: []string{"No such user."}},
		"/wrapped":  fmt.Errorf("loading user: %w", &ErrorResponse{StatusCode: http.StatusConflict}),
	}
	for path, err := range handlers {
		AddTypedHandler(r, MethodGet, path, func(req *Request[NoBody, NoQuery]) (*HttpResponse[string], error) {
			return nil, err
		})
	}

	recorder := serveBody(r, http.MethodGet, "/plain", "")
	errResp := decodeErrorResponse(t, recorder)
	if recorder.Code != http.StatusInternalServerError || errResp.Errors[0] != "An internal server error occurred." {
		t.Fatalf("expected a general 500 error, got %d %v", recorder.Code, errResp.Errors)
	}
	if strings.Contains(recorder.Body.String(), "database") {
		t.Fatalf("expected the cause to stay private, got %q", recorder.Body.String())
	}
	if !errors.Is(seenCause, errDatabase) {
		t.Fatalf("expected the error handler to see the cause, got %v", seenCause)
	}

	if code := serveBody(r, http.MethodGet, "/response", "").Code; code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", code)
	}
	if code := serveBody(r, http.MethodGet, "/wrapped", "").Code; code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", code)
	}
}

func TestTypedHandlerWithoutBodyOrQueryAndWithPointers(t *testing.T) {
	r := NewRPCRouter()
	AddTypedHandler(r, MethodGet, "/ping", func(req *Request[NoBody, NoQuery]) (*HttpResponse[string], error) {
		return &HttpResponse[string]{Body: "pong"}, nil
	})
	AddTypedHandler(r, MethodPut, "/pointer", func(req *Request[*createUserBody, *userQuery]) (*HttpResponse[string], error) {
		return &HttpResponse[string]{Body: fmt.Sprintf("%s %v", req.Body.Name, req.Query.Notify)}, nil
	})

	if recorder := serveBody(r, http.MethodGet, "/ping", ""); recorder.Body.String() != `"pong"` {
		t.Fatalf("expected pong, got %d %q", recorder.Code, recorder.Body.String())
	}
	if recorder := serveBody(r, http.MethodPut, "/pointer?notify=1", `{"name":"Ada"}`); recorder.Body.String() != `"Ada true"` {
		t.Fatalf("expected pointer types to be decoded, got %d %q", recorder.Code, recorder.Body.String())
	}

	content := generateSchema(t, r)
	ping := content[strings.Index(content, `"/ping"`):strings.Index(content, `"/pointer"`)]
	if strings.Contains(ping, "body") || strings.Contains(ping, "query") {
		t.Fatalf("expected /ping to have no body or query, got %s", ping)
	}
}

func TestAddTypedHandlerPanicsForUnusableTypes(t *testing.T) {
	expectPanic := func(want string, register func()) {
		t.Helper()
		defer func() {
			if recovered := fmt.Sprint(recover()); !strings.Contains(recovered, want) {
				t.Fatalf("expected a panic containing %q, got %q", want, recovered)
			}
		}()
		register()
	}

	r := NewRPCRouter()
	expectPanic("the body type must be a struct", func() {
		AddTypedHandler(r, MethodPost, "/a", func(req *Request[[]string, NoQuery]) (*HttpResponse[string], error) { return nil, nil })
	})
	expectPanic("uses the tsKey tag, which is no longer supported", func() {
		AddTypedHandler(r, MethodGet, "/c", func(req *Request[NoBody, struct {
			Page int `tsKey:"page"`
		}]) (*HttpResponse[string], error) {
			return nil, nil
		})
	})
	expectPanic("cannot be read from a query string", func() {
		AddTypedHandler(r, MethodGet, "/b", func(req *Request[NoBody, struct{ Inner createUserBody }]) (*HttpResponse[string], error) {
			return nil, nil
		})
	})
}

func TestErrorResponseIsAnError(t *testing.T) {
	var err error = &ErrorResponse{Errors: []string{"First.", "Second."}}
	if err.Error() != "First. Second." {
		t.Fatalf("unexpected message %q", err.Error())
	}

	wrapped := &ErrorResponse{Cause: errDatabase}
	if wrapped.Error() != errDatabase.Error() || !errors.Is(wrapped, errDatabase) {
		t.Fatalf("expected the cause to be the message and to unwrap, got %q", wrapped.Error())
	}

	if (&ErrorResponse{StatusCode: http.StatusNotFound}).Error() != "Not Found" {
		t.Fatal("expected the status text when there are no messages")
	}
}

func TestTypedHandlerReportsMissingNestedBodyFields(t *testing.T) {
	type line struct {
		SKU      string `json:"sku"`
		Quantity int    `json:"quantity,omitempty"`
	}
	type orderBody struct {
		Customer string `json:"customer"`
		Lines    []line `json:"lines"`
	}

	called := false
	r := NewRPCRouter()
	AddTypedHandler(r, MethodPost, "/orders", func(req *Request[orderBody, NoQuery]) (*HttpResponse[string], error) {
		called = true
		return &HttpResponse[string]{Body: "ok"}, nil
	})

	recorder := serveBody(r, http.MethodPost, "/orders", `{"customer":"Ada","lines":[{"sku":"a"},{"quantity":2}]}`)
	if recorder.Code != http.StatusBadRequest || called {
		t.Fatalf("expected 400 without calling the handler, got %d (called: %v)", recorder.Code, called)
	}
	want := map[string][]string{"lines[1].sku": {"This field is required."}}
	if got := decodeErrorResponse(t, recorder).ValidationErrors; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected validation errors %v, got %v", want, got)
	}

	recorder = serveBody(r, http.MethodPost, "/orders", `{"customer":"Ada","lines":[{"sku":"a"},{"sku":"b","quantity":"two"}]}`)
	want = map[string][]string{"lines[1].quantity": {"This value must be a number."}}
	if got := decodeErrorResponse(t, recorder).ValidationErrors; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected validation errors %v, got %v", want, got)
	}

	if code := serveBody(r, http.MethodPost, "/orders", `{"customer":"Ada","lines":[{"sku":"a"}]}`).Code; code != http.StatusOK || !called {
		t.Fatalf("expected a complete body to reach the handler, got %d", code)
	}
}

func TestTypedHandlerReportsValuesWithTheirOwnDecoding(t *testing.T) {
	type eventBody struct {
		At time.Time `json:"at"`
	}
	r := NewRPCRouter()
	AddTypedHandler(r, MethodPost, "/events", func(req *Request[eventBody, NoQuery]) (*HttpResponse[string], error) {
		return &HttpResponse[string]{Body: "ok"}, nil
	})

	recorder := serveBody(r, http.MethodPost, "/events", `{"at":"yesterday"}`)
	want := map[string][]string{"at": {"This value is not valid."}}
	if got := decodeErrorResponse(t, recorder).ValidationErrors; recorder.Code != http.StatusBadRequest || !reflect.DeepEqual(got, want) {
		t.Fatalf("expected 400 with %v, got %d with %v", want, recorder.Code, got)
	}
}

func TestTypedHandlerReadsDurationsAsNanoseconds(t *testing.T) {
	type timerBody struct {
		Wait time.Duration `json:"wait"`
	}
	r := NewRPCRouter()
	AddTypedHandler(r, MethodPost, "/timers", func(req *Request[timerBody, NoQuery]) (*HttpResponse[timerBody], error) {
		return &HttpResponse[timerBody]{Body: timerBody{Wait: req.Body.Wait * 2}}, nil
	})

	recorder := serveBody(r, http.MethodPost, "/timers", `{"wait":1500000000}`)
	if recorder.Code != http.StatusOK || recorder.Body.String() != `{"wait":3000000000}` {
		t.Fatalf("expected the doubled duration in nanoseconds, got %d with %q", recorder.Code, recorder.Body.String())
	}
}
