package tui

import (
	"strings"

	"github.com/jontk/tp/internal/config"
)

// rect represents a pane's position in the grid
type rect struct {
	x, y, w, h int
	command    string
	active     bool
}

// RenderLayout generates an ASCII diagram from a layout config.
func RenderLayout(layout []config.PaneConfig, width, height int) string {
	if len(layout) == 0 || width < 6 || height < 3 {
		return ""
	}

	// Simulate splits to get pane rectangles
	rects := simulateSplits(layout, width, height)

	// Build character grid
	grid := make([][]rune, height)
	for y := range grid {
		grid[y] = make([]rune, width)
		for x := range grid[y] {
			grid[y][x] = ' '
		}
	}

	// Draw borders for each pane
	for _, r := range rects {
		// Top and bottom edges
		for x := r.x; x < r.x+r.w && x < width; x++ {
			safeSet(grid, r.y, x, '─')
			safeSet(grid, r.y+r.h-1, x, '─')
		}
		// Left and right edges
		for y := r.y; y < r.y+r.h && y < height; y++ {
			safeSet(grid, y, r.x, '│')
			safeSet(grid, y, r.x+r.w-1, '│')
		}
	}

	// Fix all intersections
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ch := grid[y][x]
			if ch != '─' && ch != '│' && ch != '+' {
				continue
			}
			up := y > 0 && isVert(grid[y-1][x])
			down := y < height-1 && isVert(grid[y+1][x])
			left := x > 0 && isHoriz(grid[y][x-1])
			right := x < width-1 && isHoriz(grid[y][x+1])

			grid[y][x] = junction(up, down, left, right)
		}
	}

	// Draw labels
	for _, r := range rects {
		label := r.command
		if r.active {
			label += " *"
		}
		maxLen := r.w - 2
		if maxLen < 1 {
			continue
		}
		if len(label) > maxLen {
			label = label[:maxLen]
		}
		labelY := r.y + r.h/2
		labelX := r.x + (r.w-len(label))/2
		for i, ch := range label {
			px := labelX + i
			if px > r.x && px < r.x+r.w-1 && labelY > r.y && labelY < r.y+r.h-1 {
				grid[labelY][px] = ch
			}
		}
	}

	var b strings.Builder
	for _, row := range grid {
		b.WriteString(string(row))
		b.WriteRune('\n')
	}
	return b.String()
}

func simulateSplits(layout []config.PaneConfig, w, h int) []rect {
	rects := []rect{{x: 0, y: 0, w: w, h: h, command: shortCmd(layout[0].Command), active: layout[0].Active}}

	for i := 1; i < len(layout); i++ {
		cfg := layout[i]
		last := &rects[len(rects)-1]

		var r rect
		if cfg.Split == "horizontal" {
			splitAt := last.w * (100 - cfg.Percent) / 100
			// Share the border: new pane starts at the split line
			r = rect{x: last.x + splitAt, y: last.y, w: last.w - splitAt, h: last.h}
			last.w = splitAt + 1 // +1 to include shared border
		} else {
			splitAt := last.h * (100 - cfg.Percent) / 100
			r = rect{x: last.x, y: last.y + splitAt, w: last.w, h: last.h - splitAt}
			last.h = splitAt + 1 // +1 to include shared border
		}
		r.command = shortCmd(cfg.Command)
		r.active = cfg.Active
		rects = append(rects, r)
	}
	return rects
}

func safeSet(grid [][]rune, y, x int, ch rune) {
	if y >= 0 && y < len(grid) && x >= 0 && x < len(grid[0]) {
		existing := grid[y][x]
		if existing == ' ' {
			grid[y][x] = ch
		} else if (existing == '─' && ch == '│') || (existing == '│' && ch == '─') {
			// Mark as needing junction resolution
			grid[y][x] = '+'
		}
	}
}

func isVert(ch rune) bool {
	return ch == '│' || ch == '+' || ch == '┌' || ch == '┐' || ch == '└' || ch == '┘' ||
		ch == '├' || ch == '┤' || ch == '┬' || ch == '┴' || ch == '┼'
}

func isHoriz(ch rune) bool {
	return ch == '─' || ch == '+' || ch == '┌' || ch == '┐' || ch == '└' || ch == '┘' ||
		ch == '├' || ch == '┤' || ch == '┬' || ch == '┴' || ch == '┼'
}

func junction(up, down, left, right bool) rune {
	switch {
	case up && down && left && right:
		return '┼'
	case down && right && !up && !left:
		return '┌'
	case down && left && !up && !right:
		return '┐'
	case up && right && !down && !left:
		return '└'
	case up && left && !down && !right:
		return '┘'
	case up && down && right && !left:
		return '├'
	case up && down && left && !right:
		return '┤'
	case down && left && right && !up:
		return '┬'
	case up && left && right && !down:
		return '┴'
	case up && down:
		return '│'
	case left && right:
		return '─'
	default:
		return '+'
	}
}

func shortCmd(cmd string) string {
	if cmd == "" {
		return "shell"
	}
	if len(cmd) > 12 {
		parts := strings.Fields(cmd)
		if len(parts) > 0 {
			cmd = parts[0]
			if len(cmd) > 12 {
				cmd = cmd[:12]
			}
		}
	}
	return cmd
}
