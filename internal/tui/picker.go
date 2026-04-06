package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jontk/tp/internal/projects"
)

type Model struct {
	projects   []projects.Project
	selected   map[int]bool
	openInTmux map[string]bool
	cursor     int
	filtered   []int
	filter     textinput.Model
	quitting   bool
	confirmed  bool
	width      int
	height     int
}

func NewPicker(projs []projects.Project, defaults []string, openWindows map[string]bool) Model {
	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.Prompt = "/ "
	ti.PromptStyle = filterPromptStyle
	ti.Focus()

	selected := make(map[int]bool)
	defaultSet := make(map[string]bool)
	for _, d := range defaults {
		defaultSet[d] = true
	}

	for i, p := range projs {
		if openWindows[p.Name] || (defaultSet[p.Name] && !openWindows[p.Name]) {
			selected[i] = true
		}
	}

	filtered := make([]int, len(projs))
	for i := range projs {
		filtered[i] = i
	}

	return Model{
		projects:   projs,
		selected:   selected,
		openInTmux: openWindows,
		filtered:   filtered,
		filter:     ti,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			m.confirmed = true
			return m, tea.Quit

		case " ":
			if len(m.filtered) > 0 {
				idx := m.filtered[m.cursor]
				m.selected[idx] = !m.selected[idx]
			}
			return m, nil

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil

		case "ctrl+a":
			allSelected := true
			for _, idx := range m.filtered {
				if !m.selected[idx] {
					allSelected = false
					break
				}
			}
			for _, idx := range m.filtered {
				m.selected[idx] = !allSelected
			}
			return m, nil

		case "tab":
			if len(m.filtered) > 0 {
				idx := m.filtered[m.cursor]
				m.selected[idx] = !m.selected[idx]
			}
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	// Update filter input
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.applyFilter()

	return m, cmd
}

func (m *Model) applyFilter() {
	query := strings.ToLower(m.filter.Value())
	m.filtered = m.filtered[:0]

	for i, p := range m.projects {
		if query == "" || strings.Contains(strings.ToLower(p.Name), query) {
			m.filtered = append(m.filtered, i)
		}
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m Model) View() string {
	if m.quitting && !m.confirmed {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("tp — tmux project picker"))
	b.WriteString("\n")
	b.WriteString(m.filter.View())
	b.WriteString("\n\n")

	// Count selected
	selectedCount := 0
	for _, v := range m.selected {
		if v {
			selectedCount++
		}
	}
	b.WriteString(counterStyle.Render(fmt.Sprintf("%d selected", selectedCount)))
	b.WriteString("\n\n")

	// Calculate visible area
	maxVisible := m.height - 9 // title + filter + counter + help + margins
	if maxVisible < 5 {
		maxVisible = 20
	}

	// Scroll offset
	scrollOffset := 0
	if m.cursor >= maxVisible {
		scrollOffset = m.cursor - maxVisible + 1
	}

	visibleEnd := scrollOffset + maxVisible
	if visibleEnd > len(m.filtered) {
		visibleEnd = len(m.filtered)
	}

	for vi := scrollOffset; vi < visibleEnd; vi++ {
		idx := m.filtered[vi]
		p := m.projects[idx]

		// Cursor
		cursor := "  "
		if vi == m.cursor {
			cursor = cursorStyle.Render("> ")
		}

		// Checkbox and name
		var line string
		isOpen := m.openInTmux[p.Name]
		isSelected := m.selected[idx]

		switch {
		case isOpen && isSelected:
			line = openStyle.Render(fmt.Sprintf("[x] %s", p.Name)) +
				" " + openBadgeStyle.Render("(open)")
		case isOpen && !isSelected:
			line = closingStyle.Render(fmt.Sprintf("[ ] %s", p.Name)) +
				" " + closingBadgeStyle.Render("(closing)")
		case isSelected:
			line = selectedStyle.Render(fmt.Sprintf("[x] %s", p.Name))
		default:
			line = unselectedStyle.Render(fmt.Sprintf("[ ] %s", p.Name))
		}

		// Show parent dir basename for disambiguation
		dir := dirStyle.Render(filepath.Base(p.Dir))

		b.WriteString(fmt.Sprintf("%s%s  %s\n", cursor, line, dir))
	}

	if len(m.filtered) == 0 {
		b.WriteString(openStyle.Render("  no matches\n"))
	}

	help := "space/tab: toggle • enter: confirm • ctrl+a: all • esc: cancel"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func (m Model) Selected() []projects.Project {
	var result []projects.Project
	for idx, sel := range m.selected {
		if sel && !m.openInTmux[m.projects[idx].Name] {
			result = append(result, m.projects[idx])
		}
	}
	return result
}

func (m Model) Closed() []projects.Project {
	var result []projects.Project
	for idx, p := range m.projects {
		if m.openInTmux[p.Name] && !m.selected[idx] {
			result = append(result, p)
		}
	}
	return result
}

func (m Model) Confirmed() bool {
	return m.confirmed
}
