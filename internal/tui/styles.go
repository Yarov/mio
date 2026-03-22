package tui

import "github.com/charmbracelet/lipgloss"

// Mio color palette
var (
	colorPrimary   = lipgloss.Color("#3B82F6") // Blue
	colorSecondary = lipgloss.Color("#10B981") // Green
	colorAccent    = lipgloss.Color("#F59E0B") // Amber
	colorMuted     = lipgloss.Color("#6B7280") // Gray
	colorDanger    = lipgloss.Color("#EF4444") // Red
	colorBase      = lipgloss.Color("#111827") // Dark background
	colorSurface   = lipgloss.Color("#1F2937") // Surface
	colorText      = lipgloss.Color("#F9FAFB") // Light text
	colorSubtext   = lipgloss.Color("#9CA3AF") // Dimmer text
)

// Layout
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorSubtext).
			MarginTop(1)
)

// Dashboard
var (
	logoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	statNumberStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	statLabelStyle = lipgloss.NewStyle().
			Foreground(colorSubtext)

	statCardStyle = lipgloss.NewStyle().
			Padding(0, 2).
			MarginRight(2)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(colorText).
			PaddingLeft(2)

	menuSelectedStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				PaddingLeft(1).
				SetString("> ")
)

// Lists
var (
	listItemStyle = lipgloss.NewStyle().
			Foreground(colorText).
			PaddingLeft(2)

	listSelectedStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				PaddingLeft(1)

	idStyle = lipgloss.NewStyle().
		Foreground(colorMuted)

	timestampStyle = lipgloss.NewStyle().
			Foreground(colorSubtext)

	projectStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	scoreStyle = lipgloss.NewStyle().
			Foreground(colorAccent)

	contentPreviewStyle = lipgloss.NewStyle().
				Foreground(colorSubtext)
)

// Detail view
var (
	detailLabelStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Bold(true).
				Width(12)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(colorText)

	detailContentStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Padding(1, 2).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorMuted)

	sectionHeadingStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true).
				MarginTop(1).
				MarginBottom(1)
)

// Timeline
var (
	timelineFocusStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	timelineItemStyle = lipgloss.NewStyle().
				Foreground(colorText)

	timelineConnectorStyle = lipgloss.NewStyle().
				Foreground(colorMuted)
)

// Type badges
func typeBadgeStyle(obsType string) lipgloss.Style {
	color := colorMuted
	switch obsType {
	case "bugfix":
		color = colorDanger
	case "decision":
		color = colorPrimary
	case "architecture":
		color = lipgloss.Color("#8B5CF6") // Purple
	case "discovery":
		color = colorAccent
	case "pattern":
		color = colorSecondary
	case "config":
		color = lipgloss.Color("#06B6D4") // Cyan
	case "preference":
		color = lipgloss.Color("#EC4899") // Pink
	case "learning":
		color = lipgloss.Color("#F97316") // Orange
	case "summary":
		color = colorSubtext
	}
	return lipgloss.NewStyle().
		Foreground(color).
		Bold(true)
}
