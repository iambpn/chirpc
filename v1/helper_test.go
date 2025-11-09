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

func TestParseURLSlugSingle(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"/users/{id}", []string{"id"}},
		{"{one}", []string{"one"}},
		{"/no/slugs/here", []string{}},
		{"", []string{}},
	}

	for _, c := range cases {
		got := parseURLSlug(c.in)
		if len(got) != len(c.want) {
			t.Fatalf("parseURLSlug(%q) len = %d, want %d (%v)", c.in, len(got), len(c.want), got)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("parseURLSlug(%q)[%d] = %q, want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}

func TestParseURLSlugMultiple(t *testing.T) {
	got := parseURLSlug("/{user}/{id}")
	if len(got) != 2 || got[0] != "user" || got[1] != "id" {
		t.Fatalf("parseURLSlug multi failed: got %v, want [user id]", got)
	}
}

func TestIsJSONMarshable(t *testing.T) {
	type sample struct{ X int }
	var (
		sampleVal sample
		intVal    int
		strVal    string
		mapVal    = map[string]int{}
		sliceVal  = []string{}
		arrayVal  = [2]int{}
		boolVal   = true
		floatVal  = 3.14
		ifaceVal  interface{} = 5
		ptrVal    = &sampleVal
		chanVal   = make(chan int)
		funcVal   = func() {}
		complexVal complex64 = 1 + 2i
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
