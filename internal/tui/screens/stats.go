package screens

import (
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/mackeper/m_backuper/internal/config"
	"github.com/mackeper/m_backuper/internal/display"
	"github.com/mackeper/m_backuper/internal/index"
	"github.com/mackeper/m_backuper/internal/stats"
)

// StatsModel displays database statistics
type StatsModel struct {
	db         *index.DB
	cfg        *config.Config
	logger     *slog.Logger
	calculator *stats.Calculator
	stats      *display.DatabaseStats
	loading    bool
	err        error
}

type statsLoadedMsg struct {
	stats *display.DatabaseStats
	err   error
}

// NewStatsModel creates a new stats viewer model
func NewStatsModel(db *index.DB, cfg *config.Config, logger *slog.Logger) StatsModel {
	calculator := stats.NewCalculator(db, cfg.Database.Path)
	return StatsModel{
		db:         db,
		cfg:        cfg,
		logger:     logger,
		calculator: calculator,
		loading:    true,
	}
}

func (m StatsModel) Init() tea.Cmd {
	return m.loadStats()
}

func (m StatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return NavigateMsg{Screen: "menu"} }
		case "r":
			m.loading = true
			return m, m.loadStats()
		}

	case statsLoadedMsg:
		m.loading = false
		m.stats = msg.stats
		m.err = msg.err
	}

	return m, nil
}

func (m StatsModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Padding(1, 0)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(1, 0)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	valueStyle := lipgloss.NewStyle().Bold(true)

	s := titleStyle.Render("Database Statistics") + "\n\n"

	if m.loading {
		s += "Loading statistics...\n"
		return s
	}

	if m.err != nil {
		s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n"
		return s
	}

	if m.stats == nil {
		s += "No statistics available\n"
		return s
	}

	// Overall statistics
	s += headerStyle.Render("Overall") + "\n"
	s += fmt.Sprintf("%s %s\n", labelStyle.Render("Total files:"), valueStyle.Render(humanize.Comma(m.stats.TotalFiles)))
	s += fmt.Sprintf("%s %s\n", labelStyle.Render("Total size:"), valueStyle.Render(humanize.Bytes(uint64(m.stats.TotalSize))))
	s += fmt.Sprintf("%s %s\n", labelStyle.Render("Database size:"), valueStyle.Render(humanize.Bytes(uint64(m.stats.DatabaseSize))))
	s += fmt.Sprintf("%s %s\n\n", labelStyle.Render("Database path:"), valueStyle.Render(m.stats.DatabasePath))

	// Backup sets
	if len(m.stats.BackupSets) > 0 {
		s += headerStyle.Render("Backup Sets") + "\n"
		for _, bs := range m.stats.BackupSets {
			s += fmt.Sprintf("  %s\n", valueStyle.Render(bs.Name))
			s += fmt.Sprintf("    %s %s\n", labelStyle.Render("Files:"), humanize.Comma(bs.Count))
			s += fmt.Sprintf("    %s %s\n", labelStyle.Render("Size:"), humanize.Bytes(uint64(bs.Size)))
		}
		s += "\n"
	}

	// Root directories
	if len(m.stats.RootDirectories) > 0 {
		s += headerStyle.Render("Top Root Directories") + "\n"
		for i, dir := range m.stats.RootDirectories {
			if i >= 10 {
				break
			}
			s += fmt.Sprintf("  %s\n", valueStyle.Render(dir.Path))
			s += fmt.Sprintf("    %s %d\n", labelStyle.Render("Files:"), dir.Count)
		}
		s += "\n"
	}

	s += helpStyle.Render("r: refresh | esc/q: back to menu")

	return s
}

func (m StatsModel) loadStats() tea.Cmd {
	return func() tea.Msg {
		stats, err := m.calculator.Calculate()
		return statsLoadedMsg{stats: stats, err: err}
	}
}
