package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	// Gentle green for success
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9ece6a"))

	// Subtle gold for warnings
	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e0af68")).
			Bold(true)

	// Muted red for errors
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f7768e")).
			Bold(true)
)

func init() {
	// Check for color support and set appropriate profile
	setupColorProfile()
}

// setupColorProfile detects terminal capabilities and sets appropriate color profile
func setupColorProfile() {
	// Check for explicit no-color requests
	if os.Getenv("NO_COLOR") != "" {
		lipgloss.SetColorProfile(termenv.Ascii)
		return
	}

	// Check for dumb terminal
	term := strings.ToLower(os.Getenv("TERM"))
	if term == "dumb" || term == "" {
		lipgloss.SetColorProfile(termenv.Ascii)
		return
	}

	// Check for basic terminal types that might not support colors well
	basicTerms := []string{"linux", "console", "vt100", "vt102", "vt220"}
	for _, basicTerm := range basicTerms {
		if strings.Contains(term, basicTerm) {
			lipgloss.SetColorProfile(termenv.ANSI)
			return
		}
	}

	// For CI environments, use basic colors
	if os.Getenv("CI") != "" {
		lipgloss.SetColorProfile(termenv.ANSI)
		return
	}

	// Default: let lipgloss auto-detect
	// It will choose the best profile based on terminal capabilities
}

// safeRender safely renders text with styling, falling back to plain text on error
func safeRender(style lipgloss.Style, text string) string {
	defer func() {
		if r := recover(); r != nil {
			// If styling fails, just return the plain text
			fmt.Fprintf(os.Stderr, "Warning: styling failed, using plain text\n")
		}
	}()

	// Try to render with style
	return style.Render(text)
}

// Maps container status to soft-styled output
func statusColor(status string) lipgloss.Style {
	switch status {
	case "running":
		return successStyle
	case "stopped":
		return warningStyle
	default:
		return errorStyle
	}
}

// Helper functions for safe rendering of common styles
func renderSuccess(text string) string {
	return safeRender(successStyle, text)
}

func renderError(text string) string {
	return safeRender(errorStyle, text)
}
