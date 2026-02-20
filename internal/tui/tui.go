package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shravan20/vecna/internal/config"
)

type View int

const (
	ViewHome View = iota
	ViewAddHost
)

type Model struct {
	view       View
	width      int
	height     int
	keys       KeyMap
	cursor     int
	hosts      []config.Host
	inputs     []textinput.Model
	inputFocus int
	err        error
}

func New() Model {
	return Model{
		view:  ViewHome,
		keys:  DefaultKeyMap(),
		hosts: config.GetHosts(),
	}
}

func (m *Model) initAddHostInputs() {
	m.inputs = make([]textinput.Model, 4)

	for i := range m.inputs {
		m.inputs[i] = textinput.New()
		m.inputs[i].Prompt = ""
		m.inputs[i].CharLimit = 256
		m.inputs[i].Width = 40
	}

	m.inputs[0].Placeholder = "e.g. prod-server"
	m.inputs[1].Placeholder = "192.168.1.100 or host.example.com"
	m.inputs[2].Placeholder = "root"
	m.inputs[3].Placeholder = "22"
	m.inputs[3].CharLimit = 5

	m.inputs[0].Focus()
	m.inputFocus = 0
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.view {
		case ViewHome:
			return m.updateHome(msg)
		case ViewAddHost:
			return m.updateAddHost(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.MouseMsg:
		// mouse support ready
	}

	return m, nil
}

func (m Model) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.hosts)-1 {
			m.cursor++
		}

	case key.Matches(msg, m.keys.Add):
		m.view = ViewAddHost
		m.initAddHostInputs()
		return m, m.inputs[0].Focus()

	case key.Matches(msg, m.keys.Delete):
		if len(m.hosts) > 0 && m.cursor < len(m.hosts) {
			config.RemoveHost(m.cursor)
			m.hosts = config.GetHosts()
			if m.cursor >= len(m.hosts) && m.cursor > 0 {
				m.cursor--
			}
		}
	}

	return m, nil
}

func (m Model) updateAddHost(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle special keys first
	switch msg.String() {
	case "esc":
		m.view = ViewHome
		m.inputs = nil
		return m, nil

	case "enter":
		if m.inputFocus < len(m.inputs)-1 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus++
			return m, m.inputs[m.inputFocus].Focus()
		}
		if m.inputs[0].Value() != "" && m.inputs[1].Value() != "" {
			m.saveHost()
			m.view = ViewHome
			m.inputs = nil
		}
		return m, nil

	case "tab":
		if m.inputFocus < len(m.inputs)-1 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus++
			return m, m.inputs[m.inputFocus].Focus()
		}
		return m, nil

	case "shift+tab":
		if m.inputFocus > 0 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus--
			return m, m.inputs[m.inputFocus].Focus()
		}
		return m, nil

	case "up":
		if m.inputFocus > 0 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus--
			return m, m.inputs[m.inputFocus].Focus()
		}
		return m, nil

	case "down":
		if m.inputFocus < len(m.inputs)-1 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus++
			return m, m.inputs[m.inputFocus].Focus()
		}
		return m, nil
	}

	// Let textinput handle all other keys (including 'k', 'j', etc.)
	var cmd tea.Cmd
	m.inputs[m.inputFocus], cmd = m.inputs[m.inputFocus].Update(msg)
	return m, cmd
}

func (m *Model) saveHost() {
	port := 22
	if p := m.inputs[3].Value(); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}

	user := m.inputs[2].Value()
	if user == "" {
		user = "root"
	}

	h := config.Host{
		Name:     m.inputs[0].Value(),
		Hostname: m.inputs[1].Value(),
		User:     user,
		Port:     port,
	}

	config.AddHost(h)
	config.Save()
	m.hosts = config.GetHosts()
}

func (m Model) View() string {
	switch m.view {
	case ViewAddHost:
		return m.viewAddHost()
	default:
		return m.viewHome()
	}
}

func Run() error {
	p := tea.NewProgram(
		New(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
