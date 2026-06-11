// Package tasks wraps the Google Tasks API v1.
package tasks

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

// Service wraps the Google Tasks API client.
type Service struct {
	lists *tasks.TasklistsService
	tasks *tasks.TasksService
}

// TaskSummary is a lightweight task representation.
type TaskSummary struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Notes   string `json:"notes,omitempty"`
	Due     string `json:"due,omitempty"`
	Status  string `json:"status"`
	Updated string `json:"updated"`
}

// TaskListSummary summarizes a task list.
type TaskListSummary struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// New creates a Service from an OAuth2 token source.
func New(ctx context.Context, ts oauth2.TokenSource) (*Service, error) {
	svc, err := tasks.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("create tasks service: %w", err)
	}
	return &Service{
		lists: tasks.NewTasklistsService(svc),
		tasks: tasks.NewTasksService(svc),
	}, nil
}

// ListTaskLists returns all task lists.
func (s *Service) ListTaskLists(ctx context.Context) ([]*TaskListSummary, error) {
	res, err := s.lists.List().Do()
	if err != nil {
		return nil, fmt.Errorf("list tasklists: %w", err)
	}
	summaries := make([]*TaskListSummary, 0, len(res.Items))
	for _, l := range res.Items {
		summaries = append(summaries, &TaskListSummary{ID: l.Id, Title: l.Title})
	}
	return summaries, nil
}

// ListTasks returns tasks in a task list, optionally filtered by status.
func (s *Service) ListTasks(ctx context.Context, taskListID, status string, maxResults int64) ([]*TaskSummary, error) {
	if taskListID == "" {
		taskListID = "@default"
	}
	if maxResults <= 0 {
		maxResults = 50
	}

	call := s.tasks.List(taskListID).MaxResults(maxResults)
	switch status {
	case "completed":
		call.ShowCompleted(true).ShowHidden(false)
	case "pending":
		call.ShowCompleted(false).ShowHidden(false)
	}

	res, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	summaries := make([]*TaskSummary, 0, len(res.Items))
	for _, t := range res.Items {
		s := &TaskSummary{
			ID:     t.Id,
			Title:  t.Title,
			Status: t.Status,
		}
		if t.Notes != "" {
			s.Notes = t.Notes
		}
		if t.Due != "" {
			s.Due = fmtDue(t.Due)
		}
		if t.Updated != "" {
			s.Updated = t.Updated
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// CreateTask creates a new task with optional due date (YYYY-MM-DD).
func (s *Service) CreateTask(ctx context.Context, taskListID, title, notes, dueDate string) (*tasks.Task, error) {
	if taskListID == "" {
		taskListID = "@default"
	}
	task := &tasks.Task{
		Title: title,
		Notes: notes,
	}
	if dueDate != "" {
		task.Due = dueDate
	}
	created, err := s.tasks.Insert(taskListID, task).Do()
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return created, nil
}

// UpdateTask updates title, notes, status, or due date.
func (s *Service) UpdateTask(ctx context.Context, taskListID, taskID, title, notes, status, dueDate string) (*tasks.Task, error) {
	if taskListID == "" {
		taskListID = "@default"
	}
	task := &tasks.Task{
		Id:     taskID,
		Title:  title,
		Notes:  notes,
		Status: status,
	}
	if dueDate != "" {
		task.Due = dueDate
	}
	updated, err := s.tasks.Update(taskListID, taskID, task).Do()
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	return updated, nil
}

// CompleteTask marks a task as done.
func (s *Service) CompleteTask(ctx context.Context, taskListID, taskID string) (*tasks.Task, error) {
	if taskListID == "" {
		taskListID = "@default"
	}
	current, err := s.tasks.Get(taskListID, taskID).Do()
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	current.Status = "completed"
	updated, err := s.tasks.Update(taskListID, taskID, current).Do()
	if err != nil {
		return nil, fmt.Errorf("complete task: %w", err)
	}
	return updated, nil
}

// DeleteTask removes a task.
func (s *Service) DeleteTask(ctx context.Context, taskListID, taskID string) error {
	if taskListID == "" {
		taskListID = "@default"
	}
	return s.tasks.Delete(taskListID, taskID).Do()
}

func fmtDue(raw string) string {
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return raw
	}
	return t.Format("Mon Jan 2")
}
