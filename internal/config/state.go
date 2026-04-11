package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type State struct {
	LastSelection []string `yaml:"last_selection"`
}

func statePath(profile string) string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	name := "default.yaml"
	if profile != "" {
		name = profile + ".yaml"
	}
	return filepath.Join(configDir, "ratatosk", "state", name)
}

func LoadState(profile string) *State {
	path := statePath(profile)
	data, err := os.ReadFile(path)
	if err != nil {
		return &State{}
	}
	var s State
	if err := yaml.Unmarshal(data, &s); err != nil {
		return &State{}
	}
	return &s
}

func SaveState(profile string, selected []string) error {
	path := statePath(profile)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	s := State{LastSelection: selected}
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
