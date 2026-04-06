package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SourceDirs []string                 `yaml:"source_dirs"`
	Defaults   []string                 `yaml:"defaults"`
	Session    string                   `yaml:"session"`
	Sort       string                   `yaml:"sort,omitempty"`
	Layout     []PaneConfig             `yaml:"layout"`
	Projects   map[string]ProjectConfig `yaml:"projects,omitempty"`
}

type ProjectConfig struct {
	Layout []PaneConfig `yaml:"layout"`
}

type PaneConfig struct {
	Command string `yaml:"command"`
	Split   string `yaml:"split,omitempty"`
	Percent int    `yaml:"percent,omitempty"`
	Active  bool   `yaml:"active,omitempty"`
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

func DefaultLayout() []PaneConfig {
	return []PaneConfig{
		{Command: "claude --continue || claude"},
		{Split: "horizontal", Percent: 60, Command: "vim ."},
		{Split: "vertical", Percent: 50, Command: "", Active: true},
	}
}

func DefaultConfig() *Config {
	return &Config{
		SourceDirs: []string{
			filepath.Join(os.Getenv("HOME"), "src", "github.com"),
		},
		Defaults: []string{},
		Session:  "projects",
		Layout:   DefaultLayout(),
	}
}

func (c *Config) LayoutForProject(name string) []PaneConfig {
	if p, ok := c.Projects[name]; ok && len(p.Layout) > 0 {
		return p.Layout
	}
	return c.Layout
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
	if cfg.Sort == "" {
		cfg.Sort = "recent"
	}
	if len(cfg.Layout) == 0 {
		cfg.Layout = DefaultLayout()
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
		"# layout: pane layout as sequential tmux splits\n" +
		"#   split: horizontal (side-by-side) or vertical (top/bottom)\n" +
		"#   percent: size of the new pane as a percentage\n" +
		"#   command: command to run (empty = shell)\n" +
		"#   active: which pane gets focus\n\n"

	return os.WriteFile(path, []byte(header+string(data)), 0644)
}
