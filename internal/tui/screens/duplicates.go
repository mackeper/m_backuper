package screens

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/mackeper/m_backuper/internal/config"
	"github.com/mackeper/m_backuper/internal/duplicate"
	"github.com/mackeper/m_backuper/internal/index"
	"github.com/mackeper/m_backuper/internal/operations"
)

type duplicatesMode int

const (
	duplicatesModeList duplicatesMode = iota
	duplicatesModeDetail
	duplicatesModeDelete
)

type duplicateGroupItem struct {
	group index.DuplicateGroup
}

func (i duplicateGroupItem) Title() string {
	return fmt.Sprintf("%s (%d files, %s wasted)",
		i.group.Hash[:16]+"...",
		i.group.FileCount,
		humanize.Bytes(uint64(i.group.WastedSpace)))
}

func (i duplicateGroupItem) Description() string {
	return fmt.Sprintf("Size: %s per file", humanize.Bytes(uint64(i.group.FileSize)))
}

func (i duplicateGroupItem) FilterValue() string { return i.group.Hash }

// DuplicatesModel manages duplicate file browsing and cleaning
type DuplicatesModel struct {
	db        *index.DB
	cfg       *config.Config
	logger    *slog.Logger
	dupOp     *operations.DuplicateOperation
	mode      duplicatesMode
	list      list.Model
	groups    []index.DuplicateGroup
	selected  *index.DuplicateGroup
	cursor    int
	loading   bool
	err       error
	message   string
	strategy  duplicate.KeepStrategy
}

type duplicatesLoadedMsg struct {
	groups []index.DuplicateGroup
	err    error
}

type duplicatesDeletedMsg struct {
	message string
	err     error
}

// NewDuplicatesModel creates a new duplicates browser model
func NewDuplicatesModel(db *index.DB, cfg *config.Config, logger *slog.Logger) DuplicatesModel {
	dupOp := operations.NewDuplicateOperation(db, cfg, logger)

	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Duplicate Groups"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	return DuplicatesModel{
		db:       db,
		cfg:      cfg,
		logger:   logger,
		dupOp:    dupOp,
		mode:     duplicatesModeList,
		list:     l,
		loading:  true,
		strategy: duplicate.KeepFirst,
	}
}

func (m DuplicatesModel) Init() tea.Cmd {
	return m.loadDuplicates()
}

func (m DuplicatesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width-4, msg.Height-10)

	case tea.KeyMsg:
		switch m.mode {
		case duplicatesModeList:
			switch msg.String() {
			case "esc", "q":
				return m, func() tea.Msg { return NavigateMsg{Screen: "menu"} }
			case "r":
				m.loading = true
				return m, m.loadDuplicates()
			case "enter":
				if len(m.list.Items()) > 0 {
					selected := m.list.SelectedItem().(duplicateGroupItem)
					m.selected = &selected.group
					m.mode = duplicatesModeDetail
					m.cursor = 0
				}
			}

		case duplicatesModeDetail:
			switch msg.String() {
			case "esc", "q":
				m.mode = duplicatesModeList
				m.selected = nil
				return m, nil
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.selected != nil && m.cursor < len(m.selected.Files)-1 {
					m.cursor++
				}
			case "d":
				m.mode = duplicatesModeDelete
				return m, nil
			case "1":
				m.strategy = duplicate.KeepFirst
			case "2":
				m.strategy = duplicate.KeepOldest
			case "3":
				m.strategy = duplicate.KeepNewest
			case "4":
				m.strategy = duplicate.KeepShortest
			}

		case duplicatesModeDelete:
			switch msg.String() {
			case "y":
				return m, m.deleteDuplicates()
			case "n", "esc":
				m.mode = duplicatesModeDetail
				return m, nil
			}
		}

	case duplicatesLoadedMsg:
		m.loading = false
		m.groups = msg.groups
		m.err = msg.err
		if msg.err == nil {
			items := make([]list.Item, len(msg.groups))
			for i, g := range msg.groups {
				items[i] = duplicateGroupItem{group: g}
			}
			m.list.SetItems(items)
		}

	case duplicatesDeletedMsg:
		m.loading = false
		m.message = msg.message
		m.err = msg.err
		if msg.err == nil {
			m.mode = duplicatesModeList
			m.selected = nil
			// Reload duplicates
			return m, m.loadDuplicates()
		}
	}

	// Update current component
	if m.mode == duplicatesModeList && !m.loading {
		m.list, cmd = m.list.Update(msg)
	}

	return m, cmd
}

func (m DuplicatesModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Padding(1, 0)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(1, 0)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	normalStyle := lipgloss.NewStyle()

	switch m.mode {
	case duplicatesModeList:
		s := titleStyle.Render("Duplicate Browser") + "\n\n"

		if m.loading {
			s += "Loading duplicates...\n"
			return s
		}

		if m.err != nil {
			s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
		}

		if m.message != "" {
			s += successStyle.Render(m.message) + "\n\n"
		}

		if len(m.groups) == 0 {
			s += "No duplicates found\n\n"
		} else {
			s += m.list.View() + "\n\n"
		}

		s += helpStyle.Render("enter: view details | r: refresh | esc/q: back to menu")
		return s

	case duplicatesModeDetail:
		if m.selected == nil {
			return "No group selected"
		}

		s := titleStyle.Render("Duplicate Group Details") + "\n\n"
		s += fmt.Sprintf("Hash: %s\n", m.selected.Hash[:16]+"...")
		s += fmt.Sprintf("Files: %d | Size: %s | Wasted: %s\n\n",
			m.selected.FileCount,
			humanize.Bytes(uint64(m.selected.FileSize)),
			humanize.Bytes(uint64(m.selected.WastedSpace)))

		s += "Files:\n"
		for i, file := range m.selected.Files {
			cursor := "  "
			style := normalStyle
			if i == m.cursor {
				cursor = "> "
				style = selectedStyle
			}
			s += style.Render(fmt.Sprintf("%s%s", cursor, file.Path)) + "\n"
		}
		s += "\n"

		strategies := []struct {
			name     string
			strategy duplicate.KeepStrategy
		}{
			{"1: First (alphabetical)", duplicate.KeepFirst},
			{"2: Oldest (modification time)", duplicate.KeepOldest},
			{"3: Newest (modification time)", duplicate.KeepNewest},
			{"4: Shortest (path length)", duplicate.KeepShortest},
		}
		s += "Keep strategy: "
		for _, st := range strategies {
			if st.strategy == m.strategy {
				s += selectedStyle.Render(st.name) + " "
			} else {
				s += normalStyle.Render(st.name) + " "
			}
		}
		s += "\n\n"

		s += helpStyle.Render("↑/↓ or j/k: navigate | 1-4: select strategy | d: delete duplicates | esc/q: back")
		return s

	case duplicatesModeDelete:
		if m.selected == nil {
			return "No group selected"
		}

		s := titleStyle.Render("Delete Duplicates") + "\n\n"

		toDelete := duplicate.SelectFilesToDelete(*m.selected, m.strategy)
		s += fmt.Sprintf("Strategy: %s\n\n", getStrategyName(m.strategy))
		s += "Files to be deleted:\n"
		for _, path := range toDelete {
			s += errorStyle.Render("  - "+path) + "\n"
		}
		s += "\n"
		s += fmt.Sprintf("This will free %s of space.\n\n", humanize.Bytes(uint64(int64(len(toDelete))*m.selected.FileSize)))
		s += helpStyle.Render("y: confirm | n/esc: cancel")
		return s

	default:
		return ""
	}
}

func (m DuplicatesModel) loadDuplicates() tea.Cmd {
	return func() tea.Msg {
		groups, err := m.dupOp.FindDuplicates(context.Background(), operations.FindOptions{
			SortBy: "wasted",
		})
		return duplicatesLoadedMsg{groups: groups, err: err}
	}
}

func (m DuplicatesModel) deleteDuplicates() tea.Cmd {
	return func() tea.Msg {
		if m.selected == nil {
			return duplicatesDeletedMsg{err: fmt.Errorf("no group selected")}
		}

		toDelete := duplicate.SelectFilesToDelete(*m.selected, m.strategy)
		cleaner := duplicate.NewCleaner(m.db, m.logger)
		results, err := cleaner.DeleteFiles(toDelete, false)

		if err != nil {
			return duplicatesDeletedMsg{err: err}
		}

		// Count successful deletions
		var deleted int64
		var freed int64
		for _, result := range results {
			if result.Err == nil && result.Deleted {
				deleted++
				freed += result.Size
			}
		}

		message := fmt.Sprintf("Deleted %d files, freed %s", deleted, humanize.Bytes(uint64(freed)))
		return duplicatesDeletedMsg{message: message}
	}
}

func getStrategyName(strategy duplicate.KeepStrategy) string {
	switch strategy {
	case duplicate.KeepFirst:
		return "Keep First (alphabetical)"
	case duplicate.KeepOldest:
		return "Keep Oldest (modification time)"
	case duplicate.KeepNewest:
		return "Keep Newest (modification time)"
	case duplicate.KeepShortest:
		return "Keep Shortest (path length)"
	default:
		return "Unknown"
	}
}
