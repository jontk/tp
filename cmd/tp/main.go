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
	case "kill":
		runKill()
	case "switch":
		runSwitch()
	case "validate":
		runValidate()
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

	// When creating a new session, use saved selections (falling back to defaults)
	defaults := cfg.Defaults
	if !sessionExists {
		state := config.LoadState(profile)
		if len(state.LastSelection) > 0 {
			defaults = state.LastSelection
		}
	} else {
		defaults = nil
	}

	selected, closed, confirmed := runPicker(projs, defaults, openWindows, cfg.Sort, cfg.ShowPreview(), cfg)
	if !confirmed {
		return
	}

	closeWindows(cfg, closed)

	// Save current selection state (open + newly selected - closed)
	var allOpen []string
	for w := range openWindows {
		allOpen = append(allOpen, w)
	}
	for _, p := range selected {
		allOpen = append(allOpen, p.WindowName())
	}
	closedNames := make(map[string]bool)
	for _, p := range closed {
		closedNames[p.WindowName()] = true
	}
	var finalSelection []string
	for _, name := range allOpen {
		if !closedNames[name] {
			finalSelection = append(finalSelection, name)
		}
	}
	config.SaveState(profile, finalSelection)

	if len(selected) > 0 {
		createWindows(cfg, selected, sessionExists)
	} else if !sessionExists {
		return
	}

	if tmux.InsideTmux() {
		// If we just created a new session from inside tmux, switch to it
		if !sessionExists {
			if err := tmux.SwitchClient(cfg.Session); err != nil {
				fmt.Fprintf(os.Stderr, "failed to switch to session: %v\n", err)
			}
		}
	} else {
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

func runSwitch() {
	profiles, err := config.ListProfiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to list profiles: %v\n", err)
		os.Exit(1)
	}

	if len(profiles) == 0 {
		fmt.Fprintln(os.Stderr, "no profiles found")
		os.Exit(1)
	}

	// Build list with status
	type profileEntry struct {
		profile config.Profile
		active  bool
	}
	var entries []profileEntry
	for _, p := range profiles {
		entries = append(entries, profileEntry{
			profile: p,
			active:  tmux.SessionExists(p.Session),
		})
	}

	// If only one profile, nothing to switch to
	if len(entries) == 1 {
		fmt.Println("only one profile configured")
		return
	}

	// Print numbered list
	fmt.Println("profiles:")
	for i, e := range entries {
		status := "  "
		if e.active {
			status = "* "
		}
		fmt.Printf("  %s%d) %s (session: %s)\n", status, i+1, e.profile.DisplayName(), e.profile.Session)
	}
	fmt.Print("\nswitch to [1-" + fmt.Sprintf("%d", len(entries)) + "]: ")

	// Read selection
	var choice int
	if _, err := fmt.Scanf("%d", &choice); err != nil || choice < 1 || choice > len(entries) {
		fmt.Fprintln(os.Stderr, "cancelled")
		return
	}

	selected := entries[choice-1]

	if !selected.active {
		// Launch the profile's picker to create the session
		profile = selected.profile.Name
		runDefault()
		return
	}

	// Switch to the existing session
	if tmux.InsideTmux() {
		if err := tmux.SwitchClient(selected.profile.Session); err != nil {
			fmt.Fprintf(os.Stderr, "failed to switch: %v\n", err)
			os.Exit(1)
		}
	} else {
		cc := tmux.IsITerm()
		for _, arg := range os.Args[1:] {
			if arg == "--cc" {
				cc = true
			}
		}
		if err := tmux.Attach(selected.profile.Session, cc); err != nil {
			fmt.Fprintf(os.Stderr, "failed to attach: %v\n", err)
			os.Exit(1)
		}
	}
}

func runKill() {
	cfg := loadConfig()

	if !tmux.SessionExists(cfg.Session) {
		fmt.Printf("session %q does not exist\n", cfg.Session)
		return
	}

	// Find another active session to switch to before killing
	var fallback string
	if tmux.InsideTmux() {
		profiles, err := config.ListProfiles()
		if err == nil {
			for _, p := range profiles {
				if p.Session != cfg.Session && tmux.SessionExists(p.Session) {
					fallback = p.Session
					break
				}
			}
		}

		if fallback != "" {
			// Switch first, then kill — otherwise we get detached
			tmux.SwitchClient(fallback)
		}
	}

	if err := tmux.KillSession(cfg.Session); err != nil {
		fmt.Fprintf(os.Stderr, "failed to kill session %q: %v\n", cfg.Session, err)
		os.Exit(1)
	}

	if fallback != "" {
		fmt.Printf("killed session %q, switched to %q\n", cfg.Session, fallback)
	} else {
		fmt.Printf("killed session %q\n", cfg.Session)
	}
}

func runValidate() {
	path := configPath()

	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	errors := config.Validate(cfg)
	if len(errors) == 0 {
		fmt.Printf("%s: valid\n", path)
		return
	}

	fmt.Fprintf(os.Stderr, "%s: %d issue(s)\n", path, len(errors))
	for _, e := range errors {
		fmt.Fprintf(os.Stderr, "  - %s\n", e)
	}
	os.Exit(1)
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
  tp kill         Kill the current session
  tp switch       Switch between profile sessions
  tp validate     Validate config file
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
	projs, err := projects.Scan(cfg.SourceDirs, cfg.Sort)
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

func runPicker(projs []projects.Project, defaults []string, openWindows map[string]bool, sortMode string, showPreview bool, cfg *config.Config) (selected []projects.Project, closed []projects.Project, confirmed bool) {
	picker := tui.NewPicker(projs, defaults, openWindows, sortMode, showPreview, cfg)
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
		wname := proj.WindowName()
		idx, err := tmux.WindowIndex(cfg.Session, wname)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to find window %s: %v\n", wname, err)
			continue
		}
		target := fmt.Sprintf("%s:%s", cfg.Session, idx)
		if err := tmux.KillWindow(target); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close window %s: %v\n", wname, err)
		}
	}
}

func createWindows(cfg *config.Config, selected []projects.Project, sessionExists bool) {
	for i, proj := range selected {
		layout := cfg.LayoutForProject(proj.Name)
		wname := proj.WindowName()

		if i == 0 && !sessionExists {
			if err := tmux.NewSession(cfg.Session, wname, proj.Path); err != nil {
				fmt.Fprintf(os.Stderr, "failed to create session: %v\n", err)
				os.Exit(1)
			}
			tmux.SetEnvironment(cfg.Session, "TP_PROFILE", profile)
			if err := tmux.SetupProjectWindow(cfg.Session, wname, proj.Path, layout); err != nil {
				fmt.Fprintf(os.Stderr, "failed to setup window %s: %v\n", wname, err)
			}
			continue
		}

		if err := tmux.NewWindow(cfg.Session, wname, proj.Path); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create window %s: %v\n", wname, err)
			continue
		}
		if err := tmux.SetupProjectWindow(cfg.Session, wname, proj.Path, layout); err != nil {
			fmt.Fprintf(os.Stderr, "failed to setup window %s: %v\n", wname, err)
		}
	}

	// Select first window
	if !sessionExists && len(selected) > 0 {
		if idx, err := tmux.WindowIndex(cfg.Session, selected[0].WindowName()); err == nil {
			tmux.SelectWindow(fmt.Sprintf("%s:%s", cfg.Session, idx))
		}
	}
}
