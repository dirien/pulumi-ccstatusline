package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// stdinData represents the relevant fields from the Claude Code JSON piped via stdin.
type stdinData struct {
	CWD       string `json:"cwd"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
	} `json:"workspace"`
}

func main() {
	ctx := context.Background()

	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return
	}

	var input stdinData
	if err := json.Unmarshal(raw, &input); err != nil {
		return
	}

	cwd := input.CWD
	if cwd == "" {
		cwd = input.Workspace.CurrentDir
	}
	if cwd == "" {
		return
	}

	// Check for Pulumi project
	if _, err := os.Stat(filepath.Join(cwd, "Pulumi.yaml")); err != nil {
		return
	}

	data := getPulumiData(ctx, cwd)
	if data == nil {
		fmt.Println(colorize(colorYellow, "☁ No stack selected"))
		return
	}

	sep := colorize(colorDim, " | ")

	parts := []string{colorize(colorMagenta, "☁ "+data.StackName)}

	parts = append(parts, colorize(colorCyan, fmt.Sprintf("%d resources", data.ResourceCount)))

	if data.LastStatus != "" {
		parts = append(parts, colorizeStatus(data.LastStatus))
	}

	if !data.LastUpdate.IsZero() {
		parts = append(parts, colorize(colorYellow, formatRelativeTime(data.LastUpdate)))
	}

	fmt.Println(strings.Join(parts, sep))
}
