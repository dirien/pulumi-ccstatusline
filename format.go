// Package main provides a Claude Code status line tool that displays Pulumi stack information.
package main

import (
	"fmt"
	"time"
)

// ANSI color codes.
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[0;31m"
	colorGreen   = "\033[0;32m"
	colorYellow  = "\033[0;33m"
	colorCyan    = "\033[0;36m"
	colorMagenta = "\033[0;35m"
	colorDim     = "\033[2m"
)

func formatRelativeTime(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
}

func formatStatus(status string) string {
	switch status {
	case "succeeded":
		return "✓ succeeded"
	case "failed":
		return "✗ failed"
	default:
		return status
	}
}

func colorize(color, text string) string {
	return color + text + colorReset
}

func colorizeStatus(status string) string {
	switch status {
	case "succeeded":
		return colorize(colorGreen, "✓ succeeded")
	case "failed":
		return colorize(colorRed, "✗ failed")
	default:
		return colorize(colorYellow, status)
	}
}
