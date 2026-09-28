package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	chirpc "github.com/iambpn/chirpc/v1"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// health is a plain AddHandler handler. It reads the raw *http.Request and has no body or query.
func health(r *http.Request) (*chirpc.HttpResponse[HealthResponse], *chirpc.ErrorResponse) {
	return &chirpc.HttpResponse[HealthResponse]{
		Body:    HealthResponse{Status: "ok", Time: time.Now().UTC()},
		Headers: map[string]string{"Cache-Control": "no-store"},
	}, nil
}

// handleError turns every *chirpc.ErrorResponse into an APIError. This includes the 400
// errors that chirpc returns when it cannot decode a body or query.
func handleError(r *http.Request, errResp *chirpc.ErrorResponse) *chirpc.HttpResponse[APIError] {
	status := errResp.StatusCode
	if status == 0 {
		status = http.StatusInternalServerError
	}
	requestID := middleware.GetReqID(r.Context())

	// Cause is never sent to the client, so this is the place to log it.
	if status >= http.StatusInternalServerError && errResp.Cause != nil {
		slog.Error("The request failed.", "requestId", requestID, "path", r.URL.Path, "error", errResp.Cause)
	}

	message := strings.Join(errResp.Errors, " ")
	if message == "" {
		message = http.StatusText(status)
	}
	return &chirpc.HttpResponse[APIError]{
		StatusCode: status,
		Body: APIError{
			Status:    status,
			Message:   message,
			Fields:    errResp.ValidationErrors,
			RequestID: requestID,
		},
	}
}

// getMe returns the signed-in user.
func getMe(req *chirpc.Request[chirpc.NoBody, chirpc.NoQuery]) (*chirpc.HttpResponse[User], error) {
	return &chirpc.HttpResponse[User]{Body: currentUser(req.Context())}, nil
}

// taskHandlers serves the task routes. Each method is a chirpc.TypedHandler, so chirpc
// decodes the body and query before the method runs.
type taskHandlers struct {
	store *Store
}

// list returns one page of the user's tasks, filtered by the query parameters.
func (h *taskHandlers) list(req *chirpc.Request[chirpc.NoBody, ListTasksQuery]) (*chirpc.HttpResponse[TaskPage], error) {
	query := req.Query
	problems := fieldErrors{}
	if query.Status != "" && !query.Status.Valid() {
		problems.add("status", "Use todo, doing, or done.")
	}
	limit := query.Limit
	if limit == 0 {
		limit = defaultPageSize
	}
	if limit < 1 || limit > maxPageSize {
		problems.add("limit", fmt.Sprintf("Use a number from 1 to %d.", maxPageSize))
	}
	if query.Offset < 0 {
		problems.add("offset", "Use a number that is 0 or more.")
	}
	if err := problems.err(); err != nil {
		return nil, err
	}

	tasks := h.store.List(currentUser(req.Context()).ID, taskFilter{
		status:    query.Status,
		tags:      query.Tag,
		dueBefore: query.DueBefore,
	})

	page := TaskPage{Items: []Task{}, Total: len(tasks)}
	if query.Offset < len(tasks) {
		end := min(query.Offset+limit, len(tasks))
		page.Items = tasks[query.Offset:end]
		if end < len(tasks) {
			page.NextOffset = &end
		}
	}
	return &chirpc.HttpResponse[TaskPage]{Body: page}, nil
}

// create adds a task. chirpc already rejected a body without a title or with a checklist
// item without text. This handler checks the rules that chirpc cannot know.
func (h *taskHandlers) create(req *chirpc.Request[CreateTaskBody, chirpc.NoQuery]) (*chirpc.HttpResponse[Task], error) {
	body := req.Body
	problems := fieldErrors{}

	title := strings.TrimSpace(body.Title)
	if title == "" {
		problems.add("title", "The title must not be empty.")
	}
	priority := body.Priority
	if priority == "" {
		priority = PriorityMedium
	} else if !priority.Valid() {
		problems.add("priority", "Use low, medium, or high.")
	}
	checklist := make([]ChecklistItem, 0, len(body.Checklist))
	for i, item := range body.Checklist {
		text := strings.TrimSpace(item.Text)
		if text == "" {
			problems.add(fmt.Sprintf("checklist[%d].text", i), "The text must not be empty.")
		}
		checklist = append(checklist, ChecklistItem{Text: text, Done: item.Done})
	}
	if err := problems.err(); err != nil {
		return nil, err
	}

	task := h.store.Create(currentUser(req.Context()).ID, Task{
		Title:       title,
		Description: body.Description,
		Status:      StatusTodo,
		Priority:    priority,
		Tags:        body.Tags,
		Checklist:   checklist,
		DueAt:       body.DueAt,
	})
	return &chirpc.HttpResponse[Task]{
		StatusCode: http.StatusCreated,
		Body:       task,
		Headers:    map[string]string{"Location": "/api/v1/tasks/" + task.ID.String()},
	}, nil
}

// get returns one task.
func (h *taskHandlers) get(req *chirpc.Request[chirpc.NoBody, chirpc.NoQuery]) (*chirpc.HttpResponse[Task], error) {
	id, err := taskIDParam(req)
	if err != nil {
		return nil, err
	}
	task, err := h.store.Get(currentUser(req.Context()).ID, id)
	if err != nil {
		return nil, storeError(err)
	}
	return &chirpc.HttpResponse[Task]{Body: task}, nil
}

// update changes only the fields that are present in the body.
func (h *taskHandlers) update(req *chirpc.Request[UpdateTaskBody, chirpc.NoQuery]) (*chirpc.HttpResponse[Task], error) {
	id, err := taskIDParam(req)
	if err != nil {
		return nil, err
	}

	body := req.Body
	problems := fieldErrors{}
	if body.Title != nil && strings.TrimSpace(*body.Title) == "" {
		problems.add("title", "The title must not be empty.")
	}
	if body.Status != nil && !body.Status.Valid() {
		problems.add("status", "Use todo, doing, or done.")
	}
	if body.Priority != nil && !body.Priority.Valid() {
		problems.add("priority", "Use low, medium, or high.")
	}
	if err := problems.err(); err != nil {
		return nil, err
	}

	task, err := h.store.Update(currentUser(req.Context()).ID, id, func(task *Task) {
		if body.Title != nil {
			task.Title = strings.TrimSpace(*body.Title)
		}
		if body.Description != nil {
			task.Description = *body.Description
		}
		if body.Status != nil {
			task.Status = *body.Status
		}
		if body.Priority != nil {
			task.Priority = *body.Priority
		}
		if body.Tags != nil {
			task.Tags = *body.Tags
		}
		if body.DueAt != nil {
			task.DueAt = body.DueAt
		}
	})
	if err != nil {
		return nil, storeError(err)
	}
	return &chirpc.HttpResponse[Task]{Body: task}, nil
}

// delete removes a task. It returns no response, so chirpc sends 204 No Content.
func (h *taskHandlers) delete(req *chirpc.Request[chirpc.NoBody, chirpc.NoQuery]) (*chirpc.HttpResponse[chirpc.NoBody], error) {
	id, err := taskIDParam(req)
	if err != nil {
		return nil, err
	}
	if err := h.store.Delete(currentUser(req.Context()).ID, id); err != nil {
		return nil, storeError(err)
	}
	return nil, nil
}

// taskIDParam reads the {taskId} path parameter.
func taskIDParam[Body, Query any](req *chirpc.Request[Body, Query]) (TaskID, error) {
	id, err := ParseTaskID(req.Param("taskId"))
	if err != nil {
		return 0, &chirpc.ErrorResponse{
			StatusCode:       http.StatusBadRequest,
			Errors:           []string{"The task ID is not valid."},
			ValidationErrors: map[string][]string{"taskId": {"Use an ID such as task_1."}},
			Cause:            err,
		}
	}
	return id, nil
}

// storeError maps store errors to responses. Any other error is returned as it is,
// and chirpc sends it as a general 500 error without its details.
func storeError(err error) error {
	if errors.Is(err, ErrTaskNotFound) {
		return &chirpc.ErrorResponse{
			StatusCode: http.StatusNotFound,
			Errors:     []string{"This task does not exist."},
			Cause:      err,
		}
	}
	return err
}

// fieldErrors collects validation messages by field name.
type fieldErrors map[string][]string

func (f fieldErrors) add(field, message string) {
	f[field] = append(f[field], message)
}

// err returns a 400 ErrorResponse with the collected messages, or nil when there are none.
func (f fieldErrors) err() error {
	if len(f) == 0 {
		return nil
	}
	return &chirpc.ErrorResponse{
		StatusCode:       http.StatusBadRequest,
		Errors:           []string{"The request is not valid."},
		ValidationErrors: f,
	}
}
