package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shravan20/vecna/internal/config"
	"github.com/shravan20/vecna/internal/ssh"
)

type View int

const (
	ViewHome View = iota
	ViewAddHost
	ViewSSH
)

type Model struct {
	view         View
	width        int
	height       int
	keys         KeyMap
	cursor       int
	hosts        []config.Host
	inputs       []textinput.Model
	inputFocus   int
	err          error
	sshSession   *ssh.Session
	sshOutput    strings.Builder
	sshHost      *config.Host
	connecting   bool
	toast        string
	toastTimer   int
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
	if m.connecting && m.sshHost != nil {
		return m.connectSSH()
	}
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
		case ViewSSH:
			return m.updateSSH(msg)
		}

	case sshOutputMsg:
		if string(msg) != "" {
			m.sshOutput.WriteString(string(msg))
		}
		if m.sshSession != nil {
			return m, m.readSSHOutput()
		}
		return m, nil

	case sshErrorMsg:
		m.toast = string(msg)
		m.toastTimer = 50
		m.view = ViewHome
		if m.sshSession != nil {
			m.sshSession.Close()
			m.sshSession = nil
		}
		return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })

	case sshConnectedMsg:
		m.connecting = false
		m.sshSession = msg.session
		if m.width > 0 && m.height > 0 {
			m.sshSession.Resize(m.width, m.height-2)
		}
		return m, m.readSSHOutput()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.sshSession != nil {
			m.sshSession.Resize(msg.Width, msg.Height-2)
		}

	case tea.MouseMsg:
		// mouse support ready

	case tickMsg:
		if m.toastTimer > 0 {
			m.toastTimer--
			if m.toastTimer == 0 {
				m.toast = ""
			} else {
				return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
			}
		}
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

	case key.Matches(msg, m.keys.Connect), key.Matches(msg, m.keys.Enter):
		if len(m.hosts) > 0 && m.cursor < len(m.hosts) {
			h := m.hosts[m.cursor]
			m.sshHost = &h
			m.connecting = true
			m.view = ViewSSH
			m.sshOutput.Reset()
			return m, m.connectSSH()
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
			if m.toastTimer > 0 {
				return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
			}
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
	name := m.inputs[0].Value()
	hostname := m.inputs[1].Value()

	if name == "" || hostname == "" {
		m.toast = "Name and Host are required"
		m.toastTimer = 50
		return
	}

	port := 22
	if p := m.inputs[3].Value(); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}

	user := m.inputs[2].Value()
	if user == "" {
		user = "root"
	}

	h := config.Host{
		Name:     name,
		Hostname: hostname,
		User:     user,
		Port:     port,
	}

	config.AddHost(h)
	if err := config.Save(); err != nil {
		m.toast = fmt.Sprintf("Failed to save: %v", err)
		m.toastTimer = 50
		return
	}

	m.hosts = config.GetHosts()
	m.toast = fmt.Sprintf("Host '%s' added", name)
	m.toastTimer = 30
}

func (m Model) View() string {
	switch m.view {
	case ViewAddHost:
		return m.viewAddHost()
	case ViewSSH:
		return m.viewSSH()
	default:
		return m.viewHome()
	}
}

type sshOutputMsg string
type sshErrorMsg string
type sshConnectedMsg struct {
	session *ssh.Session
}
type tickMsg struct{}

func (m Model) connectSSH() tea.Cmd {
	return func() tea.Msg {
		if m.sshHost == nil {
			return sshErrorMsg("no host selected")
		}

		h := ssh.Host{
			Name:         m.sshHost.Name,
			Hostname:     m.sshHost.Hostname,
			User:         m.sshHost.User,
			Port:         m.sshHost.Port,
			IdentityFile: m.sshHost.IdentityFile,
		}

		session, err := ssh.Connect(h)
		if err != nil {
			return sshErrorMsg(err.Error())
		}

		return sshConnectedMsg{session: session}
	}
}

func (m Model) readSSHOutput() tea.Cmd {
	return func() tea.Msg {
		if m.sshSession == nil {
			return nil
		}

		buf := make([]byte, 4096)
		n, err := m.sshSession.Read(buf)
		if err == io.EOF {
			return sshErrorMsg("connection closed")
		}
		if err != nil {
			return sshErrorMsg(err.Error())
		}

		if n > 0 {
			return sshOutputMsg(string(buf[:n]))
		}
		return nil
	}
}

func (m Model) updateSSH(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		if m.sshSession != nil {
			m.sshSession.Close()
			m.sshSession = nil
		}
		m.view = ViewHome
		m.sshHost = nil
		m.sshOutput.Reset()
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		if m.sshSession != nil {
			m.sshSession.Close()
			m.sshSession = nil
		}
		return m, tea.Quit
	}

	if m.sshSession != nil {
		var data []byte
		if msg.Type == tea.KeyRunes {
			data = []byte(msg.String())
		} else {
			switch msg.String() {
			case "enter":
				data = []byte("\r")
			case "backspace":
				data = []byte("\x7f")
			case "tab":
				data = []byte("\t")
			case "space":
				data = []byte(" ")
			case "up":
				data = []byte("\x1b[A")
			case "down":
				data = []byte("\x1b[B")
			case "right":
				data = []byte("\x1b[C")
			case "left":
				data = []byte("\x1b[D")
			case "ctrl+c":
				data = []byte("\x03")
			case "ctrl+d":
				data = []byte("\x04")
			case "esc":
				data = []byte("\x1b")
			default:
				data = []byte(msg.String())
			}
		}
		if len(data) > 0 {
			m.sshSession.Write(data)
			return m, m.readSSHOutput()
		}
	}

	return m, m.readSSHOutput()
}

func Run() error {
	m := New()
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
