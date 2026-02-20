package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractStackName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple name",
			input: "dev",
			want:  "dev",
		},
		{
			name:  "org/stack format",
			input: "myorg/dev",
			want:  "dev",
		},
		{
			name:  "org/project/stack format",
			input: "myorg/myproject/production",
			want:  "production",
		},
		{
			name:  "single segment",
			input: "staging",
			want:  "staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := extractStackName(tt.input)
			if got != tt.want {
				t.Errorf("extractStackName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCacheFilePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cwd  string
	}{
		{
			name: "basic path",
			cwd:  "/Users/test/project",
		},
		{
			name: "different path produces different hash",
			cwd:  "/Users/test/other-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := cacheFilePath(tt.cwd)

			if !filepath.IsAbs(path) {
				t.Errorf("cacheFilePath(%q) = %q, want absolute path", tt.cwd, path)
			}
			if filepath.Ext(path) != ".json" {
				t.Errorf("cacheFilePath(%q) = %q, want .json extension", tt.cwd, path)
			}
		})
	}

	// Verify different paths produce different cache files.
	path1 := cacheFilePath("/path/a")
	path2 := cacheFilePath("/path/b")
	if path1 == path2 {
		t.Errorf("different cwds should produce different cache paths, got same: %q", path1)
	}
}

func TestCacheRoundTrip(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()

	data := &PulumiData{
		StackName:      "dev",
		ResourceCount:  42,
		LastStatus:     "succeeded",
		LastUpdate:     time.Now().Truncate(time.Second),
		FetchedAt:      time.Now().Truncate(time.Second),
		WorkspaceMtime: 1234567890,
	}

	writeCache(cwd, data)

	// Same mtime → cache hit.
	got := readCache(cwd, 1234567890)
	if got == nil {
		t.Fatal("readCache() returned nil, want cached data")
	}

	if got.StackName != data.StackName {
		t.Errorf("StackName = %q, want %q", got.StackName, data.StackName)
	}
	if got.ResourceCount != data.ResourceCount {
		t.Errorf("ResourceCount = %d, want %d", got.ResourceCount, data.ResourceCount)
	}
	if got.LastStatus != data.LastStatus {
		t.Errorf("LastStatus = %q, want %q", got.LastStatus, data.LastStatus)
	}
}

func TestReadCacheInvalidatedByMtime(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()

	data := &PulumiData{
		StackName:      "dev",
		FetchedAt:      time.Now(),
		WorkspaceMtime: 1000,
	}

	writeCache(cwd, data)

	// Different mtime → cache miss (stack was switched).
	got := readCache(cwd, 2000)
	if got != nil {
		t.Errorf("readCache() = %v, want nil when workspace mtime changed", got)
	}
}

func TestReadCacheInvalidatedByWorkspaceRemoval(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()

	data := &PulumiData{
		StackName:      "prod",
		FetchedAt:      time.Now(),
		WorkspaceMtime: 1000,
	}

	writeCache(cwd, data)

	// Workspace file removed (mtime 0) but cache had mtime 1000 → invalidate.
	got := readCache(cwd, 0)
	if got != nil {
		t.Errorf("readCache() = %v, want nil when workspace file was removed", got)
	}
}

func TestReadCacheBothZeroMtime(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()

	data := &PulumiData{
		StackName:      "dev",
		FetchedAt:      time.Now(),
		WorkspaceMtime: 0,
	}

	writeCache(cwd, data)

	// Both zero → mtime matches, use TTL only.
	got := readCache(cwd, 0)
	if got == nil {
		t.Error("readCache() returned nil, want cached data when both mtimes are 0")
	}
}

func TestReadCacheExpired(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()

	data := &PulumiData{
		StackName:      "dev",
		FetchedAt:      time.Now().Add(-2 * cacheTTL),
		WorkspaceMtime: 1000,
	}

	writeCache(cwd, data)

	// Same mtime but TTL expired → cache miss.
	got := readCache(cwd, 1000)
	if got != nil {
		t.Errorf("readCache() = %v, want nil for expired cache", got)
	}
}

func TestReadCacheMissing(t *testing.T) {
	t.Parallel()

	got := readCache("/nonexistent/path/that/should/not/exist", 0)
	if got != nil {
		t.Errorf("readCache() = %v, want nil for missing cache file", got)
	}
}

func TestReadCacheCorrupt(t *testing.T) {
	t.Parallel()

	cwd := t.TempDir()
	cachePath := cacheFilePath(cwd)

	if err := os.WriteFile(cachePath, []byte("not-json"), 0600); err != nil {
		t.Fatalf("failed to write corrupt cache: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(cachePath) })

	got := readCache(cwd, 0)
	if got != nil {
		t.Errorf("readCache() = %v, want nil for corrupt cache", got)
	}
}

func TestReadProjectName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "simple project",
			content: "name: my-project\nruntime: go\n",
			want:    "my-project",
		},
		{
			name:    "with runtime options",
			content: "name: pulumi-voting-app\nruntime:\n  name: nodejs\n",
			want:    "pulumi-voting-app",
		},
		{
			name:    "empty file",
			content: "",
			want:    "",
		},
		{
			name:    "no name field",
			content: "runtime: go\ndescription: test\n",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if tt.content != "" {
				if err := os.WriteFile(filepath.Join(dir, "Pulumi.yaml"), []byte(tt.content), 0600); err != nil {
					t.Fatalf("failed to write Pulumi.yaml: %v", err)
				}
			}

			got := readProjectName(dir)
			if got != tt.want {
				t.Errorf("readProjectName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPulumiDataJSON(t *testing.T) {
	t.Parallel()

	data := &PulumiData{
		StackName:      "production",
		ResourceCount:  10,
		LastStatus:     "succeeded",
		LastUpdate:     time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		FetchedAt:      time.Date(2025, 1, 15, 10, 31, 0, 0, time.UTC),
		WorkspaceMtime: 9999,
	}

	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got PulumiData
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.StackName != data.StackName {
		t.Errorf("StackName = %q, want %q", got.StackName, data.StackName)
	}
	if got.ResourceCount != data.ResourceCount {
		t.Errorf("ResourceCount = %d, want %d", got.ResourceCount, data.ResourceCount)
	}
	if got.LastStatus != data.LastStatus {
		t.Errorf("LastStatus = %q, want %q", got.LastStatus, data.LastStatus)
	}
	if !got.LastUpdate.Equal(data.LastUpdate) {
		t.Errorf("LastUpdate = %v, want %v", got.LastUpdate, data.LastUpdate)
	}
	if got.WorkspaceMtime != data.WorkspaceMtime {
		t.Errorf("WorkspaceMtime = %d, want %d", got.WorkspaceMtime, data.WorkspaceMtime)
	}
}
