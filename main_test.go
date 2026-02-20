package main

import (
	"encoding/json"
	"testing"
)

func TestStdinDataParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantCWD string
	}{
		{
			name:    "cwd field",
			input:   `{"cwd":"/Users/test/project"}`,
			wantCWD: "/Users/test/project",
		},
		{
			name:    "workspace.current_dir field",
			input:   `{"workspace":{"current_dir":"/Users/test/project"}}`,
			wantCWD: "/Users/test/project",
		},
		{
			name:    "cwd takes precedence over workspace",
			input:   `{"cwd":"/primary","workspace":{"current_dir":"/fallback"}}`,
			wantCWD: "/primary",
		},
		{
			name:    "empty cwd falls back to workspace",
			input:   `{"cwd":"","workspace":{"current_dir":"/fallback"}}`,
			wantCWD: "/fallback",
		},
		{
			name:    "both empty",
			input:   `{"cwd":"","workspace":{"current_dir":""}}`,
			wantCWD: "",
		},
		{
			name:    "no relevant fields",
			input:   `{"model":"claude-sonnet"}`,
			wantCWD: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var data stdinData
			if err := json.Unmarshal([]byte(tt.input), &data); err != nil {
				t.Fatalf("json.Unmarshal() error: %v", err)
			}

			cwd := data.CWD
			if cwd == "" {
				cwd = data.Workspace.CurrentDir
			}

			if cwd != tt.wantCWD {
				t.Errorf("resolved cwd = %q, want %q", cwd, tt.wantCWD)
			}
		})
	}
}
