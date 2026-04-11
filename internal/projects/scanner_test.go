package projects

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanEmptyGlob(t *testing.T) {
	dir := t.TempDir()
	projs, err := Scan([]string{filepath.Join(dir, "*")}, "alphabetical")
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(projs) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projs))
	}
}

func TestScanGlobFindsProjects(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "alpha"), 0755)
	os.Mkdir(filepath.Join(dir, "beta"), 0755)
	os.Mkdir(filepath.Join(dir, ".hidden"), 0755)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("not a dir"), 0644)

	projs, err := Scan([]string{filepath.Join(dir, "*")}, "alphabetical")
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(projs) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projs))
	}
	if projs[0].Name != "alpha" {
		t.Errorf("first project = %q, want 'alpha'", projs[0].Name)
	}
	if projs[1].Name != "beta" {
		t.Errorf("second project = %q, want 'beta'", projs[1].Name)
	}
}

func TestScanLiteralDir(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "myproject")
	os.Mkdir(proj, 0755)

	projs, err := Scan([]string{proj}, "alphabetical")
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(projs) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projs))
	}
	if projs[0].Name != "myproject" {
		t.Errorf("project name = %q, want 'myproject'", projs[0].Name)
	}
}

func TestScanPrefixGlob(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "api-users"), 0755)
	os.Mkdir(filepath.Join(dir, "api-billing"), 0755)
	os.Mkdir(filepath.Join(dir, "webapp"), 0755)

	projs, err := Scan([]string{filepath.Join(dir, "api-*")}, "alphabetical")
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(projs) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projs))
	}
	if projs[0].Name != "api-billing" {
		t.Errorf("first project = %q, want 'api-billing'", projs[0].Name)
	}
}

func TestScanMultipleDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	os.Mkdir(filepath.Join(dir1, "shared"), 0755)
	os.Mkdir(filepath.Join(dir2, "shared"), 0755)
	os.Mkdir(filepath.Join(dir2, "unique"), 0755)

	projs, err := Scan([]string{
		filepath.Join(dir1, "*"),
		filepath.Join(dir2, "*"),
	}, "alphabetical")
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(projs) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(projs))
	}

	dupes := 0
	for _, p := range projs {
		if p.HasDuplicate {
			dupes++
		}
	}
	if dupes != 2 {
		t.Errorf("expected 2 duplicates, got %d", dupes)
	}
}

func TestScanSkipsNonexistentDirs(t *testing.T) {
	projs, err := Scan([]string{"/nonexistent/path"}, "alphabetical")
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(projs) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projs))
	}
}

func TestScanSkipsNonexistentGlob(t *testing.T) {
	projs, err := Scan([]string{"/nonexistent/path/*"}, "alphabetical")
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(projs) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projs))
	}
}

func TestExpandTilde(t *testing.T) {
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}
	tests := []struct {
		input    string
		expected string
	}{
		{"~/src", home + "/src"},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~user/path", "~user/path"}, // only bare ~ is expanded
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := expandTilde(tt.input)
			if got != tt.expected {
				t.Errorf("expandTilde(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestWindowName(t *testing.T) {
	tests := []struct {
		name         string
		project      Project
		expectedName string
	}{
		{
			name:         "no duplicate",
			project:      Project{Name: "myapp", Dir: "/src/github.com/user"},
			expectedName: "myapp",
		},
		{
			name:         "has duplicate",
			project:      Project{Name: "api", Dir: "/src/github.com/orgA", HasDuplicate: true},
			expectedName: "orgA/api",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.project.WindowName(); got != tt.expectedName {
				t.Errorf("WindowName() = %q, want %q", got, tt.expectedName)
			}
		})
	}
}

func TestSortProjects(t *testing.T) {
	now := time.Now()
	projs := []Project{
		{Name: "charlie", LastCommit: now.Add(-3 * time.Hour)},
		{Name: "alpha", LastCommit: now.Add(-1 * time.Hour)},
		{Name: "bravo", LastCommit: now.Add(-2 * time.Hour)},
	}

	SortProjects(projs, "alphabetical")
	if projs[0].Name != "alpha" || projs[1].Name != "bravo" || projs[2].Name != "charlie" {
		t.Errorf("alphabetical sort failed: %v", names(projs))
	}

	SortProjects(projs, "recent")
	if projs[0].Name != "alpha" || projs[1].Name != "bravo" || projs[2].Name != "charlie" {
		t.Errorf("recent sort failed: %v", names(projs))
	}
}

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		name     string
		t        time.Time
		expected string
	}{
		{"zero", time.Time{}, ""},
		{"now", time.Now(), "now"},
		{"minutes", time.Now().Add(-30 * time.Minute), "30m ago"},
		{"hours", time.Now().Add(-5 * time.Hour), "5h ago"},
		{"days", time.Now().Add(-3 * 24 * time.Hour), "3d ago"},
		{"months", time.Now().Add(-60 * 24 * time.Hour), "2mo ago"},
		{"years", time.Now().Add(-400 * 24 * time.Hour), "1y ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RelativeTime(tt.t)
			if got != tt.expected {
				t.Errorf("RelativeTime() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func names(projs []Project) []string {
	var n []string
	for _, p := range projs {
		n = append(n, p.Name)
	}
	return n
}
