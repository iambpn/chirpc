package chirpc

import (
	"context"
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
	type sample struct{ X int }
	var (
		sampleVal  sample
		intVal     int
		strVal     string
		mapVal                 = map[string]int{}
		sliceVal               = []string{}
		arrayVal               = [2]int{}
		boolVal                = true
		floatVal               = 3.14
		ifaceVal   interface{} = 5
		ptrVal                 = &sampleVal
		chanVal                = make(chan int)
		funcVal                = func() {}
		complexVal complex64   = 1 + 2i
	)

	cases := []struct {
		val  any
		want bool
	}{
		{sampleVal, true},
		{mapVal, true},
		{sliceVal, true},
		{arrayVal, true},
		{strVal, true},
		{intVal, true},
		{boolVal, true},
		{floatVal, true},
		{ifaceVal, true},
		{ptrVal, true},
		{chanVal, false},
		{funcVal, false},
		{complexVal, false},
	}

	for _, c := range cases {
		kind := reflect.TypeOf(c.val).Kind()
		if got := isJSONMarshable(kind); got != c.want {
			t.Fatalf("isJSONMarshable(%v) = %v, want %v", kind, got, c.want)
		}
	}
}

func TestSendStreamBytes_WritesHeadersAndNDJSONChunks(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stream", nil)

	stream := make(chan []byte, 2)
	stream <- []byte("{\"message\":\"one\"}\n")
	stream <- []byte("{\"message\":\"two\"}\n")
	close(stream)

	sendStreamBytes(recorder, req, &StreamResponse{
		Stream: stream,
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if recorder.Header().Get("Content-Type") != "application/x-ndjson" {
		t.Fatalf("expected Content-Type application/x-ndjson")
	}

	if recorder.Header().Get("Transfer-Encoding") != "chunked" {
		t.Fatalf("expected Transfer-Encoding chunked")
	}

	got := recorder.Body.String()
	expected := "{\"message\":\"one\"}\n{\"message\":\"two\"}\n"
	if got != expected {
		t.Fatalf("expected body %q, got %q", expected, got)
	}
}

func TestSendStreamBytes_StopsWhenRequestContextCanceled(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stream", nil)

	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)

	stream := make(chan []byte)

	sendStreamBytes(recorder, req, &StreamResponse{
		Stream: stream,
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestSendStreamBytes_UsesCustomHeadersAndStatus(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stream", nil)

	stream := make(chan []byte, 1)
	stream <- []byte("{\"message\":\"custom\"}\n")
	close(stream)

	sendStreamBytes(recorder, req, &StreamResponse{
		StatusCode: http.StatusAccepted,
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Stream-Mode":     "custom",
			"Transfer-Encoding": "identity",
		},
		Stream: stream,
	})

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, recorder.Code)
	}

	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type override to be applied")
	}

	if recorder.Header().Get("X-Stream-Mode") != "custom" {
		t.Fatalf("expected custom header to be set")
	}

	if recorder.Header().Get("Transfer-Encoding") != "identity" {
		t.Fatalf("expected Transfer-Encoding override to be applied")
	}
}

func TestSendStreamBytes_WritesRawBytes(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stream", nil)

	stream := make(chan []byte, 2)
	stream <- []byte("hello")
	stream <- []byte("world")
	close(stream)

	sendStreamBytes(recorder, req, &StreamResponse{
		Headers: map[string]string{
			"Content-Type": "application/octet-stream",
		},
		Stream: stream,
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if recorder.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("expected Content-Type override to be applied")
	}

	if got := recorder.Body.String(); got != "helloworld" {
		t.Fatalf("expected body %q, got %q", "helloworld", got)
	}
}
