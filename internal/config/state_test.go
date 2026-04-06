package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSaveAndLoadState(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	os.MkdirAll(stateDir, 0755)
	path := filepath.Join(stateDir, "test.yaml")

	selected := []string{"project-a", "project-b", "project-c"}

	s := State{LastSelection: selected}
	data, err := yaml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Read it back
	readData, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var state State
	if err := yaml.Unmarshal(readData, &state); err != nil {
		t.Fatal(err)
	}

	if len(state.LastSelection) != 3 {
		t.Fatalf("expected 3 selections, got %d", len(state.LastSelection))
	}
	if state.LastSelection[0] != "project-a" {
		t.Errorf("first selection = %q, want 'project-a'", state.LastSelection[0])
	}
}

func TestLoadStateMissing(t *testing.T) {
	state := LoadState("nonexistent-profile-" + t.Name())
	if len(state.LastSelection) != 0 {
		t.Errorf("expected empty selections for missing state, got %d", len(state.LastSelection))
	}
}
