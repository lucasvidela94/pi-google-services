package tasks

import "testing"

func TestNormalizeDueDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "date-only value becomes RFC 3339 midnight UTC",
			input: "2026-10-26",
			want:  "2026-10-26T00:00:00Z",
		},
		{
			name:  "RFC 3339 datetime is preserved",
			input: "2026-10-26T14:13:37+03:00",
			want:  "2026-10-26T14:13:37+03:00",
		},
		{
			name:    "invalid date is rejected",
			input:   "2026-02-30",
			wantErr: true,
		},
		{
			name:    "unsupported format is rejected",
			input:   "tomorrow",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeDueDate(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeDueDate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("normalizeDueDate() = %q, want %q", got, tt.want)
			}
		})
	}
}
