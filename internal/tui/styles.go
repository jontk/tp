package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#BD93F9")).
			MarginBottom(1)

	filterPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6272A4"))

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B")).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B"))

	unselectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2"))

	openStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4"))

	openBadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Italic(true)

	closingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555"))

	closingBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF5555")).
				Italic(true)

	dirStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Faint(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			MarginTop(1)

	counterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF79C6"))

	previewBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#6272A4")).
				Padding(1, 2)

	previewLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#BD93F9")).
				Bold(true)

	previewValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F8F8F2"))

	previewCleanStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#50FA7B"))

	previewDirtyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFB86C"))
)
