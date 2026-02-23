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
	"github.com/shravan20/vecna/internal/sftp"
	"github.com/shravan20/vecna/internal/ssh"
)

type View int

const (
	ViewHome View = iota
	ViewAddHost
	ViewSSH
	ViewPortForward
	ViewRunCommand
	ViewFileTransfer
)

type tabKind int

const (
	tabKindHome tabKind = iota
	tabKindSSH
)

type tab struct {
	Id         int
	Kind       tabKind
	Title      string
	Host       config.Host // host for SSH tabs (by value so it outlives the add-tab call)
	Session    *ssh.Session
	Output     string
	Connecting bool
	Log        []string
}

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
	sshHost      *config.Host
	toast        string
	toastTimer   int
	toastSuccess bool
	showPassword bool
	animFrame    int
	editingHostIndex int

	// Tabs: tab 0 is always Hosts; rest are SSH sessions
	tabs            []tab
	currentTabIndex int
	nextTabId       int

	// Port forward view
	portForwardInputs  []textinput.Model
	portForwardFocus   int
	portForwardCursor  int
	activeForwards     []activeForward
	portForwardStarting bool
	portForwardNextID  int

	// Home filter
	hostFilter textinput.Model
	homeFocus  int // 0 = list, 1 = filter input

	// Run command view
	runCommandCursor  int
	runCommandOutput  string
	runCommandRunning bool

	// File transfer view
	transferInputs  []textinput.Model
	transferFocus   int
	transferOutput  string
	transferRunning bool
}

type activeForward struct {
	id         int
	label      string
	localAddr  string
	remoteAddr string
	stop       func() // closes listener and SSH client
}

// hostEntry pairs a host with its index in config (for delete/edit after filtering).
type hostEntry struct {
	Host  config.Host
	Index int
}

// filteredHostEntries returns hosts matching the filter query (name or tag, case-insensitive).
func (m Model) filteredHostEntries() []hostEntry {
	query := strings.TrimSpace(m.hostFilter.Value())
	if query == "" {
		out := make([]hostEntry, len(m.hosts))
		for i, h := range m.hosts {
			out[i] = hostEntry{Host: h, Index: i}
		}
		return out
	}
	q := strings.ToLower(query)
	var out []hostEntry
	for i, h := range m.hosts {
		if strings.Contains(strings.ToLower(h.Name), q) {
			out = append(out, hostEntry{Host: h, Index: i})
			continue
		}
		for _, t := range h.Tags {
			if strings.Contains(strings.ToLower(t), q) {
				out = append(out, hostEntry{Host: h, Index: i})
				break
			}
		}
	}
	return out
}

func New() Model {
	f := textinput.New()
	f.Placeholder = "Filter by name or tag..."
	f.CharLimit = 80
	f.Width = 30
	return Model{
		view:             ViewHome,
		keys:             DefaultKeyMap(),
		hosts:            config.GetHosts(),
		hostFilter:       f,
		homeFocus:        0,
		tabs:             []tab{{Id: 0, Kind: tabKindHome, Title: "Hosts"}},
		currentTabIndex:  0,
		nextTabId:        1,
	}
}

func (m *Model) initAddHostInputs() {
	m.inputs = make([]textinput.Model, 7)

	for i := range m.inputs {
		m.inputs[i] = textinput.New()
		m.inputs[i].Prompt = ""
		m.inputs[i].CharLimit = 256
		m.inputs[i].Width = 40
		if i == 4 {
			m.inputs[i].EchoMode = textinput.EchoPassword
			m.inputs[i].EchoCharacter = '•'
		}
	}

	m.inputs[0].Placeholder = "e.g. prod-server"
	m.inputs[1].Placeholder = "192.168.1.100 or host.example.com"
	m.inputs[2].Placeholder = "root"
	m.inputs[3].Placeholder = "22"
	m.inputs[3].CharLimit = 5
	m.inputs[4].Placeholder = "password (for first-time key setup, optional)"
	m.inputs[5].Placeholder = "y/n (auto-generate SSH key?)"
	m.inputs[6].Placeholder = "optional: name of jump/bastion host"

	m.inputs[0].Focus()
	m.inputFocus = 0
}

// initEditHostInputs fills the add-host form with the given host for editing.
func (m *Model) initEditHostInputs(h config.Host, configIndex int) {
	m.initAddHostInputs()
	m.inputs[0].SetValue(h.Name)
	m.inputs[1].SetValue(h.Hostname)
	m.inputs[2].SetValue(h.User)
	if h.Port == 0 {
		m.inputs[3].SetValue("22")
	} else {
		m.inputs[3].SetValue(fmt.Sprintf("%d", h.Port))
	}
	// Password: leave empty for edit (user can type to change)
	m.inputs[5].SetValue("n")
	if h.AutoGenerateKey {
		m.inputs[5].SetValue("y")
	}
	m.inputs[6].SetValue(h.ProxyJump)
	m.editingHostIndex = configIndex
}

func (m *Model) initPortForwardInputs() {
	m.portForwardInputs = make([]textinput.Model, 3)
	for i := range m.portForwardInputs {
		m.portForwardInputs[i] = textinput.New()
		m.portForwardInputs[i].Prompt = ""
		m.portForwardInputs[i].CharLimit = 64
		m.portForwardInputs[i].Width = 24
	}
	m.portForwardInputs[0].Placeholder = "e.g. 8080"
	m.portForwardInputs[1].Placeholder = "e.g. localhost or 127.0.0.1"
	m.portForwardInputs[2].Placeholder = "e.g. 5432"
	m.portForwardInputs[0].Focus()
	m.portForwardFocus = 0
	m.portForwardCursor = 0
}

// startPortForwardCmd runs blocking SSH dial + forward; call from a tea.Cmd.
func startPortForwardCmd(host config.Host, localPort, remoteHost, remotePort string, nextID int) tea.Msg {
	localAddr := "127.0.0.1:" + localPort
	remoteAddr := remoteHost + ":" + remotePort

	sshHost := ssh.Host{
		Name:         host.Name,
		Hostname:     host.Hostname,
		User:         host.User,
		Port:         host.Port,
		IdentityFile: host.IdentityFile,
	}
	password := ""
	if host.Password != "" {
		decrypted, err := config.DecryptPassword(host.Password)
		if err != nil {
			return portForwardErrorMsg(fmt.Sprintf("decrypt password: %v", err))
		}
		password = decrypted
	}
	skipKey := !host.KeyDeployed && host.IdentityFile != ""

	client, err := ssh.DialClient(sshHost, password, skipKey)
	if err != nil {
		return portForwardErrorMsg(err.Error())
	}
	stopForward, err := ssh.StartPortForward(client, localAddr, remoteAddr)
	if err != nil {
		client.Close()
		return portForwardErrorMsg(err.Error())
	}
	stopAll := func() {
		stopForward()
		client.Close()
	}
	return portForwardStartedMsg{
		fwd: activeForward{
			id:         nextID,
			label:      fmt.Sprintf("%s → %s", localAddr, remoteAddr),
			localAddr:  localAddr,
			remoteAddr: remoteAddr,
			stop:       stopAll,
		},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return animTickMsg{} })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.view {
		case ViewHome:
			// Tab switching: 1-9, Ctrl+Left, Ctrl+Right
			switch msg.String() {
			case "1":
				m.currentTabIndex = 0
				return m, nil
			case "2", "3", "4", "5", "6", "7", "8", "9":
				idx := int(msg.String()[0] - '1')
				if idx+1 < len(m.tabs) {
					m.currentTabIndex = idx + 1
					if t := &m.tabs[m.currentTabIndex]; t.Session != nil {
						return m, m.readSSHOutput(t.Id, t.Session)
					}
				}
				return m, nil
			case "ctrl+right":
				if len(m.tabs) > 1 && m.currentTabIndex < len(m.tabs)-1 {
					m.currentTabIndex++
					if t := &m.tabs[m.currentTabIndex]; t.Session != nil {
						return m, m.readSSHOutput(t.Id, t.Session)
					}
				}
				return m, nil
			case "ctrl+left":
				if m.currentTabIndex > 0 {
					m.currentTabIndex--
					if t := &m.tabs[m.currentTabIndex]; t.Session != nil {
						return m, m.readSSHOutput(t.Id, t.Session)
					}
				}
				return m, nil
			}
			if m.currentTabIndex == 0 {
				return m.updateHome(msg)
			}
			return m.updateSSH(msg)
		case ViewAddHost:
			return m.updateAddHost(msg)
		case ViewSSH:
			return m.updateSSH(msg)
		case ViewPortForward:
			return m.updatePortForward(msg)
		case ViewRunCommand:
			return m.updateRunCommand(msg)
		case ViewFileTransfer:
			return m.updateFileTransfer(msg)
		}

	case sshOutputMsg:
		for i := range m.tabs {
			if m.tabs[i].Id == msg.TabId && msg.Data != "" {
				m.tabs[i].Output += msg.Data
				break
			}
		}
		var cmd tea.Cmd
		for i := range m.tabs {
			if m.tabs[i].Id == msg.TabId && m.tabs[i].Session != nil {
				cmd = m.readSSHOutput(msg.TabId, m.tabs[i].Session)
				break
			}
		}
		return m, cmd

	case sshErrorMsg:
		m.toast = msg.Msg
		m.toastSuccess = false
		m.toastTimer = 120
		if msg.TabId == 0 {
			// Error during connect: remove the tab that was connecting
			for i := range m.tabs {
				if m.tabs[i].Connecting {
					if m.tabs[i].Session != nil {
						m.tabs[i].Session.Close()
					}
					m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)
					if m.currentTabIndex >= len(m.tabs) {
						m.currentTabIndex = len(m.tabs) - 1
					}
					if m.currentTabIndex < 0 {
						m.currentTabIndex = 0
					}
					break
				}
			}
		} else {
			// Error for a specific tab (e.g. connection closed): close that tab
			for i := range m.tabs {
				if m.tabs[i].Id == msg.TabId {
					if m.tabs[i].Session != nil {
						m.tabs[i].Session.Close()
					}
					m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)
					if m.currentTabIndex >= len(m.tabs) {
						m.currentTabIndex = len(m.tabs) - 1
					}
					if m.currentTabIndex < 0 {
						m.currentTabIndex = 0
					}
					break
				}
			}
		}
		return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })

	case sshConnectedMsg:
		for i := range m.tabs {
			if m.tabs[i].Connecting {
				m.tabs[i].Connecting = false
				m.tabs[i].Session = msg.session
				m.tabs[i].Log = append(m.tabs[i].Log, "✓ Connected")
				m.tabs[i].Output = ""
				cols, rows := m.width, m.height-2
				if cols < 80 {
					cols = 80
				}
				if rows < 24 {
					rows = 24
				}
				if m.width > 0 && m.height > 0 {
					msg.session.Resize(cols, rows)
				}
				return m, tea.Batch(m.readSSHOutput(m.tabs[i].Id, msg.session), tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return sshTickMsg{} }))
			}
		}
		return m, nil

	case sshTickMsg:
		if m.view == ViewHome && m.currentTabIndex > 0 && m.currentTabIndex < len(m.tabs) && m.tabs[m.currentTabIndex].Session != nil {
			return m, tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return sshTickMsg{} })
		}

	case portForwardStartedMsg:
		m.portForwardStarting = false
		m.activeForwards = append(m.activeForwards, msg.fwd)
		m.toast = fmt.Sprintf("Forward %s → %s", msg.fwd.localAddr, msg.fwd.remoteAddr)
		m.toastSuccess = true
		m.toastTimer = 30
		return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })

	case portForwardErrorMsg:
		m.portForwardStarting = false
		m.toast = string(msg)
		m.toastSuccess = false
		m.toastTimer = 50
		return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })

	case runCommandResultMsg:
		m.runCommandRunning = false
		if msg.err != nil {
			m.runCommandOutput = msg.output + "\n" + msg.err.Error()
		} else {
			m.runCommandOutput = msg.output
		}
		if m.runCommandOutput == "" {
			m.runCommandOutput = "(no output)"
		}
		return m, nil

	case transferResultMsg:
		m.transferRunning = false
		if msg.err != nil {
			m.transferOutput = msg.output + "\n" + msg.err.Error()
		} else {
			m.transferOutput = msg.output
		}
		if m.transferOutput == "" {
			m.transferOutput = "Done."
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.currentTabIndex >= 0 && m.currentTabIndex < len(m.tabs) && m.tabs[m.currentTabIndex].Session != nil {
			m.tabs[m.currentTabIndex].Session.Resize(msg.Width, msg.Height-2)
		}

	case tea.MouseMsg:
		// mouse support ready

	case tickMsg:
		if m.toastTimer > 0 {
			m.toastTimer--
			if m.toastTimer == 0 {
				m.toast = ""
				m.toastSuccess = false
			} else {
				return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
			}
		}

	case animTickMsg:
		connecting := false
		for i := range m.tabs {
			if m.tabs[i].Connecting {
				connecting = true
				break
			}
		}
		needsAnim := connecting || m.width == 0
		if needsAnim {
			m.animFrame++
			return m, tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return animTickMsg{} })
		}
	}

	return m, nil
}

func (m Model) updateHome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	entries := m.filteredHostEntries()

	// Filter bar focused: only filter input and esc
	if m.homeFocus == 1 {
		switch msg.String() {
		case "esc":
			m.hostFilter.Blur()
			m.homeFocus = 0
			return m, nil
		case "down", "enter", "tab":
			m.hostFilter.Blur()
			m.homeFocus = 0
			return m, nil
		}
		var cmd tea.Cmd
		m.hostFilter, cmd = m.hostFilter.Update(msg)
		// After filter change, clamp cursor
		if m.cursor >= len(entries) && len(entries) > 0 {
			m.cursor = len(entries) - 1
		}
		if len(entries) == 0 {
			m.cursor = 0
		}
		return m, cmd
	}

	// List focused
	switch {
	case msg.String() == "/":
		m.homeFocus = 1
		return m, m.hostFilter.Focus()

	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(entries)-1 {
			m.cursor++
		}

	case key.Matches(msg, m.keys.Add):
		m.view = ViewAddHost
		m.editingHostIndex = -1
		m.initAddHostInputs()
		return m, m.inputs[0].Focus()

	case key.Matches(msg, m.keys.Edit):
		if len(entries) > 0 && m.cursor < len(entries) {
			e := entries[m.cursor]
			m.view = ViewAddHost
			m.initEditHostInputs(e.Host, e.Index)
			return m, m.inputs[0].Focus()
		}

	case key.Matches(msg, m.keys.Delete):
		if len(entries) > 0 && m.cursor < len(entries) {
			realIndex := entries[m.cursor].Index
			config.RemoveHost(realIndex)
			m.hosts = config.GetHosts()
			if m.cursor >= len(entries)-1 && m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		}

	case key.Matches(msg, m.keys.Connect), key.Matches(msg, m.keys.Enter):
		if len(entries) > 0 && m.cursor < len(entries) {
			h := entries[m.cursor].Host
			m.nextTabId++
			m.tabs = append(m.tabs, tab{
				Id:         m.nextTabId,
				Kind:       tabKindSSH,
				Title:      h.Name,
				Host:       h,
				Connecting: true,
				Log:        []string{"→ Connecting..."},
			})
			m.currentTabIndex = len(m.tabs) - 1
			m.animFrame = 0
			return m, tea.Batch(
				m.connectSSH(h),
				tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return animTickMsg{} }),
			)
		}

	case key.Matches(msg, m.keys.Forward):
		if len(entries) > 0 && m.cursor < len(entries) {
			h := entries[m.cursor].Host
			m.sshHost = &h
			m.view = ViewPortForward
			m.initPortForwardInputs()
			return m, m.portForwardInputs[0].Focus()
		}

	case key.Matches(msg, m.keys.RunCommand):
		if len(entries) > 0 && m.cursor < len(entries) {
			h := entries[m.cursor].Host
			m.sshHost = &h
			m.view = ViewRunCommand
			m.runCommandCursor = 0
			m.runCommandOutput = ""
			return m, nil
		}

	case key.Matches(msg, m.keys.Transfer):
		if len(entries) > 0 && m.cursor < len(entries) {
			h := entries[m.cursor].Host
			m.sshHost = &h
			m.view = ViewFileTransfer
			m.initTransferInputs()
			m.transferOutput = ""
			return m, m.transferInputs[0].Focus()
		}

	case key.Matches(msg, m.keys.SFTP):
		if len(entries) > 0 && m.cursor < len(entries) {
			h := entries[m.cursor].Host
			conn := sftp.HostConnection{
				User:         h.User,
				Hostname:     h.Hostname,
				Port:         h.Port,
				IdentityFile: h.IdentityFile,
			}
			cmdStr, err := sftp.RunInNewTerminal(conn)
			if err != nil {
				m.toast = "Run: " + cmdStr
				m.toastSuccess = false
				m.toastTimer = 80
			} else {
				m.toast = "SFTP opened in new terminal"
				m.toastSuccess = true
				m.toastTimer = 25
			}
			return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
		}
	}

	return m, nil
}

func (m Model) updateAddHost(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+p":
		m.showPassword = !m.showPassword
		if m.showPassword {
			m.inputs[4].EchoMode = textinput.EchoNormal
		} else {
			m.inputs[4].EchoMode = textinput.EchoPassword
			m.inputs[4].EchoCharacter = '•'
		}
		return m, nil

	case "esc":
		m.view = ViewHome
		m.inputs = nil
		m.showPassword = false
		m.editingHostIndex = -1
		return m, nil

	case "enter":
		if m.inputFocus < len(m.inputs)-1 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus++
			return m, m.inputs[m.inputFocus].Focus()
		}
		if m.inputFocus < len(m.inputs)-1 {
			m.inputs[m.inputFocus].Blur()
			m.inputFocus++
			return m, m.inputs[m.inputFocus].Focus()
		}
		if m.inputs[0].Value() != "" && m.inputs[1].Value() != "" {
			m.saveHost()
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

func (m Model) updatePortForward(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Esc: back to home
	if key.Matches(msg, m.keys.Back) {
		m.view = ViewHome
		return m, nil
	}

	// Focus 3 = list of active forwards
	if m.portForwardFocus == 3 {
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.portForwardCursor > 0 {
				m.portForwardCursor--
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.portForwardCursor < len(m.activeForwards)-1 {
				m.portForwardCursor++
			}
			return m, nil
		case key.Matches(msg, m.keys.Delete):
			if len(m.activeForwards) > 0 && m.portForwardCursor >= 0 && m.portForwardCursor < len(m.activeForwards) {
				fwd := m.activeForwards[m.portForwardCursor]
				fwd.stop()
				m.activeForwards = append(m.activeForwards[:m.portForwardCursor], m.activeForwards[m.portForwardCursor+1:]...)
				if m.portForwardCursor >= len(m.activeForwards) && m.portForwardCursor > 0 {
					m.portForwardCursor--
				}
				m.toast = "Forward stopped"
				m.toastSuccess = true
				m.toastTimer = 20
			}
			return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
		case msg.String() == "tab", msg.String() == "enter":
			m.portForwardFocus = 0
			return m, m.portForwardInputs[0].Focus()
		}
		return m, nil
	}

	// Form fields 0..2
	switch msg.String() {
	case "tab", "down", "enter":
		if m.portForwardFocus < 2 {
			m.portForwardInputs[m.portForwardFocus].Blur()
			m.portForwardFocus++
			return m, m.portForwardInputs[m.portForwardFocus].Focus()
		}
		if m.portForwardFocus == 2 {
			// Enter on last field: start forward
			localPort := strings.TrimSpace(m.portForwardInputs[0].Value())
			remoteHost := strings.TrimSpace(m.portForwardInputs[1].Value())
			remotePort := strings.TrimSpace(m.portForwardInputs[2].Value())
			if localPort != "" && remoteHost != "" && remotePort != "" && !m.portForwardStarting && len(m.hosts) > 0 && m.cursor < len(m.hosts) {
				host := m.hosts[m.cursor]
				m.portForwardNextID++
				m.portForwardStarting = true
				nextID := m.portForwardNextID
				return m, func() tea.Msg {
					return startPortForwardCmd(host, localPort, remoteHost, remotePort, nextID)
				}
			}
			// Move to list
			m.portForwardInputs[2].Blur()
			m.portForwardFocus = 3
			return m, nil
		}
	case "shift+tab", "up":
		if m.portForwardFocus > 0 {
			m.portForwardInputs[m.portForwardFocus].Blur()
			m.portForwardFocus--
			return m, m.portForwardInputs[m.portForwardFocus].Focus()
		}
		if m.portForwardFocus == 0 {
			m.portForwardFocus = 3
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.portForwardInputs[m.portForwardFocus], cmd = m.portForwardInputs[m.portForwardFocus].Update(msg)
	return m, cmd
}

func (m Model) updateRunCommand(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	commands := config.GetCommands()

	if m.runCommandOutput != "" {
		// Showing output: any key (e.g. esc) goes back to home
		if key.Matches(msg, m.keys.Back) || msg.String() == "q" {
			m.view = ViewHome
			m.runCommandOutput = ""
			return m, nil
		}
		return m, nil
	}

	if key.Matches(msg, m.keys.Back) {
		m.view = ViewHome
		m.sshHost = nil
		return m, nil
	}

	if m.runCommandRunning {
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Up):
		if m.runCommandCursor > 0 {
			m.runCommandCursor--
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.runCommandCursor < len(commands)-1 {
			m.runCommandCursor++
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		if len(commands) > 0 && m.runCommandCursor < len(commands) && m.sshHost != nil {
			cmd := commands[m.runCommandCursor]
			m.runCommandRunning = true
			return m, func() tea.Msg {
				return runCommandCmd(*m.sshHost, cmd.Command)
			}
		}
	}

	return m, nil
}

func (m Model) updateFileTransfer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		m.view = ViewHome
		m.sshHost = nil
		m.transferInputs = nil
		return m, nil
	}

	if m.transferOutput != "" {
		if key.Matches(msg, m.keys.Back) || msg.String() == "q" {
			m.view = ViewHome
			m.transferOutput = ""
			m.sshHost = nil
			return m, nil
		}
		return m, nil
	}

	if m.transferRunning {
		return m, nil
	}

	switch msg.String() {
	case "tab", "down", "enter":
		if m.transferFocus < 2 {
			m.transferInputs[m.transferFocus].Blur()
			m.transferFocus++
			return m, m.transferInputs[m.transferFocus].Focus()
		}
		// Last field: run transfer
		dir := strings.TrimSpace(strings.ToLower(m.transferInputs[0].Value()))
		local := strings.TrimSpace(m.transferInputs[1].Value())
		remote := strings.TrimSpace(m.transferInputs[2].Value())
		if (dir == "push" || dir == "pull") && local != "" && remote != "" && m.sshHost != nil {
			m.transferRunning = true
			conn := sftp.HostConnection{
				User:         m.sshHost.User,
				Hostname:     m.sshHost.Hostname,
				Port:         m.sshHost.Port,
				IdentityFile: m.sshHost.IdentityFile,
			}
			return m, func() tea.Msg {
				return transferCmd(conn, dir, local, remote)
			}
		}
	case "shift+tab", "up":
		if m.transferFocus > 0 {
			m.transferInputs[m.transferFocus].Blur()
			m.transferFocus--
			return m, m.transferInputs[m.transferFocus].Focus()
		}
	}

	var cmd tea.Cmd
	m.transferInputs[m.transferFocus], cmd = m.transferInputs[m.transferFocus].Update(msg)
	return m, cmd
}

func (m *Model) saveHost() {
	name := m.inputs[0].Value()
	hostname := m.inputs[1].Value()

	if name == "" || hostname == "" {
		m.toast = "Name and Host are required"
		m.toastSuccess = false
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

	isEdit := m.editingHostIndex >= 0
	if isEdit {
		// Update existing host: only change name, hostname, user, port; keep auth fields
		existing := config.GetHosts()
		if m.editingHostIndex >= len(existing) {
			m.toast = "Host no longer exists"
			m.toastSuccess = false
			m.toastTimer = 50
			return
		}
		cur := existing[m.editingHostIndex]
		h := config.Host{
			Name:            name,
			Hostname:        hostname,
			User:            user,
			Port:            port,
			IdentityFile:    cur.IdentityFile,
			Password:        cur.Password,
			KeyDeployed:     cur.KeyDeployed,
			AutoGenerateKey: cur.AutoGenerateKey,
			Tags:            cur.Tags,
			ProxyJump:       strings.TrimSpace(m.inputs[6].Value()),
		}
		if newPass := m.inputs[4].Value(); newPass != "" {
			encrypted, err := config.EncryptPassword(newPass)
			if err != nil {
				m.toast = fmt.Sprintf("Failed to encrypt password: %v", err)
				m.toastSuccess = false
				m.toastTimer = 50
				return
			}
			h.Password = encrypted
		}
		config.UpdateHost(m.editingHostIndex, h)
		m.hosts = config.GetHosts()
		m.toast = fmt.Sprintf("Host '%s' updated", name)
		m.toastSuccess = true
		m.toastTimer = 30
		m.view = ViewHome
		m.inputs = nil
		m.showPassword = false
		m.editingHostIndex = -1
		return
	}

	// Add new host
	password := m.inputs[4].Value()
	autoGenKey := strings.ToLower(m.inputs[5].Value()) == "y" || strings.ToLower(m.inputs[5].Value()) == "yes"

	var identityFile string
	if autoGenKey {
		privatePath, _, err := ssh.GenerateKeyPair(name)
		if err != nil {
			m.toast = fmt.Sprintf("Failed to generate key: %v", err)
			m.toastSuccess = false
			m.toastTimer = 50
			return
		}
		identityFile = privatePath
	}

	sshHost := ssh.Host{
		Name:         name,
		Hostname:     hostname,
		User:         user,
		Port:         port,
		IdentityFile: identityFile,
	}

	if password == "" && identityFile == "" {
		m.toast = "Password or existing key required for validation"
		m.toastSuccess = false
		m.toastTimer = 50
		return
	}

	m.toast = "→ Validating connection..."
	m.toastSuccess = false
	m.toastTimer = 100

	if err := ssh.ValidateConnection(sshHost, password); err != nil {
		m.toast = fmt.Sprintf("Validation failed: %v", err)
		m.toastSuccess = false
		m.toastTimer = 50
		return
	}

	var encryptedPassword string
	if password != "" {
		var err error
		encryptedPassword, err = config.EncryptPassword(password)
		if err != nil {
			m.toast = fmt.Sprintf("Failed to encrypt password: %v", err)
			m.toastSuccess = false
			m.toastTimer = 50
			return
		}
	}

	keyDeployed := false
	if autoGenKey && password != "" && identityFile != "" {
		m.toast = "→ Deploying SSH key..."
		m.toastSuccess = false
		m.toastTimer = 100
		publicKeyPath := identityFile + ".pub"
		if err := ssh.DeployPublicKey(sshHost, password, publicKeyPath); err == nil {
			keyDeployed = true
			m.toast = "✓ SSH key deployed successfully"
			m.toastSuccess = true
			m.toastTimer = 30
		} else {
			m.toast = fmt.Sprintf("Key deployment failed: %v", err)
			m.toastSuccess = false
			m.toastTimer = 50
			return
		}
	}

	h := config.Host{
		Name:            name,
		Hostname:        hostname,
		User:            user,
		Port:            port,
		IdentityFile:    identityFile,
		Password:        encryptedPassword,
		KeyDeployed:     keyDeployed,
		AutoGenerateKey: autoGenKey,
		ProxyJump:       strings.TrimSpace(m.inputs[6].Value()),
	}

	config.AddHost(h)
	if err := config.Save(); err != nil {
		m.toast = fmt.Sprintf("Failed to save: %v", err)
		m.toastSuccess = false
		m.toastTimer = 50
		return
	}

	m.hosts = config.GetHosts()
	if keyDeployed {
		m.toast = fmt.Sprintf("Host '%s' ready", name)
	} else {
		m.toast = fmt.Sprintf("Host '%s' added", name)
	}
	m.toastSuccess = true
	m.toastTimer = 30
	m.view = ViewHome
	m.inputs = nil
	m.showPassword = false
}

func (m Model) View() string {
	switch m.view {
	case ViewAddHost:
		return m.viewAddHost()
	case ViewPortForward:
		return m.viewPortForward()
	case ViewRunCommand:
		return m.viewRunCommand()
	case ViewFileTransfer:
		return m.viewFileTransfer()
	default:
		return m.viewWithTabs()
	}
}

type sshOutputMsg struct {
	TabId int
	Data  string
}

// sshErrorMsg: TabId 0 = error during connect (find tab with Connecting); TabId > 0 = error for that tab (e.g. connection closed).
type sshErrorMsg struct {
	TabId int
	Msg   string
}

type sshConnectedMsg struct {
	session *ssh.Session
}
type tickMsg struct{}
type animTickMsg struct{}
type sshTickMsg struct{}

type portForwardStartedMsg struct {
	fwd activeForward
}
type portForwardErrorMsg string

type runCommandResultMsg struct {
	output string
	err    error
}

type transferResultMsg struct {
	output string
	err    error
}

func runCommandCmd(host config.Host, command string) tea.Msg {
	sshHost := ssh.Host{
		Name:         host.Name,
		Hostname:     host.Hostname,
		User:         host.User,
		Port:         host.Port,
		IdentityFile: host.IdentityFile,
	}
	password := ""
	if host.Password != "" {
		dec, err := config.DecryptPassword(host.Password)
		if err != nil {
			return runCommandResultMsg{output: "", err: err}
		}
		password = dec
	}
	skipKey := !host.KeyDeployed && host.IdentityFile != ""
	var jumpHost *ssh.Host
	var jumpPassword string
	var jumpSkipKey bool
	if host.ProxyJump != "" {
		jumpConfig, _, ok := config.GetHostByName(host.ProxyJump)
		if !ok {
			return runCommandResultMsg{output: "", err: fmt.Errorf("proxy jump host not found: %s", host.ProxyJump)}
		}
		jumpHost = &ssh.Host{
			Name:         jumpConfig.Name,
			Hostname:     jumpConfig.Hostname,
			User:         jumpConfig.User,
			Port:         jumpConfig.Port,
			IdentityFile: jumpConfig.IdentityFile,
		}
		if jumpConfig.Password != "" {
			dec, _ := config.DecryptPassword(jumpConfig.Password)
			jumpPassword = dec
		}
		jumpSkipKey = !jumpConfig.KeyDeployed && jumpConfig.IdentityFile != ""
	}
	out, err := ssh.RunCommand(sshHost, password, skipKey, jumpHost, jumpPassword, jumpSkipKey, command)
	return runCommandResultMsg{output: out, err: err}
}

func (m *Model) initTransferInputs() {
	m.transferInputs = make([]textinput.Model, 3)
	for i := range m.transferInputs {
		m.transferInputs[i] = textinput.New()
		m.transferInputs[i].Prompt = ""
		m.transferInputs[i].CharLimit = 256
		m.transferInputs[i].Width = 50
	}
	m.transferInputs[0].Placeholder = "push or pull"
	m.transferInputs[1].Placeholder = "local path"
	m.transferInputs[2].Placeholder = "remote path (e.g. /tmp/file)"
	m.transferFocus = 0
}

func transferCmd(conn sftp.HostConnection, direction, localPath, remotePath string) tea.Msg {
	out, err := sftp.RunSCP(conn, direction, localPath, remotePath)
	return transferResultMsg{output: out, err: err}
}


func (m Model) connectSSH(host config.Host) tea.Cmd {
	return func() tea.Msg {
		h := ssh.Host{
			Name:         host.Name,
			Hostname:     host.Hostname,
			User:         host.User,
			Port:         host.Port,
			IdentityFile: host.IdentityFile,
		}

		var password string
		if host.Password != "" {
			decrypted, err := config.DecryptPassword(host.Password)
			if err != nil {
				return sshErrorMsg{0, fmt.Sprintf("failed to decrypt password: %v", err)}
			}
			password = decrypted
		}

		if password == "" && host.IdentityFile == "" {
			return sshErrorMsg{0, "no authentication method available (need password or key)"}
		}

		skipKey := !host.KeyDeployed && host.IdentityFile != ""

		var jumpHost *ssh.Host
		var jumpPassword string
		var jumpSkipKey bool
		if host.ProxyJump != "" {
			jumpConfig, _, ok := config.GetHostByName(host.ProxyJump)
			if !ok {
				return sshErrorMsg{0, "proxy jump host not found: " + host.ProxyJump}
			}
			jumpHost = &ssh.Host{
				Name:         jumpConfig.Name,
				Hostname:     jumpConfig.Hostname,
				User:         jumpConfig.User,
				Port:         jumpConfig.Port,
				IdentityFile: jumpConfig.IdentityFile,
			}
			if jumpConfig.Password != "" {
				dec, err := config.DecryptPassword(jumpConfig.Password)
				if err != nil {
					return sshErrorMsg{0, "failed to decrypt jump host password: " + err.Error()}
				}
				jumpPassword = dec
			}
			jumpSkipKey = !jumpConfig.KeyDeployed && jumpConfig.IdentityFile != ""
		}

		type connResult struct {
			session *ssh.Session
			err     error
		}
		ch := make(chan connResult, 1)
		go func() {
			session, err := ssh.Connect(h, password, skipKey, jumpHost, jumpPassword, jumpSkipKey)
			ch <- connResult{session, err}
		}()

		select {
		case res := <-ch:
			if res.err != nil {
				return sshErrorMsg{0, res.err.Error()}
			}
			return sshConnectedMsg{session: res.session}
		case <-time.After(15 * time.Second):
			return sshErrorMsg{0, "connection timed out (15s)"}
		}
	}
}


func (m Model) readSSHOutput(tabId int, session *ssh.Session) tea.Cmd {
	return func() tea.Msg {
		if session == nil {
			return nil
		}
		buf := make([]byte, 4096)
		n, err := session.Read(buf)
		if err == io.EOF {
			return sshErrorMsg{TabId: tabId, Msg: "connection closed"}
		}
		if err != nil {
			return sshErrorMsg{TabId: tabId, Msg: err.Error()}
		}
		if n > 0 {
			return sshOutputMsg{TabId: tabId, Data: string(buf[:n])}
		}
		return nil
	}
}

func (m Model) updateSSH(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.currentTabIndex <= 0 || m.currentTabIndex >= len(m.tabs) {
		return m, nil
	}
	cur := &m.tabs[m.currentTabIndex]
	// Close tab: Esc, Ctrl+Q, or Ctrl+]
	switch msg.String() {
	case "ctrl+]", "ctrl+q", "esc":
		if cur.Session != nil {
			cur.Session.Close()
			cur.Session = nil
		}
		m.tabs = append(m.tabs[:m.currentTabIndex], m.tabs[m.currentTabIndex+1:]...)
		if m.currentTabIndex >= len(m.tabs) {
			m.currentTabIndex = len(m.tabs) - 1
		}
		if m.currentTabIndex < 0 {
			m.currentTabIndex = 0
		}
		return m, nil
	}

	if cur.Session == nil {
		return m, nil
	}

	var data []byte
	if msg.Type == tea.KeyRunes {
		data = []byte(msg.String())
	} else {
		switch msg.Type {
		case tea.KeyEnter:
			data = []byte("\r")
		case tea.KeyBackspace:
			data = []byte("\x7f")
		case tea.KeyTab:
			data = []byte("\t")
		case tea.KeySpace:
			data = []byte(" ")
		case tea.KeyUp:
			data = []byte("\x1b[A")
		case tea.KeyDown:
			data = []byte("\x1b[B")
		case tea.KeyRight:
			data = []byte("\x1b[C")
		case tea.KeyLeft:
			data = []byte("\x1b[D")
		case tea.KeyEscape:
			data = []byte("\x1b")
		case tea.KeyDelete:
			data = []byte("\x1b[3~")
		case tea.KeyHome:
			data = []byte("\x1b[H")
		case tea.KeyEnd:
			data = []byte("\x1b[F")
		case tea.KeyPgUp:
			data = []byte("\x1b[5~")
		case tea.KeyPgDown:
			data = []byte("\x1b[6~")
		case tea.KeyInsert:
			data = []byte("\x1b[2~")
		case tea.KeyF1:
			data = []byte("\x1bOP")
		case tea.KeyF2:
			data = []byte("\x1bOQ")
		case tea.KeyF3:
			data = []byte("\x1bOR")
		case tea.KeyF4:
			data = []byte("\x1bOS")
		case tea.KeyF5:
			data = []byte("\x1b[15~")
		case tea.KeyF6:
			data = []byte("\x1b[17~")
		case tea.KeyF7:
			data = []byte("\x1b[18~")
		case tea.KeyF8:
			data = []byte("\x1b[19~")
		case tea.KeyF9:
			data = []byte("\x1b[20~")
		case tea.KeyF10:
			data = []byte("\x1b[21~")
		case tea.KeyF11:
			data = []byte("\x1b[23~")
		case tea.KeyF12:
			data = []byte("\x1b[24~")
		default:
			s := msg.String()
			switch s {
			case "ctrl+c":
				data = []byte("\x03")
			case "ctrl+d":
				data = []byte("\x04")
			case "ctrl+z":
				data = []byte("\x1a")
			case "ctrl+l":
				data = []byte("\x0c")
			case "ctrl+a":
				data = []byte("\x01")
			case "ctrl+e":
				data = []byte("\x05")
			case "ctrl+k":
				data = []byte("\x0b")
			case "ctrl+u":
				data = []byte("\x15")
			case "ctrl+w":
				data = []byte("\x17")
			case "ctrl+r":
				data = []byte("\x12")
			case "ctrl+p":
				data = []byte("\x10")
			case "ctrl+n":
				data = []byte("\x0e")
			default:
				if len(s) == 1 {
					data = []byte(s)
				}
			}
		}
	}

	if len(data) > 0 {
		cur.Session.Write(data)
	}
	return m, m.readSSHOutput(cur.Id, cur.Session)
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
