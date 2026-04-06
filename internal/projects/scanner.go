package projects

import (
	"os"
	"path/filepath"
	"sort"
)

type Project struct {
	Name string
	Path string
	Dir  string // parent source directory
}

func Scan(dirs []string) ([]Project, error) {
	var projects []Project
	seen := make(map[string]bool)

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

			name := entry.Name()
			if seen[name] {
				continue
			}
			seen[name] = true

			projects = append(projects, Project{
				Name: name,
				Path: filepath.Join(dir, name),
				Dir:  dir,
			})
		}
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}
