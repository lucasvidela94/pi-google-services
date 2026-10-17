package services

import (
	"testing"

	"github.com/sombi/pi-google-services/internal/tasks"
)

func TestFormatTaskSummaries(t *testing.T) {
	tests := []struct {
		name  string
		items []*tasks.TaskSummary
		want  string
	}{
		{
			name: "empty list",
			want: "No tasks found.",
		},
		{
			name: "pending task includes ID",
			items: []*tasks.TaskSummary{{
				ID:     "task-123",
				Title:  "heyyo",
				Status: "needsAction",
			}},
			want: "1. ⬜ heyyo (ID: task-123)\n",
		},
		{
			name: "completed task includes ID",
			items: []*tasks.TaskSummary{{
				ID:     "task-456",
				Title:  "done",
				Status: "completed",
			}},
			want: "1. ✅ done (ID: task-456)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatTaskSummaries(tt.items); got != tt.want {
				t.Errorf("formatTaskSummaries() = %q, want %q", got, tt.want)
			}
		})
	}
}
