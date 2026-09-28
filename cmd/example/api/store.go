package api

import (
	"cmp"
	"errors"
	"slices"
	"sync"
	"time"
)

// ErrTaskNotFound is returned when a task does not exist or belongs to another user.
var ErrTaskNotFound = errors.New("task not found")

// Store keeps tasks in memory. It is safe for concurrent use.
// A real application would use a database here.
type Store struct {
	mu     sync.Mutex
	nextID TaskID
	tasks  map[TaskID]*Task
	now    func() time.Time
}

// NewStore returns an empty store.
func NewStore() *Store {
	return &Store{
		tasks: make(map[TaskID]*Task),
		now:   time.Now,
	}
}

// taskFilter selects the tasks returned by List.
type taskFilter struct {
	status    Status
	tags      []string
	dueBefore *time.Time
}

// matches reports whether the task passes every filter that is set.
func (f taskFilter) matches(task *Task) bool {
	if f.status != "" && task.Status != f.status {
		return false
	}
	for _, tag := range f.tags {
		if !slices.Contains(task.Tags, tag) {
			return false
		}
	}
	if f.dueBefore != nil && (task.DueAt == nil || !task.DueAt.Before(*f.dueBefore)) {
		return false
	}
	return true
}

// List returns the owner's tasks that match the filter, oldest first.
func (s *Store) List(ownerID string, filter taskFilter) []Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []Task
	for _, task := range s.tasks {
		if task.ownerID == ownerID && filter.matches(task) {
			result = append(result, cloneTask(task))
		}
	}
	slices.SortFunc(result, func(a, b Task) int { return cmp.Compare(a.ID, b.ID) })
	return result
}

// Create saves a new task for the owner and returns it.
func (s *Store) Create(ownerID string, task Task) Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	now := s.now().UTC()
	task.ID = s.nextID
	task.ownerID = ownerID
	task.CreatedAt = now
	task.UpdatedAt = now
	s.tasks[task.ID] = &task
	return cloneTask(&task)
}

// Get returns the owner's task with the given ID.
func (s *Store) Get(ownerID string, id TaskID) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok || task.ownerID != ownerID {
		return Task{}, ErrTaskNotFound
	}
	return cloneTask(task), nil
}

// Update changes the owner's task with the given function and returns the result.
func (s *Store) Update(ownerID string, id TaskID, change func(*Task)) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok || task.ownerID != ownerID {
		return Task{}, ErrTaskNotFound
	}
	change(task)
	task.UpdatedAt = s.now().UTC()
	return cloneTask(task), nil
}

// Delete removes the owner's task with the given ID.
func (s *Store) Delete(ownerID string, id TaskID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok || task.ownerID != ownerID {
		return ErrTaskNotFound
	}
	delete(s.tasks, id)
	return nil
}

// cloneTask copies a task, so callers cannot change the stored slices and pointers.
func cloneTask(task *Task) Task {
	out := *task
	out.Tags = slices.Clone(task.Tags)
	out.Checklist = slices.Clone(task.Checklist)
	if task.DueAt != nil {
		dueAt := *task.DueAt
		out.DueAt = &dueAt
	}
	if out.Tags == nil {
		out.Tags = []string{}
	}
	if out.Checklist == nil {
		out.Checklist = []ChecklistItem{}
	}
	return out
}
