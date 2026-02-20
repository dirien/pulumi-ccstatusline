package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PulumiData holds the cached Pulumi stack information.
type PulumiData struct {
	LastUpdate     time.Time `json:"last_update"`
	FetchedAt      time.Time `json:"fetched_at"`
	ProjectName    string    `json:"project_name"`
	StackName      string    `json:"stack_name"`
	LastStatus     string    `json:"last_status"`
	WorkspaceMtime int64     `json:"workspace_mtime"`
	ResourceCount  int       `json:"resource_count"`
}

const (
	cacheTTL       = 30 * time.Second
	commandTimeout = 10 * time.Second
)

func cacheFilePath(cwd string) string {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(cwd)))
	return filepath.Join(os.TempDir(), fmt.Sprintf("pulumi-ccstatusline-%s.json", hash))
}

// workspaceMtime returns the modification time of the Pulumi workspace file
// that tracks the currently selected stack. Returns 0 if not found.
func workspaceMtime(cwd string) int64 {
	// Pulumi workspace files live in ~/.pulumi/workspaces/<project>-<hash>-workspace.json.
	// The project name comes from Pulumi.yaml. We glob for a matching file
	// rather than recomputing Pulumi's internal hash.
	wsDir := filepath.Join(os.Getenv("HOME"), ".pulumi", "workspaces")
	projectName := readProjectName(cwd)
	if projectName == "" {
		return 0
	}

	matches, err := filepath.Glob(filepath.Join(wsDir, projectName+"-*-workspace.json"))
	if err != nil || len(matches) == 0 {
		return 0
	}

	info, err := os.Stat(matches[0])
	if err != nil {
		return 0
	}

	return info.ModTime().UnixNano()
}

// readProjectName extracts the "name" field from Pulumi.yaml.
func readProjectName(cwd string) string {
	data, err := os.ReadFile(filepath.Join(cwd, "Pulumi.yaml"))
	if err != nil {
		return ""
	}

	// Simple regex-free extraction: find the "name:" line.
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if val, ok := strings.CutPrefix(line, "name:"); ok {
			return strings.TrimSpace(val)
		}
	}

	return ""
}

func readCache(cwd string, currentMtime int64) *PulumiData {
	data, err := os.ReadFile(cacheFilePath(cwd))
	if err != nil {
		return nil
	}

	var cached PulumiData
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil
	}

	// Invalidate if the workspace file changed (stack switch/delete).
	if cached.WorkspaceMtime != currentMtime {
		return nil
	}

	// Invalidate if TTL expired.
	if time.Since(cached.FetchedAt) > cacheTTL {
		return nil
	}

	return &cached
}

func writeCache(cwd string, data *PulumiData) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	_ = os.WriteFile(cacheFilePath(cwd), raw, 0o600)
}

func getPulumiData(ctx context.Context, cwd string) *PulumiData {
	mtime := workspaceMtime(cwd)

	if cached := readCache(cwd, mtime); cached != nil {
		return cached
	}

	data := fillStackListData(ctx, cwd)
	if data == nil {
		return nil
	}

	data.ProjectName = readProjectName(cwd)
	fillHistoryStatus(ctx, cwd, data)

	data.WorkspaceMtime = mtime
	writeCache(cwd, data)
	return data
}

// stackListEntry represents one entry from `pulumi stack ls --json`.
type stackListEntry struct {
	Name             string `json:"name"`
	LastUpdate       string `json:"lastUpdate"`
	URL              string `json:"url"`
	ResourceCount    int    `json:"resourceCount"`
	Current          bool   `json:"current"`
	UpdateInProgress bool   `json:"updateInProgress"`
}

// fillStackListData uses `pulumi stack ls --json` to get the current stack's
// name, resource count, and last update time in a single CLI call.
func fillStackListData(ctx context.Context, cwd string) *PulumiData {
	out, err := runPulumi(ctx, cwd, "stack", "ls", "--json")
	if err != nil {
		return nil
	}

	var entries []stackListEntry
	if err := json.Unmarshal([]byte(out), &entries); err != nil {
		return nil
	}

	// Find the current stack
	for _, entry := range entries {
		if !entry.Current {
			continue
		}

		data := &PulumiData{
			StackName:     extractStackName(entry.Name),
			ResourceCount: entry.ResourceCount,
			FetchedAt:     time.Now(),
		}

		if entry.LastUpdate != "" {
			if t, err := time.Parse(time.RFC3339, entry.LastUpdate); err == nil {
				data.LastUpdate = t
			}
		}

		return data
	}

	return nil
}

// extractStackName returns just the stack portion from a potentially
// fully-qualified name like "org/project/stack" or "org/stack".
func extractStackName(name string) string {
	parts := strings.Split(name, "/")
	return parts[len(parts)-1]
}

// historyEntry represents one entry from `pulumi stack history --json`.
type historyEntry struct {
	Result string `json:"result"`
}

// fillHistoryStatus uses `pulumi stack history --json` to get the last
// operation's result status (succeeded/failed).
func fillHistoryStatus(ctx context.Context, cwd string, data *PulumiData) {
	out, err := runPulumi(ctx, cwd, "stack", "history", "--json", "--page-size", "1")
	if err != nil {
		return
	}

	var entries []historyEntry
	if err := json.Unmarshal([]byte(out), &entries); err != nil || len(entries) == 0 {
		return
	}

	data.LastStatus = entries[0].Result
}

func runPulumi(ctx context.Context, cwd string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	args = append(args, "--cwd", cwd)
	cmd := exec.CommandContext(ctx, "pulumi", args...) //nolint:gosec // args are constructed internally, not from user input
	cmd.Env = os.Environ()

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pulumi %s: %w", args[0], err)
	}

	return string(out), nil
}
