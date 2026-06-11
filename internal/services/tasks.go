package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sombi/pi-google-services/internal/mcp"
	"github.com/sombi/pi-google-services/internal/tasks"
)

// TasksService implements the Service interface for Google Tasks.
type TasksService struct {
	api *tasks.Service
}

// NewTasks creates a TasksService from the Tasks API wrapper.
func NewTasks(api *tasks.Service) *TasksService {
	return &TasksService{api: api}
}

func (s *TasksService) Name() string { return "tasks" }

func (s *TasksService) Scopes() []string {
	return []string{"https://www.googleapis.com/auth/tasks"}
}

func (s *TasksService) Tools() []mcp.ToolDefinition {
	return []mcp.ToolDefinition{
		{
			Name:        "list-tasklists",
			Description: "List all task lists",
			InputSchema: mcp.InputSchema{Type: "object", Properties: map[string]mcp.PropertySchema{}},
		},
		{
			Name:        "list-tasks",
			Description: "List tasks (pending, completed, or all)",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"taskListId": {Type: "string", Description: "Task list ID (default: @default)"},
					"status":     {Type: "string", Description: "Filter: pending, completed, or '' for all"},
					"maxResults": {Type: "number", Description: "Max results (default: 50)", Default: 50},
				},
			},
		},
		{
			Name:        "create-task",
			Description: "Create a new task",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"taskListId": {Type: "string", Description: "Task list ID (default: @default)"},
					"title":      {Type: "string", Description: "Task title"},
					"notes":      {Type: "string", Description: "Optional notes"},
					"dueDate":    {Type: "string", Description: "Due date (YYYY-MM-DD)"},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "complete-task",
			Description: "Mark a task as completed",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"taskListId": {Type: "string", Description: "Task list ID (default: @default)"},
					"taskId":     {Type: "string", Description: "Task ID to complete"},
				},
				Required: []string{"taskId"},
			},
		},
		{
			Name:        "delete-task",
			Description: "Delete a task",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"taskListId": {Type: "string", Description: "Task list ID (default: @default)"},
					"taskId":     {Type: "string", Description: "Task ID to delete"},
				},
				Required: []string{"taskId"},
			},
		},
	}
}

func (s *TasksService) Handle(ctx context.Context, toolName string, params json.RawMessage) (interface{}, *mcp.RPCError) {
	switch toolName {
	case "list-tasklists":
		return s.handleListTaskLists(ctx)
	case "list-tasks":
		return s.handleListTasks(ctx, params)
	case "create-task":
		return s.handleCreateTask(ctx, params)
	case "complete-task":
		return s.handleCompleteTask(ctx, params)
	case "delete-task":
		return s.handleDeleteTask(ctx, params)
	default:
		return nil, &mcp.RPCError{Code: -32601, Message: fmt.Sprintf("Tasks tool not found: %s", toolName)}
	}
}

func (s *TasksService) handleListTaskLists(ctx context.Context) (interface{}, *mcp.RPCError) {
	lists, err := s.api.ListTaskLists(ctx)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to list task lists", Data: err.Error()}
	}
	var b strings.Builder
	if len(lists) == 0 {
		b.WriteString("No task lists found.")
	} else {
		for _, l := range lists {
			b.WriteString(fmt.Sprintf("📋 %s\n  ID: %s\n", l.Title, l.ID))
		}
	}
	return contentResponse(b.String()), nil
}

func (s *TasksService) handleListTasks(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		TaskListID string `json:"taskListId"`
		Status     string `json:"status"`
		MaxResults int64  `json:"maxResults"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}

	items, err := s.api.ListTasks(ctx, args.TaskListID, args.Status, args.MaxResults)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to list tasks", Data: err.Error()}
	}

	var b strings.Builder
	if len(items) == 0 {
		b.WriteString("No tasks found.")
	} else {
		for i, t := range items {
			status := "⬜"
			if t.Status == "completed" {
				status = "✅"
			}
			b.WriteString(fmt.Sprintf("%d. %s %s", i+1, status, t.Title))
			if t.Due != "" {
				b.WriteString(fmt.Sprintf(" (due: %s)", t.Due))
			}
			if t.Notes != "" {
				trunc := t.Notes
				if len(trunc) > 80 {
					trunc = trunc[:80] + "..."
				}
				b.WriteString(fmt.Sprintf("\n   📝 %s", trunc))
			}
			b.WriteString("\n")
		}
	}
	return contentResponse(b.String()), nil
}

func (s *TasksService) handleCreateTask(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		TaskListID string `json:"taskListId"`
		Title      string `json:"title"`
		Notes      string `json:"notes"`
		DueDate    string `json:"dueDate"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.Title == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "title required"}
	}

	created, err := s.api.CreateTask(ctx, args.TaskListID, args.Title, args.Notes, args.DueDate)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to create task", Data: err.Error()}
	}

	dueInfo := ""
	if created.Due != "" {
		dueInfo = fmt.Sprintf("\n   Due: %s", created.Due)
	}
	return contentResponse(fmt.Sprintf("✅ Task created: %s%s\n   ID: %s", created.Title, dueInfo, created.Id)), nil
}

func (s *TasksService) handleCompleteTask(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		TaskListID string `json:"taskListId"`
		TaskID     string `json:"taskId"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.TaskID == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "taskId required"}
	}

	updated, err := s.api.CompleteTask(ctx, args.TaskListID, args.TaskID)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to complete task", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ Task completed: %s", updated.Title)), nil
}

func (s *TasksService) handleDeleteTask(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		TaskListID string `json:"taskListId"`
		TaskID     string `json:"taskId"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.TaskID == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "taskId required"}
	}

	if err := s.api.DeleteTask(ctx, args.TaskListID, args.TaskID); err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to delete task", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ Task deleted (ID: %s)", args.TaskID)), nil
}
