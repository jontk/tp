package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SourceDirs    []string                 `yaml:"source_dirs"`
	Defaults      []string                 `yaml:"defaults"`
	Session       string                   `yaml:"session"`
	Sort          string                   `yaml:"sort,omitempty"`
	Preview       *bool                    `yaml:"preview,omitempty"`
	Layout        []PaneConfig             `yaml:"layout"`
	LayoutPresets map[string][]PaneConfig  `yaml:"layout_presets,omitempty"`
	Projects      map[string]ProjectConfig `yaml:"projects,omitempty"`
}

func (c *Config) ShowPreview() bool {
	if c.Preview == nil {
		return true // enabled by default
	}
	return *c.Preview
}

type ProjectConfig struct {
	Preset string       `yaml:"preset,omitempty"`
	Layout []PaneConfig `yaml:"layout,omitempty"`
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
	return filepath.Join(configDir, "ratatosk", name)
}

// ListProfiles returns all available profiles by scanning the config dir.
// The default profile is returned as "" with session name from config.yaml.
func ListProfiles() ([]Profile, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	dir := filepath.Join(configDir, "ratatosk")

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
		{Command: "vim ."},
		{Split: "horizontal", Percent: 40, Command: "", Active: true},
		{Split: "vertical", Percent: 50, Command: "lazygit"},
	}
}

func DefaultConfig() *Config {
	return &Config{
		SourceDirs: []string{
			filepath.Join(os.Getenv("HOME"), "src", "github.com", "*"),
		},
		Defaults: []string{},
		Session:  "projects",
		Layout:   DefaultLayout(),
	}
}

func (c *Config) LayoutForProject(name string) []PaneConfig {
	if p, ok := c.Projects[name]; ok {
		// Inline layout takes priority
		if len(p.Layout) > 0 {
			return p.Layout
		}
		// Then try preset
		if p.Preset != "" {
			if preset, ok := c.LayoutPresets[p.Preset]; ok {
				return preset
			}
		}
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

	header := "# tosk - tmux project manager configuration\n#\n" +
		"# source_dirs: paths or glob patterns that resolve to project directories\n" +
		"# defaults: projects pre-selected in the picker on launch\n" +
		"# session: tmux session name\n" +
		"# layout: pane layout as sequential tmux splits\n" +
		"#   split: horizontal (side-by-side) or vertical (top/bottom)\n" +
		"#   percent: size of the new pane as a percentage\n" +
		"#   command: command to run (empty = shell)\n" +
		"#   active: which pane gets focus\n\n"

	return os.WriteFile(path, []byte(header+string(data)), 0644)
}

func expandTilde(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home := os.Getenv("HOME")
		if home != "" {
			return home + path[1:]
		}
	}
	return path
}

func Validate(cfg *Config) []string {
	var errs []string

	if len(cfg.SourceDirs) == 0 {
		errs = append(errs, "source_dirs is empty")
	}
	for _, dir := range cfg.SourceDirs {
		expanded := expandTilde(dir)
		// For glob patterns, validate the parent directory exists.
		if strings.ContainsAny(expanded, "*?[") {
			parent := filepath.Dir(expanded)
			if _, err := os.Stat(parent); os.IsNotExist(err) {
				errs = append(errs, fmt.Sprintf("source_dirs: parent %q does not exist (from %q)", parent, dir))
			}
		} else {
			if _, err := os.Stat(expanded); os.IsNotExist(err) {
				errs = append(errs, fmt.Sprintf("source_dirs: %q does not exist", dir))
			}
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

	// Validate layout presets
	for name, preset := range cfg.LayoutPresets {
		if len(preset) == 0 {
			errs = append(errs, fmt.Sprintf("layout_presets.%s: layout is empty", name))
			continue
		}
		for i, pane := range preset {
			if i > 0 && pane.Split != "horizontal" && pane.Split != "vertical" {
				errs = append(errs, fmt.Sprintf("layout_presets.%s[%d]: split %q is not valid", name, i, pane.Split))
			}
		}
	}

	// Validate per-project configs
	for name, proj := range cfg.Projects {
		if proj.Preset != "" {
			if _, ok := cfg.LayoutPresets[proj.Preset]; !ok {
				errs = append(errs, fmt.Sprintf("projects.%s: preset %q not found in layout_presets", name, proj.Preset))
			}
		}
		if len(proj.Layout) == 0 && proj.Preset == "" {
			continue // will use global default
		}
		for i, pane := range proj.Layout {
			if i > 0 && pane.Split != "horizontal" && pane.Split != "vertical" {
				errs = append(errs, fmt.Sprintf("projects.%s.layout[%d]: split %q is not valid", name, i, pane.Split))
			}
		}
	}

	return errs
}
