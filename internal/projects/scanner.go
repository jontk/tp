package projects

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Project struct {
	Name         string
	Path         string
	Dir          string // parent source directory
	LastCommit   time.Time
	HasDuplicate bool // true if another project has the same name
}

// WindowName returns the tmux window name. For duplicates, includes
// the parent dir to disambiguate (e.g. "brokkr/jontk").
func (p Project) WindowName() string {
	if p.HasDuplicate {
		return filepath.Base(p.Dir) + "/" + p.Name
	}
	return p.Name
}

func Scan(dirs []string, sortMode string) ([]Project, error) {
	var projects []Project

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		for _, entry := range entries {
			if !entry.IsDir() || entry.Name()[0] == '.' {
				continue
			}

			p := Project{
				Name: entry.Name(),
				Path: filepath.Join(dir, entry.Name()),
				Dir:  dir,
			}

			if sortMode == "recent" {
				p.LastCommit = gitLastCommit(p.Path)
			}

			projects = append(projects, p)
		}
	}

	// Mark projects that share a name across different source dirs
	nameCounts := make(map[string]int)
	for _, p := range projects {
		nameCounts[p.Name]++
	}
	for i := range projects {
		if nameCounts[projects[i].Name] > 1 {
			projects[i].HasDuplicate = true
		}
	}

	SortProjects(projects, sortMode)

	return projects, nil
}

func SortProjects(projects []Project, mode string) {
	switch mode {
	case "recent":
		sort.Slice(projects, func(i, j int) bool {
			// Projects with commits come before those without
			if projects[i].LastCommit.IsZero() != projects[j].LastCommit.IsZero() {
				return !projects[i].LastCommit.IsZero()
			}
			// Most recent first
			return projects[i].LastCommit.After(projects[j].LastCommit)
		})
	default:
		sort.Slice(projects, func(i, j int) bool {
			return projects[i].Name < projects[j].Name
		})
	}
}

func gitLastCommit(dir string) time.Time {
	cmd := exec.Command("git", "-C", dir, "log", "-1", "--format=%ct")
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}

	ts, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return time.Time{}
	}

	return time.Unix(ts, 0)
}

func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
	}
}
