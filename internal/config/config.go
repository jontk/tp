package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SourceDirs []string     `yaml:"source_dirs"`
	Defaults   []string     `yaml:"defaults"`
	Session    string       `yaml:"session"`
	Layout     LayoutConfig `yaml:"layout"`
}

type LayoutConfig struct {
	Panes []PaneConfig `yaml:"panes"`
}

type PaneConfig struct {
	Command  string `yaml:"command"`
	Percent  int    `yaml:"percent"`
	Position string `yaml:"position"`
	Active   bool   `yaml:"active,omitempty"`
}

func DefaultPath() string {
	return PathForProfile("")
}

func PathForProfile(profile string) string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	name := "config.yaml"
	if profile != "" {
		name = profile + ".yaml"
	}
	return filepath.Join(configDir, "tmux-projects", name)
}

func DefaultConfig() *Config {
	return &Config{
		SourceDirs: []string{
			filepath.Join(os.Getenv("HOME"), "src", "github.com"),
		},
		Defaults: []string{},
		Session:  "projects",
		Layout: LayoutConfig{
			Panes: []PaneConfig{
				{Command: "claude --continue", Percent: 40, Position: "left"},
				{Command: "vim .", Percent: 50, Position: "right-top"},
				{Command: "", Percent: 50, Position: "right-bottom", Active: true},
			},
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if writeErr := WriteDefault(path); writeErr != nil {
				return nil, writeErr
			}
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Session == "" {
		cfg.Session = "projects"
	}
	if len(cfg.Layout.Panes) == 0 {
		cfg.Layout = DefaultConfig().Layout
	}

	return cfg, nil
}

func WriteDefault(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	cfg := DefaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	header := "# tp - tmux project manager configuration\n#\n" +
		"# source_dirs: directories to scan for projects (each subdirectory = one project)\n" +
		"# defaults: projects pre-selected in the picker on launch\n" +
		"# session: tmux session name\n" +
		"# layout.panes: pane layout for each project window\n" +
		"#   position: left, right-top, right-bottom\n" +
		"#   command: command to run in the pane (empty = shell)\n" +
		"#   percent: split percentage\n" +
		"#   active: which pane gets focus\n\n"

	return os.WriteFile(path, []byte(header+string(data)), 0644)
}
