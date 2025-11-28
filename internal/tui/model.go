package tui

import (
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mackeper/m_backuper/internal/config"
	"github.com/mackeper/m_backuper/internal/index"
	"github.com/mackeper/m_backuper/internal/tui/screens"
)

// Screen represents different views in the TUI
type Screen int

const (
	ScreenMenu Screen = iota
	ScreenConfig
	ScreenStats
	ScreenDuplicates
	ScreenBackup
	ScreenQuit
)

// Model is the main TUI model
type Model struct {
	cfg        *config.Config
	configPath string
	db         *index.DB
	logger     *slog.Logger
	currentScreen Screen
	width      int
	height     int
	err        error

	// Screen models
	menuModel       tea.Model
	configModel     tea.Model
	statsModel      tea.Model
	duplicatesModel tea.Model
	backupModel     tea.Model
}

// NewModel creates a new TUI model
func NewModel(cfg *config.Config, configPath string, db *index.DB, logger *slog.Logger) Model {
	return Model{
		cfg:           cfg,
		configPath:    configPath,
		db:            db,
		logger:        logger,
		currentScreen: ScreenMenu,
		menuModel:     screens.NewMenuModel(),
	}
}

// Init initializes the TUI
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.currentScreen == ScreenMenu {
				return m, tea.Quit
			}
			// Return to menu from other screens
			m.currentScreen = ScreenMenu
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case screens.NavigateMsg:
		// Handle navigation between screens
		switch msg.Screen {
		case "config":
			m.currentScreen = ScreenConfig
			if m.configModel == nil {
				m.configModel = screens.NewConfigModel(m.cfg, m.configPath, m.logger)
			}
		case "stats":
			m.currentScreen = ScreenStats
			if m.statsModel == nil {
				m.statsModel = screens.NewStatsModel(m.db, m.cfg, m.logger)
			}
		case "duplicates":
			m.currentScreen = ScreenDuplicates
			if m.duplicatesModel == nil {
				m.duplicatesModel = screens.NewDuplicatesModel(m.db, m.cfg, m.logger)
			}
		case "backup":
			m.currentScreen = ScreenBackup
			if m.backupModel == nil {
				m.backupModel = screens.NewBackupModel(m.cfg, m.db, m.logger)
			}
		case "quit":
			return m, tea.Quit
		}
		return m, nil
	}

	// Delegate to current screen
	var cmd tea.Cmd
	switch m.currentScreen {
	case ScreenMenu:
		m.menuModel, cmd = m.menuModel.Update(msg)
	case ScreenConfig:
		m.configModel, cmd = m.configModel.Update(msg)
	case ScreenStats:
		m.statsModel, cmd = m.statsModel.Update(msg)
	case ScreenDuplicates:
		m.duplicatesModel, cmd = m.duplicatesModel.Update(msg)
	case ScreenBackup:
		m.backupModel, cmd = m.backupModel.Update(msg)
	}

	return m, cmd
}

// View renders the current screen
func (m Model) View() string {
	if m.err != nil {
		return RenderError(m.err)
	}

	switch m.currentScreen {
	case ScreenMenu:
		return m.menuModel.View()
	case ScreenConfig:
		return m.configModel.View()
	case ScreenStats:
		return m.statsModel.View()
	case ScreenDuplicates:
		return m.duplicatesModel.View()
	case ScreenBackup:
		return m.backupModel.View()
	default:
		return "Unknown screen"
	}
}

// Run starts the TUI application
func Run(cfg *config.Config, configPath string, db *index.DB, logger *slog.Logger) error {
	p := tea.NewProgram(NewModel(cfg, configPath, db, logger), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
