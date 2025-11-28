package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type menuItem struct {
	title       string
	description string
	screen      string
}

var menuItems = []menuItem{
	{
		title:       "Config Manager",
		description: "Add, edit, and remove backup sets",
		screen:      "config",
	},
	{
		title:       "Statistics",
		description: "View database statistics and backup info",
		screen:      "stats",
	},
	{
		title:       "Duplicate Browser",
		description: "Browse and clean duplicate files",
		screen:      "duplicates",
	},
	{
		title:       "Backup Runner",
		description: "Run backups with real-time progress",
		screen:      "backup",
	},
	{
		title:       "Quit",
		description: "Exit the application",
		screen:      "quit",
	},
}

// MenuModel represents the main menu
type MenuModel struct {
	cursor int
}

// NewMenuModel creates a new menu model
func NewMenuModel() MenuModel {
	return MenuModel{
		cursor: 0,
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(menuItems)-1 {
				m.cursor++
			}
		case "enter", " ":
			// Navigate to selected screen
			return m, func() tea.Msg {
				return NavigateMsg{Screen: menuItems[m.cursor].screen}
			}
		}
	}

	return m, nil
}

func (m MenuModel) View() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Padding(1, 0)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 0, 1, 0)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true).
		Padding(1, 0, 0, 0)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Bold(true).
		Padding(0, 2)

	normalStyle := lipgloss.NewStyle().
		Padding(0, 2)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 4)

	// Build the menu
	s := titleStyle.Render("m_backuper - Backup & Duplicate Manager") + "\n"
	s += subtitleStyle.Render("Select an option:") + "\n\n"

	for i, item := range menuItems {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}

		var title string
		if m.cursor == i {
			title = selectedStyle.Render(fmt.Sprintf("%s%s", cursor, item.title))
		} else {
			title = normalStyle.Render(fmt.Sprintf("%s%s", cursor, item.title))
		}

		s += title + "\n"
		if m.cursor == i {
			s += descStyle.Render(item.description) + "\n"
		}
		s += "\n"
	}

	s += helpStyle.Render("Navigate: ↑/↓ or j/k | Select: enter | Quit: q or ctrl+c")

	return s
}
