package screens

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/mackeper/m_backuper/internal/backup"
	"github.com/mackeper/m_backuper/internal/config"
	"github.com/mackeper/m_backuper/internal/index"
	"github.com/mackeper/m_backuper/internal/operations"
)

type backupMode int

const (
	backupModeSelect backupMode = iota
	backupModeRunning
	backupModeComplete
)

type backupSetSelectItem struct {
	name        string
	destination string
}

func (i backupSetSelectItem) Title() string       { return i.name }
func (i backupSetSelectItem) Description() string { return "Destination: " + i.destination }
func (i backupSetSelectItem) FilterValue() string { return i.name }

// BackupModel manages backup operations with progress
type BackupModel struct {
	cfg       *config.Config
	db        *index.DB
	logger    *slog.Logger
	backupOp  *operations.BackupOperation
	mode      backupMode
	list      list.Model
	progress  progress.Model
	dryRun    bool
	backupAll bool

	// Progress tracking
	currentStage   string
	filesComplete  int64
	bytesComplete  int64
	currentFile    string
	percentage     float64
	startTime      time.Time

	// Results
	result  *backup.BackupResult
	results []*backup.BackupResult
	err     error
}

type backupProgressMsg operations.OperationProgress

type backupCompleteMsg struct {
	result  *backup.BackupResult
	results []*backup.BackupResult
	err     error
}

// NewBackupModel creates a new backup runner model
func NewBackupModel(cfg *config.Config, db *index.DB, logger *slog.Logger) BackupModel {
	backupOp := operations.NewBackupOperation(cfg, db, logger)

	// Create list of backup sets
	items := make([]list.Item, len(cfg.BackupSets)+1)
	items[0] = backupSetSelectItem{name: "All Backup Sets", destination: "Run all configured backups"}
	for i, bs := range cfg.BackupSets {
		items[i+1] = backupSetSelectItem{
			name:        bs.Name,
			destination: bs.Destination,
		}
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "Select Backup Set"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	prog := progress.New(progress.WithDefaultGradient())

	return BackupModel{
		cfg:      cfg,
		db:       db,
		logger:   logger,
		backupOp: backupOp,
		mode:     backupModeSelect,
		list:     l,
		progress: prog,
		dryRun:   false,
	}
}

func (m BackupModel) Init() tea.Cmd {
	return nil
}

func (m BackupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width-4, msg.Height-10)
		m.progress.Width = msg.Width - 4

	case tea.KeyMsg:
		switch m.mode {
		case backupModeSelect:
			switch msg.String() {
			case "esc", "q":
				return m, func() tea.Msg { return NavigateMsg{Screen: "menu"} }
			case "d":
				m.dryRun = !m.dryRun
				return m, nil
			case "enter":
				// Start backup
				selected := m.list.Index()
				if selected == 0 {
					m.backupAll = true
				} else {
					m.backupAll = false
				}
				m.mode = backupModeRunning
				m.startTime = time.Now()
				return m, m.runBackup()
			}

		case backupModeComplete:
			switch msg.String() {
			case "esc", "q", "enter":
				// Reset and return to selection
				m.mode = backupModeSelect
				m.result = nil
				m.results = nil
				m.err = nil
				m.currentStage = ""
				m.filesComplete = 0
				m.bytesComplete = 0
				m.percentage = 0
				return m, nil
			}
		}

	case backupProgressMsg:
		m.currentStage = msg.Stage
		m.filesComplete = msg.FilesComplete
		m.bytesComplete = msg.BytesComplete
		m.currentFile = msg.CurrentFile
		m.percentage = msg.Percentage
		if m.percentage > 0 {
			return m, m.progress.SetPercent(m.percentage / 100.0)
		}

	case backupCompleteMsg:
		m.mode = backupModeComplete
		m.result = msg.result
		m.results = msg.results
		m.err = msg.err
		return m, nil

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	// Update current component
	if m.mode == backupModeSelect {
		m.list, cmd = m.list.Update(msg)
	}

	return m, cmd
}

func (m BackupModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Padding(1, 0)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(1, 0)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	valueStyle := lipgloss.NewStyle().Bold(true)

	switch m.mode {
	case backupModeSelect:
		s := titleStyle.Render("Backup Runner") + "\n\n"
		s += m.list.View() + "\n\n"

		if m.dryRun {
			s += successStyle.Render("Dry Run Mode: ON (no files will be copied)") + "\n\n"
		}

		s += helpStyle.Render("enter: run backup | d: toggle dry-run | esc/q: back to menu")
		return s

	case backupModeRunning:
		s := titleStyle.Render("Backup Running") + "\n\n"

		if m.backupAll {
			s += "Running all backup sets...\n\n"
		} else {
			selected := m.list.SelectedItem().(backupSetSelectItem)
			s += fmt.Sprintf("Backup Set: %s\n\n", selected.name)
		}

		if m.dryRun {
			s += successStyle.Render("DRY RUN MODE") + "\n\n"
		}

		s += fmt.Sprintf("%s %s\n", labelStyle.Render("Stage:"), valueStyle.Render(m.currentStage))
		s += fmt.Sprintf("%s %d\n", labelStyle.Render("Files processed:"), m.filesComplete)
		s += fmt.Sprintf("%s %s\n", labelStyle.Render("Bytes processed:"), humanize.Bytes(uint64(m.bytesComplete)))

		if m.currentFile != "" {
			truncated := m.currentFile
			if len(truncated) > 60 {
				truncated = "..." + truncated[len(truncated)-57:]
			}
			s += fmt.Sprintf("%s %s\n", labelStyle.Render("Current file:"), truncated)
		}

		s += "\n"

		if m.percentage > 0 {
			s += m.progress.View() + "\n"
			s += fmt.Sprintf("Progress: %.1f%%\n", m.percentage)
		} else {
			s += "Processing...\n"
		}

		elapsed := time.Since(m.startTime)
		s += fmt.Sprintf("\n%s %s\n", labelStyle.Render("Elapsed:"), elapsed.Round(time.Second))

		return s

	case backupModeComplete:
		s := titleStyle.Render("Backup Complete") + "\n\n"

		if m.err != nil {
			s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
		} else {
			if m.backupAll && m.results != nil {
				// Show summary for all backups
				var totalFiles, totalBytes int64
				for _, result := range m.results {
					s += fmt.Sprintf("%s:\n", valueStyle.Render(result.BackupSet))
					s += fmt.Sprintf("  %s %d\n", labelStyle.Render("Files copied:"), result.FilesCopied)
					s += fmt.Sprintf("  %s %s\n", labelStyle.Render("Bytes copied:"), humanize.Bytes(uint64(result.BytesCopied)))
					s += fmt.Sprintf("  %s %d\n", labelStyle.Render("Errors:"), result.Errors)
					s += fmt.Sprintf("  %s %s\n\n", labelStyle.Render("Duration:"), result.Duration.Round(time.Second))

					totalFiles += result.FilesCopied
					totalBytes += result.BytesCopied
				}

				s += fmt.Sprintf("%s\n", valueStyle.Render("Total:"))
				s += fmt.Sprintf("  %s %d\n", labelStyle.Render("Files:"), totalFiles)
				s += fmt.Sprintf("  %s %s\n\n", labelStyle.Render("Size:"), humanize.Bytes(uint64(totalBytes)))

			} else if m.result != nil {
				// Show summary for single backup
				s += fmt.Sprintf("%s %s\n", labelStyle.Render("Backup set:"), valueStyle.Render(m.result.BackupSet))
				s += fmt.Sprintf("%s %d\n", labelStyle.Render("Files copied:"), m.result.FilesCopied)
				s += fmt.Sprintf("%s %s\n", labelStyle.Render("Bytes copied:"), humanize.Bytes(uint64(m.result.BytesCopied)))
				s += fmt.Sprintf("%s %d\n", labelStyle.Render("Errors:"), m.result.Errors)
				s += fmt.Sprintf("%s %s\n\n", labelStyle.Render("Duration:"), m.result.Duration.Round(time.Second))
			}

			if m.dryRun {
				s += successStyle.Render("(Dry run - no files were actually copied)") + "\n\n"
			}
		}

		s += helpStyle.Render("enter/esc/q: back to backup selection")
		return s

	default:
		return ""
	}
}

func (m BackupModel) runBackup() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		progressCallback := func(progress operations.OperationProgress) {
			// Send progress update to TUI
			// Note: This is called from a goroutine, so we can't directly update the model
			// Instead, we would need to use a channel or tea.Cmd
		}

		var result *backup.BackupResult
		var results []*backup.BackupResult
		var err error

		if m.backupAll {
			results, err = m.backupOp.BackupAll(ctx, m.dryRun, progressCallback)
		} else {
			selected := m.list.SelectedItem().(backupSetSelectItem)
			result, err = m.backupOp.Run(ctx, operations.BackupOptions{
				BackupSetName: selected.name,
				DryRun:        m.dryRun,
				Progress:      progressCallback,
			})
		}

		return backupCompleteMsg{
			result:  result,
			results: results,
			err:     err,
		}
	}
}
