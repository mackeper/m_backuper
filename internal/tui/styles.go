package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	colorPrimary   = lipgloss.Color("86")  // Cyan
	colorSecondary = lipgloss.Color("212") // Pink
	colorSuccess   = lipgloss.Color("42")  // Green
	colorWarning   = lipgloss.Color("214") // Orange
	colorError     = lipgloss.Color("196") // Red
	colorSubtle    = lipgloss.Color("241") // Gray
	colorHighlight = lipgloss.Color("226") // Yellow
)

// Common styles
var (
	// Title style for main headers
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1)

	// Subtitle style for section headers
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	// Menu item styles
	MenuItemStyle = lipgloss.NewStyle().
			Padding(0, 2)

	SelectedMenuItemStyle = lipgloss.NewStyle().
				Foreground(colorHighlight).
				Bold(true).
				Padding(0, 2).
				Background(lipgloss.Color("237"))

	// Status styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true)

	// Help text style
	HelpStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Italic(true)

	// Border styles
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	// List item styles
	ListItemStyle = lipgloss.NewStyle().
			Padding(0, 2)

	SelectedListItemStyle = lipgloss.NewStyle().
				Foreground(colorHighlight).
				Bold(true).
				Padding(0, 2).
				Background(lipgloss.Color("237"))

	// Input styles
	FocusedInputStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	BlurredInputStyle = lipgloss.NewStyle().
				Foreground(colorSubtle)

	// Progress bar styles
	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(colorSuccess)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(colorSubtle)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)
)

// Helper functions for common UI elements

// RenderTitle renders a styled title with optional subtitle
func RenderTitle(title, subtitle string) string {
	result := TitleStyle.Render(title)
	if subtitle != "" {
		result += "\n" + SubtitleStyle.Render(subtitle)
	}
	return result
}

// RenderHelp renders help text at the bottom of the screen
func RenderHelp(helpText string) string {
	return "\n" + HelpStyle.Render(helpText)
}

// RenderError renders an error message
func RenderError(err error) string {
	if err == nil {
		return ""
	}
	return ErrorStyle.Render("Error: " + err.Error())
}

// RenderSuccess renders a success message
func RenderSuccess(msg string) string {
	return SuccessStyle.Render("✓ " + msg)
}

// RenderWarning renders a warning message
func RenderWarning(msg string) string {
	return WarningStyle.Render("⚠ " + msg)
}
