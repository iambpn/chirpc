package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TaskID identifies a task. It is sent as a string such as "task_7", so the router
// registers it as a TypeScript string with chirpc.RegisterTSType.
type TaskID uint64

const taskIDPrefix = "task_"

// ParseTaskID reads a task ID in the "task_7" form.
func ParseTaskID(value string) (TaskID, error) {
	number, ok := strings.CutPrefix(value, taskIDPrefix)
	if !ok {
		return 0, fmt.Errorf("the task ID %q must start with %q", value, taskIDPrefix)
	}
	id, err := strconv.ParseUint(number, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("the task ID %q is not valid", value)
	}
	return TaskID(id), nil
}

// String returns the ID in the "task_7" form.
func (id TaskID) String() string {
	return taskIDPrefix + strconv.FormatUint(uint64(id), 10)
}

// MarshalText makes encoding/json/v2 send the ID as a string.
func (id TaskID) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

// UnmarshalText makes encoding/json/v2 read the ID from a string.
func (id *TaskID) UnmarshalText(text []byte) error {
	parsed, err := ParseTaskID(string(text))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

// Status is the stage of a task. The router registers it as a TypeScript union of its values.
type Status string

const (
	StatusTodo  Status = "todo"
	StatusDoing Status = "doing"
	StatusDone  Status = "done"
)

// Valid reports whether s is one of the known statuses.
func (s Status) Valid() bool {
	return s == StatusTodo || s == StatusDoing || s == StatusDone
}

// Priority tells how urgent a task is. The router registers it as a TypeScript union of its values.
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// Valid reports whether p is one of the known priorities.
func (p Priority) Valid() bool {
	return p == PriorityLow || p == PriorityMedium || p == PriorityHigh
}

// User is the person who made the request.
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ChecklistItem is one step inside a task.
type ChecklistItem struct {
	Text string `json:"text"`
	Done bool   `json:"done"`
}

// Task is the main resource of the API.
type Task struct {
	ID          TaskID          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      Status          `json:"status"`
	Priority    Priority        `json:"priority"`
	Tags        []string        `json:"tags"`
	Checklist   []ChecklistItem `json:"checklist"`
	DueAt       *time.Time      `json:"dueAt"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`

	// ownerID is not exported, so it is never sent to the client or put in the TypeScript.
	ownerID string
}

// TaskPage is one page of tasks from the list endpoint.
type TaskPage struct {
	Items []Task `json:"items"`
	Total int    `json:"total"`
	// NextOffset is the offset of the next page, or null on the last page.
	NextOffset *int `json:"nextOffset"`
}

// ListTasksQuery holds the query parameters of GET /api/v1/tasks.
// Every field is optional, so the generated TypeScript uses "query?".
type ListTasksQuery struct {
	Status    Status     `json:"status,omitempty"`
	Tag       []string   `json:"tag,omitempty"` // Repeat the key to filter by more tags: ?tag=a&tag=b.
	DueBefore *time.Time `json:"dueBefore,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// CreateTaskBody is the JSON body of POST /api/v1/tasks. Only title is required.
type CreateTaskBody struct {
	Title       string               `json:"title"`
	Description string               `json:"description,omitempty"`
	Priority    Priority             `json:"priority,omitempty"`
	Tags        []string             `json:"tags,omitempty"`
	DueAt       *time.Time           `json:"dueAt,omitempty"`
	Checklist   []ChecklistItemInput `json:"checklist,omitempty"`
}

// ChecklistItemInput is a checklist item in a request. chirpc reports a missing text
// with a path such as "checklist[1].text".
type ChecklistItemInput struct {
	Text string `json:"text"`
	Done bool   `json:"done,omitempty"`
}

// UpdateTaskBody is the JSON body of PATCH /api/v1/tasks/{taskId}.
// Every field is a pointer, so a field that is left out keeps its current value.
type UpdateTaskBody struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	Status      *Status    `json:"status,omitempty"`
	Priority    *Priority  `json:"priority,omitempty"`
	Tags        *[]string  `json:"tags,omitempty"`
	DueAt       *time.Time `json:"dueAt,omitempty"`
}

// HealthResponse is the body of GET /health.
type HealthResponse struct {
	Status string    `json:"status"`
	Time   time.Time `json:"time"`
}

// APIError is the body of every error response. The router's error handler builds it,
// and the generated TypeScript uses it as the ERROR_HANDLER response.
type APIError struct {
	Status    int                 `json:"status"`
	Message   string              `json:"message"`
	Fields    map[string][]string `json:"fields,omitempty"`
	RequestID string              `json:"requestId,omitempty"`
}
