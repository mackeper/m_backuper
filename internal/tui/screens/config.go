package screens

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mackeper/m_backuper/internal/config"
	"github.com/mackeper/m_backuper/internal/configutil"
)

type configMode int

const (
	configModeList configMode = iota
	configModeAdd
	configModeEdit
	configModeDelete
)

// ConfigModel manages backup set configuration
type ConfigModel struct {
	cfg        *config.Config
	configPath string
	logger     *slog.Logger
	manager    *configutil.Manager
	mode       configMode
	list       list.Model
	inputs     []textinput.Model
	focusIdx   int
	err        error
	message    string
}

type backupSetItem struct {
	name        string
	destination string
	sources     []string
}

func (i backupSetItem) Title() string       { return i.name }
func (i backupSetItem) Description() string { return i.destination }
func (i backupSetItem) FilterValue() string { return i.name }

// NewConfigModel creates a new config manager model
func NewConfigModel(cfg *config.Config, configPath string, logger *slog.Logger) ConfigModel {
	manager, _ := configutil.NewManager(configPath)

	// Create list model
	items := []list.Item{}
	for _, bs := range cfg.BackupSets {
		items = append(items, backupSetItem{
			name:        bs.Name,
			destination: bs.Destination,
			sources:     bs.Sources,
		})
	}

	delegate := list.NewDefaultDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "Backup Sets"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return ConfigModel{
		cfg:        cfg,
		configPath: configPath,
		logger:     logger,
		manager:    manager,
		mode:       configModeList,
		list:       l,
	}
}

func (m ConfigModel) Init() tea.Cmd {
	return nil
}

func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width-4, msg.Height-10)

	case tea.KeyMsg:
		switch m.mode {
		case configModeList:
			switch msg.String() {
			case "esc", "q":
				return m, func() tea.Msg { return NavigateMsg{Screen: "menu"} }
			case "a":
				m.mode = configModeAdd
				m.initInputs("")
				return m, textinput.Blink
			case "d":
				if len(m.list.Items()) > 0 {
					m.mode = configModeDelete
				}
				return m, nil
			case "enter":
				if len(m.list.Items()) > 0 {
					selected := m.list.SelectedItem().(backupSetItem)
					m.mode = configModeEdit
					m.initInputs(selected.name)
					return m, textinput.Blink
				}
			}

		case configModeAdd, configModeEdit:
			switch msg.String() {
			case "esc":
				m.mode = configModeList
				m.message = ""
				m.err = nil
				return m, nil
			case "tab", "shift+tab":
				if msg.String() == "tab" {
					m.focusIdx++
				} else {
					m.focusIdx--
				}
				if m.focusIdx > len(m.inputs) {
					m.focusIdx = 0
				} else if m.focusIdx < 0 {
					m.focusIdx = len(m.inputs)
				}
				return m, m.updateInputFocus()
			case "enter":
				if m.focusIdx == len(m.inputs) {
					// Save button pressed
					return m, m.saveBackupSet()
				}
				m.focusIdx++
				if m.focusIdx > len(m.inputs) {
					m.focusIdx = 0
				}
				return m, m.updateInputFocus()
			}

		case configModeDelete:
			switch msg.String() {
			case "y":
				selected := m.list.SelectedItem().(backupSetItem)
				if err := m.manager.RemoveBackupSet(selected.name); err != nil {
					m.err = err
				} else {
					m.message = fmt.Sprintf("Deleted backup set: %s", selected.name)
					// Reload config
					m.cfg.BackupSets = removeBackupSet(m.cfg.BackupSets, selected.name)
					m.list.RemoveItem(m.list.Index())
				}
				m.mode = configModeList
				return m, nil
			case "n", "esc":
				m.mode = configModeList
				return m, nil
			}
		}
	}

	// Update current component
	switch m.mode {
	case configModeList:
		m.list, cmd = m.list.Update(msg)
	case configModeAdd, configModeEdit:
		if m.focusIdx < len(m.inputs) {
			m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
		}
	}

	return m, cmd
}

func (m ConfigModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Padding(1, 0)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(1, 0)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)

	switch m.mode {
	case configModeList:
		s := titleStyle.Render("Config Manager") + "\n\n"
		s += m.list.View() + "\n\n"

		if m.err != nil {
			s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n"
		} else if m.message != "" {
			s += successStyle.Render(m.message) + "\n"
		}

		s += helpStyle.Render("a: add | enter: edit | d: delete | esc/q: back to menu")
		return s

	case configModeAdd, configModeEdit:
		title := "Add Backup Set"
		if m.mode == configModeEdit {
			title = "Edit Backup Set"
		}
		s := titleStyle.Render(title) + "\n\n"

		for i, input := range m.inputs {
			s += input.View() + "\n"
			if i < len(m.inputs)-1 {
				s += "\n"
			}
		}

		s += "\n"
		saveBtn := "[ Save ]"
		if m.focusIdx == len(m.inputs) {
			saveBtn = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("226")).Render("[ Save ]")
		}
		s += saveBtn + "\n\n"

		if m.err != nil {
			s += errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n"
		}

		s += helpStyle.Render("tab/shift+tab: navigate | enter: next/save | esc: cancel")
		return s

	case configModeDelete:
		selected := m.list.SelectedItem().(backupSetItem)
		s := titleStyle.Render("Delete Backup Set") + "\n\n"
		s += fmt.Sprintf("Are you sure you want to delete '%s'?\n\n", selected.name)
		s += helpStyle.Render("y: yes | n: no | esc: cancel")
		return s

	default:
		return ""
	}
}

func (m *ConfigModel) initInputs(backupSetName string) {
	var bs *config.BackupSet
	if backupSetName != "" {
		bs = m.cfg.GetBackupSet(backupSetName)
	}

	m.inputs = make([]textinput.Model, 4)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "backup-name"
	m.inputs[0].Prompt = "Name: "
	m.inputs[0].CharLimit = 50
	if bs != nil {
		m.inputs[0].SetValue(bs.Name)
	}

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "/path/to/destination"
	m.inputs[1].Prompt = "Destination: "
	if bs != nil {
		m.inputs[1].SetValue(bs.Destination)
	}

	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "/path/to/source1,/path/to/source2"
	m.inputs[2].Prompt = "Sources (comma-separated): "
	if bs != nil {
		m.inputs[2].SetValue(strings.Join(bs.Sources, ","))
	}

	m.inputs[3] = textinput.New()
	m.inputs[3].Placeholder = "*.tmp,*.log (optional)"
	m.inputs[3].Prompt = "Excludes (comma-separated): "
	if bs != nil {
		m.inputs[3].SetValue(strings.Join(bs.Excludes, ","))
	}

	m.focusIdx = 0
	m.inputs[0].Focus()
}

func (m *ConfigModel) updateInputFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.focusIdx {
			cmds[i] = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *ConfigModel) saveBackupSet() tea.Cmd {
	return func() tea.Msg {
		name := strings.TrimSpace(m.inputs[0].Value())
		destination := strings.TrimSpace(m.inputs[1].Value())
		sourcesStr := strings.TrimSpace(m.inputs[2].Value())
		excludesStr := strings.TrimSpace(m.inputs[3].Value())

		if name == "" || destination == "" || sourcesStr == "" {
			return ErrorMsg{Err: fmt.Errorf("name, destination, and sources are required")}
		}

		sources := strings.Split(sourcesStr, ",")
		for i := range sources {
			sources[i] = strings.TrimSpace(sources[i])
		}

		var excludes []string
		if excludesStr != "" {
			excludes = strings.Split(excludesStr, ",")
			for i := range excludes {
				excludes[i] = strings.TrimSpace(excludes[i])
			}
		}

		input := configutil.BackupSetInput{
			Name:        name,
			Destination: destination,
			Sources:     sources,
			Excludes:    excludes,
		}

		var err error
		if m.mode == configModeAdd {
			err = m.manager.AddBackupSet(input)
		} else {
			err = m.manager.UpdateBackupSet(name, input)
		}

		if err != nil {
			m.err = err
			return ErrorMsg{Err: err}
		}

		// Reload config
		newCfg, err := config.Load(m.configPath)
		if err != nil {
			m.err = err
			return ErrorMsg{Err: err}
		}
		*m.cfg = *newCfg

		// Update list
		items := []list.Item{}
		for _, bs := range m.cfg.BackupSets {
			items = append(items, backupSetItem{
				name:        bs.Name,
				destination: bs.Destination,
				sources:     bs.Sources,
			})
		}
		m.list.SetItems(items)

		m.mode = configModeList
		m.message = fmt.Sprintf("Saved backup set: %s", name)
		m.err = nil

		return SuccessMsg{Message: m.message}
	}
}

func removeBackupSet(backups []config.BackupSet, name string) []config.BackupSet {
	result := []config.BackupSet{}
	for _, bs := range backups {
		if bs.Name != name {
			result = append(result, bs)
		}
	}
	return result
}
