package projects

import (
	"fmt"
	"os/exec"
	"strings"
)

type GitInfo struct {
	Branch     string
	CommitMsg  string
	CommitHash string
	Author     string
	Status     string // "clean" or "3 modified, 1 untracked"
	RemoteURL  string
}

func FetchGitInfo(dir string) GitInfo {
	info := GitInfo{}

	// Branch
	if out, err := gitCmd(dir, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		info.Branch = out
	}

	// Last commit
	if out, err := gitCmd(dir, "log", "-1", "--format=%h|%an|%s"); err == nil {
		parts := strings.SplitN(out, "|", 3)
		if len(parts) == 3 {
			info.CommitHash = parts[0]
			info.Author = parts[1]
			info.CommitMsg = parts[2]
		}
	}

	// Status summary
	if out, err := gitCmd(dir, "status", "--porcelain"); err == nil {
		if out == "" {
			info.Status = "clean"
		} else {
			lines := strings.Split(out, "\n")
			modified := 0
			untracked := 0
			for _, l := range lines {
				if l == "" {
					continue
				}
				if strings.HasPrefix(l, "??") {
					untracked++
				} else {
					modified++
				}
			}
			var parts []string
			if modified > 0 {
				parts = append(parts, pluralize(modified, "modified"))
			}
			if untracked > 0 {
				parts = append(parts, pluralize(untracked, "untracked"))
			}
			info.Status = strings.Join(parts, ", ")
		}
	}

	// Remote URL
	if out, err := gitCmd(dir, "remote", "get-url", "origin"); err == nil {
		info.RemoteURL = cleanRemoteURL(out)
	}

	return info
}

func gitCmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func cleanRemoteURL(url string) string {
	// git@github.com:user/repo.git -> user/repo
	url = strings.TrimSuffix(url, ".git")
	if strings.HasPrefix(url, "git@github.com:") {
		return strings.TrimPrefix(url, "git@github.com:")
	}
	if strings.HasPrefix(url, "https://github.com/") {
		return strings.TrimPrefix(url, "https://github.com/")
	}
	return url
}

func pluralize(n int, word string) string {
	return fmt.Sprintf("%d %s", n, word)
}
