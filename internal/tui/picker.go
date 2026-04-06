package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jontk/tp/internal/config"
	"github.com/jontk/tp/internal/projects"
)

type gitInfoMsg struct {
	path string
	info projects.GitInfo
}

type Model struct {
	projects    []projects.Project
	selected    map[int]bool
	openInTmux  map[string]bool
	cursor      int
	filtered    []int
	filter      textinput.Model
	sortMode    string
	quitting    bool
	confirmed   bool
	width       int
	height      int
	infoCache   map[string]projects.GitInfo
	infoPending string // path currently being fetched
	showPreview bool
	cfg         *config.Config
}

func NewPicker(projs []projects.Project, defaults []string, openWindows map[string]bool, sortMode string, showPreview bool, cfg *config.Config) Model {
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
		wname := p.WindowName()
		if openWindows[wname] || (defaultSet[wname] && !openWindows[wname]) {
			selected[i] = true
		}
		// Also match defaults by plain name for backwards compatibility
		if !selected[i] && defaultSet[p.Name] && !openWindows[wname] {
			selected[i] = true
		}
	}

	filtered := make([]int, len(projs))
	for i := range projs {
		filtered[i] = i
	}

	return Model{
		projects:    projs,
		selected:    selected,
		openInTmux:  openWindows,
		filtered:    filtered,
		filter:      ti,
		sortMode:    sortMode,
		showPreview: showPreview,
		infoCache:   make(map[string]projects.GitInfo),
		cfg:         cfg,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.fetchCurrentInfo())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case gitInfoMsg:
		m.infoCache[msg.path] = msg.info
		if m.infoPending == msg.path {
			m.infoPending = ""
		}
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
			return m, m.fetchCurrentInfo()

		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, m.fetchCurrentInfo()

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
			return m, m.fetchCurrentInfo()

		case "ctrl+s":
			// Track selected by path before re-sorting (path is unique)
			selectedPaths := make(map[string]bool)
			for idx, sel := range m.selected {
				if sel {
					selectedPaths[m.projects[idx].Path] = true
				}
			}
			if m.sortMode == "recent" {
				m.sortMode = "alphabetical"
			} else {
				m.sortMode = "recent"
			}
			projects.SortProjects(m.projects, m.sortMode)
			// Rebuild selected map with new indices
			m.selected = make(map[int]bool)
			for i, p := range m.projects {
				if selectedPaths[p.Path] {
					m.selected[i] = true
				}
			}
			m.cursor = 0
			m.applyFilter()
			return m, m.fetchCurrentInfo()
		}
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.applyFilter()

	return m, cmd
}

func (m Model) fetchCurrentInfo() tea.Cmd {
	if !m.showPreview || len(m.filtered) == 0 {
		return nil
	}
	idx := m.filtered[m.cursor]
	p := m.projects[idx]

	// Already cached
	if _, ok := m.infoCache[p.Path]; ok {
		return nil
	}

	// Already fetching this one
	if m.infoPending == p.Path {
		return nil
	}

	path := p.Path
	return func() tea.Msg {
		return gitInfoMsg{
			path: path,
			info: projects.FetchGitInfo(path),
		}
	}
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

	// Header
	var header strings.Builder
	header.WriteString(titleStyle.Render("tp — tmux project picker"))
	header.WriteString("\n")
	header.WriteString(m.filter.View())
	header.WriteString("\n\n")

	selectedCount := 0
	for _, v := range m.selected {
		if v {
			selectedCount++
		}
	}
	sortLabel := "recent"
	if m.sortMode == "alphabetical" {
		sortLabel = "a-z"
	}
	header.WriteString(counterStyle.Render(fmt.Sprintf("%d selected", selectedCount)))
	header.WriteString("  ")
	header.WriteString(dirStyle.Render(fmt.Sprintf("[%s]", sortLabel)))
	header.WriteString("\n\n")

	// Calculate layout
	showPreview := m.showPreview && m.width >= 80
	listWidth := m.width
	previewWidth := 0
	if showPreview {
		listWidth = m.width / 2
		previewWidth = m.width - listWidth - 3
	}

	maxVisible := m.height - 9
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

	// Build project list
	var list strings.Builder
	for vi := scrollOffset; vi < visibleEnd; vi++ {
		idx := m.filtered[vi]
		p := m.projects[idx]

		cursor := "  "
		if vi == m.cursor {
			cursor = cursorStyle.Render("> ")
		}

		var line string
		isOpen := m.openInTmux[p.WindowName()]
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

		var meta []string
		if rt := projects.RelativeTime(p.LastCommit); rt != "" {
			meta = append(meta, rt)
		}
		if p.HasDuplicate {
			meta = append(meta, filepath.Base(p.Dir))
		}

		suffix := ""
		if len(meta) > 0 {
			suffix = "  " + dirStyle.Render(strings.Join(meta, " · "))
		}
		list.WriteString(fmt.Sprintf("%s%s%s\n", cursor, line, suffix))
	}

	if len(m.filtered) == 0 {
		list.WriteString(openStyle.Render("  no matches\n"))
	}

	// Build preview panel
	listStr := list.String()
	if showPreview && previewWidth > 20 && len(m.filtered) > 0 {
		preview := m.renderPreview(previewWidth)
		listStr = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(listWidth).Render(listStr),
			preview,
		)
	}

	help := "space/tab: toggle • enter: confirm • ctrl+a: all • ctrl+s: sort • esc: cancel"

	return header.String() + listStr + "\n" + helpStyle.Render(help)
}

func (m Model) renderPreview(width int) string {
	if len(m.filtered) == 0 {
		return ""
	}

	idx := m.filtered[m.cursor]
	p := m.projects[idx]

	info, ok := m.infoCache[p.Path]
	if !ok {
		return previewBorderStyle.Width(width).Render(
			previewLabelStyle.Render(p.Name) + "\n\n" +
				dirStyle.Render("loading..."),
		)
	}

	var b strings.Builder

	b.WriteString(previewLabelStyle.Render(p.Name))
	b.WriteString("\n\n")

	if info.Branch != "" {
		b.WriteString(previewLabelStyle.Render("branch  "))
		b.WriteString(previewValueStyle.Render(info.Branch))
		b.WriteString("\n")
	}

	if info.CommitMsg != "" {
		b.WriteString(previewLabelStyle.Render("commit  "))
		msg := info.CommitMsg
		maxMsg := width - 12
		if maxMsg > 0 && len(msg) > maxMsg {
			msg = msg[:maxMsg-3] + "..."
		}
		b.WriteString(previewValueStyle.Render(info.CommitHash + " " + msg))
		b.WriteString("\n")
	}

	if info.Author != "" {
		b.WriteString(previewLabelStyle.Render("author  "))
		b.WriteString(previewValueStyle.Render(info.Author))
		b.WriteString("\n")
	}

	if info.Status != "" {
		b.WriteString(previewLabelStyle.Render("status  "))
		if info.Status == "clean" {
			b.WriteString(previewCleanStyle.Render(info.Status))
		} else {
			b.WriteString(previewDirtyStyle.Render(info.Status))
		}
		b.WriteString("\n")
	}

	if info.RemoteURL != "" {
		b.WriteString(previewLabelStyle.Render("remote  "))
		b.WriteString(previewValueStyle.Render(info.RemoteURL))
		b.WriteString("\n")
	}

	if rt := projects.RelativeTime(p.LastCommit); rt != "" {
		b.WriteString(previewLabelStyle.Render("active  "))
		b.WriteString(dirStyle.Render(rt))
		b.WriteString("\n")
	}

	// Layout diagram
	if m.cfg != nil {
		layout := m.cfg.LayoutForProject(p.Name)
		if len(layout) > 1 {
			diagW := width - 2
			diagH := 7
			if diagW > 10 {
				b.WriteString("\n")
				b.WriteString(dirStyle.Render(RenderLayout(layout, diagW, diagH)))
			}
		}
	}

	return previewBorderStyle.Width(width).Render(b.String())
}

func (m Model) Selected() []projects.Project {
	var result []projects.Project
	for idx, sel := range m.selected {
		if sel && !m.openInTmux[m.projects[idx].WindowName()] {
			result = append(result, m.projects[idx])
		}
	}
	return result
}

func (m Model) Closed() []projects.Project {
	var result []projects.Project
	for idx, p := range m.projects {
		if m.openInTmux[p.WindowName()] && !m.selected[idx] {
			result = append(result, p)
		}
	}
	return result
}

func (m Model) Confirmed() bool {
	return m.confirmed
}
