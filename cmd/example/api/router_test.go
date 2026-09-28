package api

import (
	"bytes"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// client sends requests to the router in memory. RPCRouter is an http.Handler, so no
// network server is needed.
type client struct {
	t      *testing.T
	router http.Handler
	token  string
}

// do sends a request and decodes the JSON response into out when out is not nil.
func (c client) do(method, path string, body any, out any) int {
	c.t.Helper()

	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("Could not encode the request body: %v", err)
		}
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	rec := httptest.NewRecorder()
	c.router.ServeHTTP(rec, req)

	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			c.t.Fatalf("%s %s: could not decode the response %q: %v", method, path, rec.Body.String(), err)
		}
	}
	return rec.Code
}

func expectStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("The status is %d, but it should be %d.", got, want)
	}
}

func TestTaskLifecycle(t *testing.T) {
	router := NewRouter(NewStore())
	alice := client{t: t, router: router, token: "alice-token"}
	bob := client{t: t, router: router, token: "bob-token"}

	var created Task
	status := alice.do(http.MethodPost, "/api/v1/tasks", map[string]any{
		"title":     "  Ship the example  ",
		"tags":      []string{"docs"},
		"checklist": []map[string]any{{"text": "Write the code"}, {"text": "Write the README", "done": true}},
	}, &created)
	expectStatus(t, status, http.StatusCreated)
	if created.ID.String() != "task_1" || created.Title != "Ship the example" || created.Priority != PriorityMedium || created.Status != StatusTodo {
		t.Fatalf("The created task is not right: %+v", created)
	}

	var fetched Task
	expectStatus(t, alice.do(http.MethodGet, "/api/v1/tasks/task_1", nil, &fetched), http.StatusOK)
	if len(fetched.Checklist) != 2 || !fetched.Checklist[1].Done {
		t.Fatalf("The checklist is not right: %+v", fetched.Checklist)
	}

	var apiErr APIError
	expectStatus(t, bob.do(http.MethodGet, "/api/v1/tasks/task_1", nil, &apiErr), http.StatusNotFound)
	if apiErr.Message != "This task does not exist." || apiErr.RequestID == "" {
		t.Fatalf("The error is not right: %+v", apiErr)
	}

	var updated Task
	status = alice.do(http.MethodPatch, "/api/v1/tasks/task_1", map[string]any{"status": "done"}, &updated)
	expectStatus(t, status, http.StatusOK)
	if updated.Status != StatusDone || updated.Title != "Ship the example" {
		t.Fatalf("The update changed the wrong fields: %+v", updated)
	}

	expectStatus(t, alice.do(http.MethodDelete, "/api/v1/tasks/task_1", nil, nil), http.StatusNoContent)
	expectStatus(t, alice.do(http.MethodGet, "/api/v1/tasks/task_1", nil, nil), http.StatusNotFound)
}

func TestListFiltersAndPages(t *testing.T) {
	router := NewRouter(NewStore())
	alice := client{t: t, router: router, token: "alice-token"}

	for _, body := range []map[string]any{
		{"title": "One", "tags": []string{"a", "b"}, "dueAt": "2030-01-01T00:00:00Z"},
		{"title": "Two", "tags": []string{"a"}},
		{"title": "Three", "tags": []string{"b"}, "dueAt": "2031-01-01T00:00:00Z"},
	} {
		expectStatus(t, alice.do(http.MethodPost, "/api/v1/tasks", body, nil), http.StatusCreated)
	}

	tests := []struct {
		query      string
		wantTitles []string
		wantNext   *int
	}{
		{query: "", wantTitles: []string{"One", "Two", "Three"}},
		{query: "?tag=a", wantTitles: []string{"One", "Two"}},
		{query: "?tag=a&tag=b", wantTitles: []string{"One"}},
		{query: "?dueBefore=2030-06-01T00:00:00Z", wantTitles: []string{"One"}},
		{query: "?limit=2", wantTitles: []string{"One", "Two"}, wantNext: new(2)},
		{query: "?limit=2&offset=2", wantTitles: []string{"Three"}},
		{query: "?status=done", wantTitles: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			var page TaskPage
			expectStatus(t, alice.do(http.MethodGet, "/api/v1/tasks"+tt.query, nil, &page), http.StatusOK)

			titles := []string{}
			for _, task := range page.Items {
				titles = append(titles, task.Title)
			}
			if !slices.Equal(titles, tt.wantTitles) {
				t.Fatalf("The titles are %v, but they should be %v.", titles, tt.wantTitles)
			}
			if (page.NextOffset == nil) != (tt.wantNext == nil) || (page.NextOffset != nil && *page.NextOffset != *tt.wantNext) {
				t.Fatalf("The next offset is %v, but it should be %v.", page.NextOffset, tt.wantNext)
			}
		})
	}
}

func TestErrors(t *testing.T) {
	router := NewRouter(NewStore())
	alice := client{t: t, router: router, token: "alice-token"}
	anonymous := client{t: t, router: router}

	tests := []struct {
		name       string
		client     client
		method     string
		path       string
		body       any
		wantStatus int
		wantField  string
	}{
		{name: "no token", client: anonymous, method: http.MethodGet, path: "/api/v1/tasks", wantStatus: http.StatusUnauthorized},
		{name: "missing title is found by chirpc", client: alice, method: http.MethodPost, path: "/api/v1/tasks", body: map[string]any{}, wantStatus: http.StatusBadRequest, wantField: "title"},
		{name: "missing nested text is found by chirpc", client: alice, method: http.MethodPost, path: "/api/v1/tasks", body: map[string]any{"title": "x", "checklist": []map[string]any{{"text": "ok"}, {"done": true}}}, wantStatus: http.StatusBadRequest, wantField: "checklist[1].text"},
		{name: "wrong JSON type is found by chirpc", client: alice, method: http.MethodPost, path: "/api/v1/tasks", body: map[string]any{"title": 42}, wantStatus: http.StatusBadRequest, wantField: "title"},
		{name: "empty title is found by the handler", client: alice, method: http.MethodPost, path: "/api/v1/tasks", body: map[string]any{"title": "   "}, wantStatus: http.StatusBadRequest, wantField: "title"},
		{name: "unknown priority is found by the handler", client: alice, method: http.MethodPost, path: "/api/v1/tasks", body: map[string]any{"title": "x", "priority": "urgent"}, wantStatus: http.StatusBadRequest, wantField: "priority"},
		{name: "bad query number is found by chirpc", client: alice, method: http.MethodGet, path: "/api/v1/tasks?limit=many", wantStatus: http.StatusBadRequest, wantField: "limit"},
		{name: "limit out of range is found by the handler", client: alice, method: http.MethodGet, path: "/api/v1/tasks?limit=500", wantStatus: http.StatusBadRequest, wantField: "limit"},
		{name: "bad task ID", client: alice, method: http.MethodGet, path: "/api/v1/tasks/42", wantStatus: http.StatusBadRequest, wantField: "taskId"},
		{name: "unknown route", client: alice, method: http.MethodGet, path: "/nope", wantStatus: http.StatusNotFound},
		{name: "wrong method", client: alice, method: http.MethodPut, path: "/api/v1/tasks", wantStatus: http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.client.t = t
			var apiErr APIError
			expectStatus(t, tt.client.do(tt.method, tt.path, tt.body, &apiErr), tt.wantStatus)

			if apiErr.Status != tt.wantStatus || apiErr.Message == "" {
				t.Fatalf("The error body is not right: %+v", apiErr)
			}
			if tt.wantField != "" && len(apiErr.Fields[tt.wantField]) == 0 {
				t.Fatalf("The error should name the field %q, but it has %v.", tt.wantField, apiErr.Fields)
			}
		})
	}
}

func TestAccountAndHealth(t *testing.T) {
	router := NewRouter(NewStore())

	var user User
	expectStatus(t, client{t: t, router: router, token: "bob-token"}.do(http.MethodGet, "/api/v1/account/me", nil, &user), http.StatusOK)
	if user.ID != "u_bob" {
		t.Fatalf("The user is %+v, but it should be Bob.", user)
	}

	var health HealthResponse
	expectStatus(t, client{t: t, router: router}.do(http.MethodGet, "/health", nil, &health), http.StatusOK)
	if health.Status != "ok" {
		t.Fatalf("The health status is %q.", health.Status)
	}
}
