package main

import (
	"testing"
	"time"
)

func TestFormatRelativeTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		when time.Time
		want string
	}{
		{
			name: "just now",
			when: time.Now().Add(-30 * time.Second),
			want: "just now",
		},
		{
			name: "minutes ago",
			when: time.Now().Add(-5 * time.Minute),
			want: "5m ago",
		},
		{
			name: "one minute ago",
			when: time.Now().Add(-1 * time.Minute),
			want: "1m ago",
		},
		{
			name: "hours ago",
			when: time.Now().Add(-3 * time.Hour),
			want: "3h ago",
		},
		{
			name: "one hour ago",
			when: time.Now().Add(-1 * time.Hour),
			want: "1h ago",
		},
		{
			name: "days ago",
			when: time.Now().Add(-10 * 24 * time.Hour),
			want: "10d ago",
		},
		{
			name: "one day ago",
			when: time.Now().Add(-25 * time.Hour),
			want: "1d ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := formatRelativeTime(tt.when)
			if got != tt.want {
				t.Errorf("formatRelativeTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status string
		want   string
	}{
		{
			name:   "succeeded",
			status: "succeeded",
			want:   "✓ succeeded",
		},
		{
			name:   "failed",
			status: "failed",
			want:   "✗ failed",
		},
		{
			name:   "unknown status passed through",
			status: "in_progress",
			want:   "in_progress",
		},
		{
			name:   "empty status passed through",
			status: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := formatStatus(tt.status)
			if got != tt.want {
				t.Errorf("formatStatus(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}
