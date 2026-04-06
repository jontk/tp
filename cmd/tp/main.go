package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jontk/tp/internal/config"
	"github.com/jontk/tp/internal/projects"
	"github.com/jontk/tp/internal/tmux"
	"github.com/jontk/tp/internal/tui"
)

var profile string

func main() {
	// Parse global flags
	args := parseGlobalFlags(os.Args[1:])

	// Auto-detect profile from current tmux session if not specified
	if profile == "" && tmux.InsideTmux() {
		if sess := currentSessionName(); sess != "" {
			if p := tmux.GetEnvironment(sess, "TP_PROFILE"); p != "" {
				profile = p
			}
		}
	}

	cmd := ""
	if len(args) > 0 && args[0] != "" && args[0][0] != '-' {
		cmd = args[0]
	}

	switch cmd {
	case "", "add":
		runDefault()
	case "list":
		runList()
	case "config":
		runConfig()
	case "help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func parseGlobalFlags(args []string) []string {
	var remaining []string
	for i := 0; i < len(args); i++ {
		if args[i] == "-p" && i+1 < len(args) {
			profile = args[i+1]
			i++
		} else {
			remaining = append(remaining, args[i])
		}
	}
	return remaining
}

func runDefault() {
	cfg := loadConfig()
	projs := scanProjects(cfg)

	openWindows := make(map[string]bool)
	sessionExists := tmux.SessionExists(cfg.Session)
	if sessionExists {
		windows, err := tmux.ListWindows(cfg.Session)
		if err == nil {
			for _, w := range windows {
				openWindows[w] = true
			}
		}
	}

	// Only pre-select defaults when creating a new session
	defaults := cfg.Defaults
	if sessionExists {
		defaults = nil
	}

	selected, closed, confirmed := runPicker(projs, defaults, openWindows)
	if !confirmed {
		return
	}

	closeWindows(cfg, closed)

	if len(selected) > 0 {
		createWindows(cfg, selected, sessionExists)
	} else if !sessionExists {
		return
	}

	// Attach if not already inside tmux
	if !tmux.InsideTmux() {
		cc := tmux.IsITerm()
		for _, arg := range os.Args[1:] {
			if arg == "--cc" {
				cc = true
			}
		}
		if err := tmux.Attach(cfg.Session, cc); err != nil {
			fmt.Fprintf(os.Stderr, "failed to attach: %v\n", err)
			os.Exit(1)
		}
	}
}

func runList() {
	cfg := loadConfig()

	if !tmux.SessionExists(cfg.Session) {
		fmt.Println("no active session")
		return
	}

	windows, err := tmux.ListWindows(cfg.Session)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to list windows: %v\n", err)
		os.Exit(1)
	}

	for _, w := range windows {
		fmt.Println(w)
	}
}

func runConfig() {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	path := configPath()

	// Ensure config exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := config.WriteDefault(path); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create config: %v\n", err)
			os.Exit(1)
		}
	}

	editorPath, err := exec.LookPath(editor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "editor %q not found: %v\n", editor, err)
		os.Exit(1)
	}

	if err := syscall.Exec(editorPath, []string{editor, path}, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "failed to open editor: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Print(`tp — tmux project manager

Usage:
  tp              Open picker — creates session or manages existing one
  tp list         List current session windows
  tp config       Open config in $EDITOR
  tp help         Show this help

Flags:
  -p <profile>    Use a named profile (e.g. tp -p work uses work.yaml)
  --cc            Force iTerm2 control mode (-CC) on attach

Config: ~/.config/tmux-projects/config.yaml
       ~/.config/tmux-projects/<profile>.yaml
`)
}

func currentSessionName() string {
	out, err := exec.Command("tmux", "display-message", "-p", "#{session_name}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func configPath() string {
	return config.PathForProfile(profile)
}

func loadConfig() *config.Config {
	cfg, err := config.Load(configPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

func scanProjects(cfg *config.Config) []projects.Project {
	projs, err := projects.Scan(cfg.SourceDirs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to scan projects: %v\n", err)
		os.Exit(1)
	}
	if len(projs) == 0 {
		fmt.Fprintln(os.Stderr, "no projects found in configured source_dirs")
		os.Exit(1)
	}
	return projs
}

func runPicker(projs []projects.Project, defaults []string, openWindows map[string]bool) (selected []projects.Project, closed []projects.Project, confirmed bool) {
	picker := tui.NewPicker(projs, defaults, openWindows)
	p := tea.NewProgram(picker, tea.WithAltScreen())

	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "picker error: %v\n", err)
		os.Exit(1)
	}

	m := result.(tui.Model)
	return m.Selected(), m.Closed(), m.Confirmed()
}

func closeWindows(cfg *config.Config, closed []projects.Project) {
	for _, proj := range closed {
		target := fmt.Sprintf("%s:%s", cfg.Session, proj.Name)
		if err := tmux.KillWindow(target); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close window %s: %v\n", proj.Name, err)
		}
	}
}

func createWindows(cfg *config.Config, selected []projects.Project, sessionExists bool) {
	for i, proj := range selected {
		if i == 0 && !sessionExists {
			if err := tmux.NewSession(cfg.Session, proj.Name, proj.Path); err != nil {
				fmt.Fprintf(os.Stderr, "failed to create session: %v\n", err)
				os.Exit(1)
			}
			tmux.SetEnvironment(cfg.Session, "TP_PROFILE", profile)
			if err := tmux.SetupProjectWindow(cfg.Session, proj.Name, proj.Path, cfg.Layout); err != nil {
				fmt.Fprintf(os.Stderr, "failed to setup window %s: %v\n", proj.Name, err)
			}
			continue
		}

		if err := tmux.NewWindow(cfg.Session, proj.Name, proj.Path); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create window %s: %v\n", proj.Name, err)
			continue
		}
		if err := tmux.SetupProjectWindow(cfg.Session, proj.Name, proj.Path, cfg.Layout); err != nil {
			fmt.Fprintf(os.Stderr, "failed to setup window %s: %v\n", proj.Name, err)
		}
	}

	// Select first window
	if !sessionExists && len(selected) > 0 {
		tmux.SelectWindow(fmt.Sprintf("%s:%s", cfg.Session, selected[0].Name))
	}
}
