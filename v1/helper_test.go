package chirpc

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSendResponse_WritesHeadersStatusAndJSONBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	resp := &HttpResponse[string]{
		StatusCode: http.StatusCreated,
		Body:       "ok",
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-Test":       "pass",
		},
	}

	sendResponse(recorder, resp)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type header to be set")
	}

	if recorder.Header().Get("X-Test") != "pass" {
		t.Fatalf("expected X-Test header to be set")
	}

	const expectedBody = "\"ok\""
	if recorder.Body.String() != expectedBody {
		t.Fatalf("expected body %q, got %q", expectedBody, recorder.Body.String())
	}
}

func TestSendResponse_MarshalsPointerBody(t *testing.T) {
	type payload struct {
		Message string `json:"message"`
	}

	recorder := httptest.NewRecorder()
	resp := &HttpResponse[*payload]{
		StatusCode: http.StatusOK,
		Body:       &payload{Message: "hello"},
		Headers:    map[string]string{"X-Ptr": "yes"},
	}

	sendResponse(recorder, resp)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	const expectedBody = "{\"message\":\"hello\"}"
	if recorder.Body.String() != expectedBody {
		t.Fatalf("expected body %q, got %q", expectedBody, recorder.Body.String())
	}
}

func TestSendResponse_SkipsBodyForNonJSONKinds(t *testing.T) {
	recorder := httptest.NewRecorder()
	resp := &HttpResponse[chan int]{
		StatusCode: http.StatusAccepted,
		Body:       make(chan int),
	}

	sendResponse(recorder, resp)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	expectedBody := "payload is not marshallable\n"
	if recorder.Body.String() != expectedBody {
		t.Fatalf("expected %q to be written, got %q", expectedBody, recorder.Body.String())
	}
}

func TestSendResponse_WritesInternalServerErrorOnMarshalFailure(t *testing.T) {
	type badPayload struct {
		Stream chan int `json:"stream"`
	}

	recorder := httptest.NewRecorder()
	resp := &HttpResponse[badPayload]{
		StatusCode: http.StatusOK,
		Body:       badPayload{Stream: make(chan int)},
	}

	sendResponse(recorder, resp)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}

	const expectedBody = "an error occurred while marshalling payload\n"
	if recorder.Body.String() != expectedBody {
		t.Fatalf("expected body %q, got %q", expectedBody, recorder.Body.String())
	}
}

func TestIsJSONMarshable(t *testing.T) {
	t.Helper()

	cases := []struct {
		kind    reflect.Kind
		allowed bool
	}{
		{reflect.Struct, true},
		{reflect.Map, true},
		{reflect.Slice, true},
		{reflect.Array, true},
		{reflect.String, true},
		{reflect.Int64, true},
		{reflect.Uint8, true},
		{reflect.Float32, true},
		{reflect.Bool, true},
		{reflect.Pointer, true},
		{reflect.Interface, true},
		{reflect.Func, false},
		{reflect.Chan, false},
		{reflect.UnsafePointer, false},
	}

	for _, tc := range cases {
		t.Run(tc.kind.String(), func(t *testing.T) {
			if got := isJSONMarshable(tc.kind); got != tc.allowed {
				t.Fatalf("expected %v for kind %s, got %v", tc.allowed, tc.kind, got)
			}
		})
	}
}
