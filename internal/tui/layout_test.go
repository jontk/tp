package tui

import (
	"strings"
	"testing"

	"github.com/jontk/ratatosk/internal/config"
)

func TestRenderLayoutDefault(t *testing.T) {
	layout := []config.PaneConfig{
		{Command: "vim ."},
		{Split: "horizontal", Percent: 40, Command: "", Active: true},
		{Split: "vertical", Percent: 50, Command: "lazygit"},
	}

	result := RenderLayout(layout, 40, 7)

	if result == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(result, "vim") {
		t.Error("expected 'vim' in output")
	}
	if !strings.Contains(result, "shell") {
		t.Error("expected 'shell' in output")
	}
	if !strings.Contains(result, "lazygit") {
		t.Error("expected 'lazygit' in output")
	}
	if !strings.Contains(result, "*") {
		t.Error("expected active pane marker '*' in output")
	}
	// Check box drawing characters
	if !strings.Contains(result, "┌") || !strings.Contains(result, "┘") {
		t.Error("expected box drawing corners")
	}
}

func TestRenderLayoutWide(t *testing.T) {
	layout := []config.PaneConfig{
		{Command: "lazygit"},
		{Split: "horizontal", Percent: 66, Command: "vim ."},
		{Split: "horizontal", Percent: 50, Command: ""},
		{Split: "vertical", Percent: 50, Command: "", Active: true},
	}

	result := RenderLayout(layout, 40, 7)

	// Should have 3 columns
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	for _, line := range lines {
		// Count vertical bars (should be at least 4 for 3 columns: left, 2 dividers, right)
		vbars := strings.Count(line, "│") + strings.Count(line, "├") + strings.Count(line, "┤") + strings.Count(line, "┼")
		if vbars < 2 {
			// Allow top/bottom border lines
			if !strings.Contains(line, "─") {
				t.Errorf("expected at least 2 vertical elements, got %d in %q", vbars, line)
			}
		}
	}
}

func TestRenderLayoutEmpty(t *testing.T) {
	result := RenderLayout(nil, 40, 7)
	if result != "" {
		t.Error("expected empty output for nil layout")
	}
}

func TestRenderLayoutSinglePane(t *testing.T) {
	layout := []config.PaneConfig{
		{Command: "vim ."},
	}
	result := RenderLayout(layout, 20, 5)
	if result == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(result, "vim") {
		t.Error("expected 'vim' in output")
	}
}

func TestRenderLayoutTooSmall(t *testing.T) {
	layout := []config.PaneConfig{
		{Command: "vim ."},
		{Split: "horizontal", Percent: 50, Command: "shell"},
	}
	result := RenderLayout(layout, 3, 2)
	if result != "" {
		t.Error("expected empty output for tiny dimensions")
	}
}

func TestShortCmd(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "shell"},
		{"vim .", "vim ."},
		{"lazygit", "lazygit"},
		{"npm run dev", "npm run dev"},
		{"a-very-long-command-name", "a-very-long-"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := shortCmd(tt.input)
			if got != tt.expected {
				t.Errorf("shortCmd(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
