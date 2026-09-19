package tui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zhravan/vecna/internal/config"
	vecnassh "github.com/zhravan/vecna/internal/ssh"
)

func (m Model) updateKnownHostConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "n" || msg.String() == "N" || msg.String() == "esc" || key.Matches(msg, m.keys.Back) {
		return m.cancelKnownHostPrompt()
	}

	var unknown *vecnassh.UnknownHostKeyError
	if errors.As(m.err, &unknown) {
		if msg.String() == "y" || msg.String() == "Y" || msg.String() == "enter" {
			if m.sshHost == nil || unknown.Key == nil {
				return m.cancelKnownHostPrompt()
			}
			if err := vecnassh.TrustHostKey(knownHostAddress(*m.sshHost), unknown.Key); err != nil {
				m.toast = fmt.Sprintf("Failed to trust host key: %v", err)
				m.toastSuccess = false
				m.toastTimer = 80
				return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
			}

			// Host creation validation reached this screen before the host was
			// persisted. Keep the form and retry validation; saveHost will only
			// persist after the trusted key is accepted.
			if m.knownHostFromAdd {
				m.err = nil
				m.view = ViewAddHost
				return m, nil
			}

			host := *m.sshHost
			m.err = nil
			m.sshHost = nil
			m.view = ViewHome
			return m, m.connectSSH(host)
		}
		return m, nil
	}

	// Changed keys intentionally have no trust/replace action here.
	var changed *vecnassh.ChangedHostKeyError
	if errors.As(m.err, &changed) {
		return m, nil
	}
	return m.cancelKnownHostPrompt()
}

func (m Model) cancelKnownHostPrompt() (tea.Model, tea.Cmd) {
	if m.knownHostFromAdd {
		m.err = nil
		m.sshHost = nil
		m.knownHostFromAdd = false
		m.view = ViewAddHost
		return m, nil
	}

	for i := range m.tabs {
		if m.tabs[i].Connecting {
			if m.tabs[i].Session != nil {
				_ = m.tabs[i].Session.Close()
			}
			m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)
			break
		}
	}
	if m.currentTabIndex >= len(m.tabs) {
		m.currentTabIndex = len(m.tabs) - 1
	}
	if m.currentTabIndex < 0 {
		m.currentTabIndex = 0
	}
	m.err = nil
	m.sshHost = nil
	m.view = ViewHome
	return m, nil
}

func knownHostAddress(h config.Host) string {
	if h.Port == 0 || h.Port == 22 {
		return h.Hostname
	}
	return fmt.Sprintf("[%s]:%d", h.Hostname, h.Port)
}

func (m Model) viewKnownHostConfirm() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#38BDF8")).Render("SSH HOST KEY VERIFICATION")
	if m.sshHost == nil {
		return centerKnownHostModal(title + "\n\nUnable to identify the host.\n\nPress Esc to cancel.")
	}
	host := *m.sshHost
	lines := []string{title, "", lipgloss.NewStyle().Bold(true).Render(host.Name), host.User + "@" + host.Hostname, ""}

	var unknown *vecnassh.UnknownHostKeyError
	if errors.As(m.err, &unknown) {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true).Render("FIRST CONNECTION — KEY NOT TRUSTED"),
			"",
			"Algorithm:   "+unknown.Algorithm,
			"Fingerprint: "+unknown.Fingerprint,
			"",
			"Verify this fingerprint independently before trusting it.",
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Bold(true).Render("[Enter/Y] Trust & Connect"),
			"[Esc/N] Cancel",
		)
		return centerKnownHostModal(strings.Join(lines, "\n"))
	}

	var changed *vecnassh.ChangedHostKeyError
	if errors.As(m.err, &changed) {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true).Render("WARNING: HOST KEY CHANGED"),
			"",
			"Previous fingerprint(s):",
		)
		if len(changed.PreviousFingerprints) == 0 {
			lines = append(lines, "  unavailable")
		} else {
			for _, fp := range changed.PreviousFingerprints {
				lines = append(lines, "  "+fp)
			}
		}
		lines = append(lines,
			"Presented fingerprint:",
			"  "+changed.Fingerprint,
			"",
			"The saved key does not match the server key.",
			"Vecna will not automatically replace it.",
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true).Render("[Esc] Close"),
		)
		return centerKnownHostModal(strings.Join(lines, "\n"))
	}

	return centerKnownHostModal(strings.Join(lines, "\n"))
}

func centerKnownHostModal(content string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#475569")).
		Padding(1, 3).
		Width(76).
		Render(content)
}
