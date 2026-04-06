package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SourceDirs []string                 `yaml:"source_dirs"`
	Defaults   []string                 `yaml:"defaults"`
	Session    string                   `yaml:"session"`
	Sort       string                   `yaml:"sort,omitempty"`
	Preview    *bool                    `yaml:"preview,omitempty"`
	Layout     []PaneConfig             `yaml:"layout"`
	Projects   map[string]ProjectConfig `yaml:"projects,omitempty"`
}

func (c *Config) ShowPreview() bool {
	if c.Preview == nil {
		return true // enabled by default
	}
	return *c.Preview
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

// ListProfiles returns all available profiles by scanning the config dir.
// The default profile is returned as "" with session name from config.yaml.
func ListProfiles() ([]Profile, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	dir := filepath.Join(configDir, "tmux-projects")

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var profiles []Profile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		profileName := name
		if name == "config" {
			profileName = ""
		}

		cfg, err := Load(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		profiles = append(profiles, Profile{
			Name:    profileName,
			Session: cfg.Session,
		})
	}

	return profiles, nil
}

type Profile struct {
	Name    string // "" for default
	Session string
}

func (p Profile) DisplayName() string {
	if p.Name == "" {
		return "default"
	}
	return p.Name
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

func Validate(cfg *Config) []string {
	var errs []string

	if len(cfg.SourceDirs) == 0 {
		errs = append(errs, "source_dirs is empty")
	}
	for _, dir := range cfg.SourceDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("source_dirs: %q does not exist", dir))
		}
	}

	if cfg.Session == "" {
		errs = append(errs, "session name is empty")
	}

	if cfg.Sort != "" && cfg.Sort != "recent" && cfg.Sort != "alphabetical" {
		errs = append(errs, fmt.Sprintf("sort: %q is not valid (use 'recent' or 'alphabetical')", cfg.Sort))
	}

	if len(cfg.Layout) == 0 {
		errs = append(errs, "layout is empty")
	}
	for i, pane := range cfg.Layout {
		if i > 0 && pane.Split == "" {
			errs = append(errs, fmt.Sprintf("layout[%d]: missing 'split' (horizontal or vertical)", i))
		}
		if i > 0 && pane.Split != "horizontal" && pane.Split != "vertical" {
			errs = append(errs, fmt.Sprintf("layout[%d]: split %q is not valid (use 'horizontal' or 'vertical')", i, pane.Split))
		}
		if pane.Percent < 0 || pane.Percent > 100 {
			errs = append(errs, fmt.Sprintf("layout[%d]: percent %d is out of range (0-100)", i, pane.Percent))
		}
	}

	for name, proj := range cfg.Projects {
		if len(proj.Layout) == 0 {
			errs = append(errs, fmt.Sprintf("projects.%s: layout is empty", name))
			continue
		}
		for i, pane := range proj.Layout {
			if i > 0 && pane.Split != "horizontal" && pane.Split != "vertical" {
				errs = append(errs, fmt.Sprintf("projects.%s.layout[%d]: split %q is not valid", name, i, pane.Split))
			}
		}
	}

	return errs
}
