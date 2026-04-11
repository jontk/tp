package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Session != "projects" {
		t.Errorf("expected session 'projects', got %q", cfg.Session)
	}
	if len(cfg.Layout) != 3 {
		t.Errorf("expected 3 default layout panes, got %d", len(cfg.Layout))
	}
	if len(cfg.SourceDirs) != 1 {
		t.Errorf("expected 1 source dir, got %d", len(cfg.SourceDirs))
	}
}

func TestShowPreview(t *testing.T) {
	tests := []struct {
		name     string
		preview  *bool
		expected bool
	}{
		{"nil defaults to true", nil, true},
		{"explicit true", boolPtr(true), true},
		{"explicit false", boolPtr(false), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Preview: tt.preview}
			if got := cfg.ShowPreview(); got != tt.expected {
				t.Errorf("ShowPreview() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLayoutForProject(t *testing.T) {
	preset := []PaneConfig{
		{Command: "claude"},
		{Split: "horizontal", Percent: 50, Command: "vim ."},
	}
	inline := []PaneConfig{
		{Command: "custom"},
	}
	global := []PaneConfig{
		{Command: "default"},
	}

	tests := []struct {
		name     string
		cfg      *Config
		project  string
		expected string // command of first pane
	}{
		{
			name: "uses global layout",
			cfg: &Config{
				Layout: global,
			},
			project:  "unknown",
			expected: "default",
		},
		{
			name: "uses inline layout",
			cfg: &Config{
				Layout: global,
				Projects: map[string]ProjectConfig{
					"myproject": {Layout: inline},
				},
			},
			project:  "myproject",
			expected: "custom",
		},
		{
			name: "uses preset",
			cfg: &Config{
				Layout:        global,
				LayoutPresets: map[string][]PaneConfig{"frontend": preset},
				Projects: map[string]ProjectConfig{
					"webapp": {Preset: "frontend"},
				},
			},
			project:  "webapp",
			expected: "claude",
		},
		{
			name: "inline takes priority over preset",
			cfg: &Config{
				Layout:        global,
				LayoutPresets: map[string][]PaneConfig{"frontend": preset},
				Projects: map[string]ProjectConfig{
					"special": {Preset: "frontend", Layout: inline},
				},
			},
			project:  "special",
			expected: "custom",
		},
		{
			name: "missing preset falls back to global",
			cfg: &Config{
				Layout: global,
				Projects: map[string]ProjectConfig{
					"broken": {Preset: "nonexistent"},
				},
			},
			project:  "broken",
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := tt.cfg.LayoutForProject(tt.project)
			if len(layout) == 0 {
				t.Fatal("got empty layout")
			}
			if layout[0].Command != tt.expected {
				t.Errorf("first pane command = %q, want %q", layout[0].Command, tt.expected)
			}
		})
	}
}

func TestLoadAndWriteDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Should have created the file
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected config file to be created")
	}

	if cfg.Session != "projects" {
		t.Errorf("expected session 'projects', got %q", cfg.Session)
	}
}

func TestLoadCustomConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `
source_dirs:
    - /tmp/test
session: custom
sort: alphabetical
layout:
    - command: vim
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Session != "custom" {
		t.Errorf("session = %q, want 'custom'", cfg.Session)
	}
	if cfg.Sort != "alphabetical" {
		t.Errorf("sort = %q, want 'alphabetical'", cfg.Sort)
	}
	if len(cfg.Layout) != 1 {
		t.Errorf("layout panes = %d, want 1", len(cfg.Layout))
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		wantErrs  int
		wantMatch string // substring that should appear in errors
	}{
		{
			name: "valid config",
			cfg: &Config{
				SourceDirs: []string{os.TempDir() + "/*"},
				Session:    "projects",
				Layout:     DefaultLayout(),
			},
			wantErrs: 0,
		},
		{
			name: "empty source dirs",
			cfg: &Config{
				Session: "test",
				Layout:  DefaultLayout(),
			},
			wantErrs:  1,
			wantMatch: "source_dirs is empty",
		},
		{
			name: "invalid sort",
			cfg: &Config{
				SourceDirs: []string{os.TempDir()},
				Session:    "test",
				Sort:       "invalid",
				Layout:     DefaultLayout(),
			},
			wantErrs:  1,
			wantMatch: "not valid",
		},
		{
			name: "empty layout",
			cfg: &Config{
				SourceDirs: []string{os.TempDir()},
				Session:    "test",
			},
			wantErrs:  1,
			wantMatch: "layout is empty",
		},
		{
			name: "invalid split in layout",
			cfg: &Config{
				SourceDirs: []string{os.TempDir()},
				Session:    "test",
				Layout: []PaneConfig{
					{Command: "vim"},
					{Split: "diagonal", Percent: 50},
				},
			},
			wantErrs:  1,
			wantMatch: "not valid",
		},
		{
			name: "missing preset reference",
			cfg: &Config{
				SourceDirs: []string{os.TempDir()},
				Session:    "test",
				Layout:     DefaultLayout(),
				Projects: map[string]ProjectConfig{
					"broken": {Preset: "nonexistent"},
				},
			},
			wantErrs:  1,
			wantMatch: "not found in layout_presets",
		},
		{
			name: "valid preset reference",
			cfg: &Config{
				SourceDirs:    []string{os.TempDir()},
				Session:       "test",
				Layout:        DefaultLayout(),
				LayoutPresets: map[string][]PaneConfig{"web": DefaultLayout()},
				Projects: map[string]ProjectConfig{
					"app": {Preset: "web"},
				},
			},
			wantErrs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.cfg)
			if len(errs) != tt.wantErrs {
				t.Errorf("got %d errors, want %d: %v", len(errs), tt.wantErrs, errs)
			}
			if tt.wantMatch != "" && len(errs) > 0 {
				found := false
				for _, e := range errs {
					if contains(e, tt.wantMatch) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got %v", tt.wantMatch, errs)
				}
			}
		})
	}
}

func TestPathForProfile(t *testing.T) {
	tests := []struct {
		profile  string
		wantFile string
	}{
		{"", "config.yaml"},
		{"work", "work.yaml"},
		{"brokkr", "brokkr.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			path := PathForProfile(tt.profile)
			if filepath.Base(path) != tt.wantFile {
				t.Errorf("PathForProfile(%q) = %q, want file %q", tt.profile, path, tt.wantFile)
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
