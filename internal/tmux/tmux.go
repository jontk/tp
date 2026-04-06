package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/jontk/tp/internal/config"
)

func SessionExists(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	return cmd.Run() == nil
}

func NewSession(name, window, dir string) error {
	return run("new-session", "-d", "-s", name, "-n", window, "-c", dir)
}

func SetEnvironment(session, key, value string) error {
	return run("set-environment", "-t", session, key, value)
}

func GetEnvironment(session, key string) string {
	cmd := exec.Command("tmux", "show-environment", "-t", session, key)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	// Output format: KEY=value
	parts := strings.SplitN(strings.TrimSpace(string(out)), "=", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

func NewWindow(session, name, dir string) error {
	return run("new-window", "-t", session, "-n", name, "-c", dir)
}

func SetWindowOption(target, option, value string) error {
	return run("set-window-option", "-t", target, option, value)
}

func SplitWindow(target, dir string, horizontal bool, percent int) error {
	flag := "-v"
	if horizontal {
		flag = "-h"
	}
	return run("split-window", "-t", target, flag, "-p", fmt.Sprintf("%d", percent), "-c", dir)
}

func SendKeys(target, keys string) error {
	return run("send-keys", "-t", target, keys, "C-m")
}

func SelectPane(target string) error {
	return run("select-pane", "-t", target)
}

func SelectWindow(target string) error {
	return run("select-window", "-t", target)
}

func KillWindow(target string) error {
	return run("kill-window", "-t", target)
}

func KillSession(name string) error {
	return run("kill-session", "-t", name)
}

func WindowIndex(session, name string) (string, error) {
	cmd := exec.Command("tmux", "list-windows", "-t", session, "-F", "#{window_index}:#{window_name}")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && parts[1] == name {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("window %q not found", name)
}

func ListWindows(session string) ([]string, error) {
	cmd := exec.Command("tmux", "list-windows", "-t", session, "-F", "#{window_name}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var windows []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			windows = append(windows, line)
		}
	}
	return windows, nil
}

func Attach(session string, cc bool) error {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return err
	}

	args := []string{"tmux"}
	if cc {
		args = append(args, "-CC")
	}
	args = append(args, "attach-session", "-t", session)

	return syscall.Exec(tmuxPath, args, os.Environ())
}

func InsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

func IsITerm() bool {
	return os.Getenv("TERM_PROGRAM") == "iTerm.app" || os.Getenv("LC_TERMINAL") == "iTerm2"
}

func SetupProjectWindow(session, name, dir string, layout []config.PaneConfig) error {
	// Look up window by index to avoid tmux interpreting dots in window
	// names as pane delimiters (e.g. "jontk.com")
	idx, err := WindowIndex(session, name)
	if err != nil {
		return fmt.Errorf("find window %s: %w", name, err)
	}
	target := fmt.Sprintf("%s:%s", session, idx)

	SetWindowOption(target, "automatic-rename", "off")

	if len(layout) == 0 {
		return nil
	}

	// First entry is the initial pane (no split needed).
	// Subsequent entries split the last created pane.
	// paneIndex tracks the tmux pane number of the last created pane.
	lastPane := 1 // tmux pane indices start at 1
	activePaneIdx := 1

	for i, pane := range layout {
		if i > 0 {
			// Split the last created pane
			horizontal := pane.Split == "horizontal"
			percent := pane.Percent
			if percent == 0 {
				percent = 50
			}
			splitTarget := fmt.Sprintf("%s.%d", target, lastPane)
			if err := SplitWindow(splitTarget, dir, horizontal, percent); err != nil {
				return fmt.Errorf("split pane %d: %w", i, err)
			}
			lastPane = i + 1
		}

		if pane.Active {
			activePaneIdx = i + 1
		}
	}

	time.Sleep(100 * time.Millisecond)

	// Send commands to all panes
	for i, pane := range layout {
		if pane.Command != "" {
			paneTarget := fmt.Sprintf("%s.%d", target, i+1)
			if err := SendKeys(paneTarget, pane.Command); err != nil {
				return fmt.Errorf("send keys to pane %d: %w", i, err)
			}
		}
	}

	time.Sleep(100 * time.Millisecond)

	SelectPane(fmt.Sprintf("%s.%d", target, activePaneIdx))

	return nil
}

func run(args ...string) error {
	cmd := exec.Command("tmux", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
