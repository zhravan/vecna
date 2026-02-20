package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewHome() string {
	if m.width == 0 {
		return "Loading..."
	}

	logo := styleLogo.Render("◈ VECNA")
	header := styleHeader.Render(logo)

	contentHeight := m.height - 6
	if contentHeight < 10 {
		contentHeight = 10
	}

	leftWidth := m.width*2/5 - 4
	rightWidth := m.width*3/5 - 4

	if leftWidth < 20 {
		leftWidth = 20
	}
	if rightWidth < 30 {
		rightWidth = 30
	}

	hostsPanel := m.renderHostsPanel(leftWidth, contentHeight-2)
	detailPanel := m.renderDetailPanel(rightWidth, contentHeight-2)

	content := lipgloss.JoinHorizontal(lipgloss.Top, hostsPanel, " ", detailPanel)

	statusBar := m.renderStatusBar()

	mainView := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		content,
		"",
		statusBar,
	)

	return m.renderWithToast(mainView)
}

func (m Model) renderHostsPanel(width, height int) string {
	title := stylePanelTitle.Render("HOSTS")

	var items []string
	if len(m.hosts) == 0 {
		items = append(items, styleDim.Render("  No hosts yet"))
		items = append(items, styleDim.Render("  Press 'a' to add"))
	} else {
		for i, h := range m.hosts {
			status := styleStatusOnline.Render("●")
			name := h.Name
			info := styleDim.Render(fmt.Sprintf("%s@%s", h.User, h.Hostname))

			var line string
			if i == m.cursor {
				name = styleListItemSelected.Render(name)
				line = fmt.Sprintf(" ▸ %s %s %s", status, name, info)
			} else {
				name = styleListItem.Render(name)
				line = fmt.Sprintf("   %s %s %s", status, name, info)
			}
			items = append(items, line)
		}
	}

	listHeight := height - 3
	if listHeight < 1 {
		listHeight = 1
	}

	for len(items) < listHeight {
		items = append(items, "")
	}
	if len(items) > listHeight {
		items = items[:listHeight]
	}

	list := strings.Join(items, "\n")
	panelContent := lipgloss.JoinVertical(lipgloss.Left, title, "", list)

	panel := stylePanelActive.
		Width(width).
		Height(height).
		Render(panelContent)

	return panel
}

func (m Model) renderDetailPanel(width, height int) string {
	title := stylePanelTitle.Render("DETAILS")

	var content string
	if len(m.hosts) == 0 || m.cursor >= len(m.hosts) {
		content = styleDim.Render("Select a host to view details")
	} else {
		h := m.hosts[m.cursor]
		lines := []string{
			fmt.Sprintf("%s  %s", styleKey.Render("Name"), h.Name),
			fmt.Sprintf("%s  %s", styleKey.Render("Host"), h.Hostname),
			fmt.Sprintf("%s  %s", styleKey.Render("User"), h.User),
			fmt.Sprintf("%s  %d", styleKey.Render("Port"), h.Port),
		}
		if h.IdentityFile != "" {
			lines = append(lines, fmt.Sprintf("%s  %s", styleKey.Render("Key "), h.IdentityFile))
		}
		if len(h.Tags) > 0 {
			lines = append(lines, fmt.Sprintf("%s  %s", styleKey.Render("Tags"), strings.Join(h.Tags, ", ")))
		}

		lines = append(lines, "")
		lines = append(lines, stylePanelTitle.Render("ACTIONS"))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s  SSH into host", styleKey.Render("c")))
		lines = append(lines, fmt.Sprintf("  %s  Edit host", styleKey.Render("e")))
		lines = append(lines, fmt.Sprintf("  %s  SFTP browser", styleKey.Render("f")))
		lines = append(lines, fmt.Sprintf("  %s  Port forward", styleKey.Render("p")))

		content = strings.Join(lines, "\n")
	}

	panelContent := lipgloss.JoinVertical(lipgloss.Left, title, "", content)

	panel := stylePanel.
		Width(width).
		Height(height).
		Render(panelContent)

	return panel
}

func (m Model) renderStatusBar() string {
	keys := []string{
		keyHint("↑↓", "nav"),
		keyHint("a", "add"),
		keyHint("d", "del"),
		keyHint("c", "connect"),
		keyHint("?", "help"),
		keyHint("q", "quit"),
	}

	left := strings.Join(keys, "  ")

	hostCount := styleDim.Render(fmt.Sprintf("%d hosts", len(m.hosts)))

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(hostCount) - 4
	if gap < 1 {
		gap = 1
	}

	return styleStatusBar.Render(left + strings.Repeat(" ", gap) + hostCount)
}

func (m Model) viewAddHost() string {
	if m.width == 0 {
		return "Loading..."
	}

	logo := styleLogo.Render("◈ VECNA")
	header := styleHeader.Render(logo + styleDim.Render(" / Add Host"))

	formWidth := 50
	if formWidth > m.width-4 {
		formWidth = m.width - 4
	}

	var fields []string
	labels := []string{"Name", "Host", "User", "Port"}

	for i, input := range m.inputs {
		label := styleInputLabel.Render(fmt.Sprintf("%s:", labels[i]))
		field := input.View()

		if i == m.inputFocus {
			fields = append(fields, fmt.Sprintf("%s\n%s", label, field))
		} else {
			fields = append(fields, fmt.Sprintf("%s\n%s", styleDim.Render(labels[i]+":"), field))
		}
	}

	form := strings.Join(fields, "\n\n")

	panel := stylePanelActive.
		Width(formWidth).
		Padding(1, 2).
		Render(form)

	hint := styleDim.Render("Tab/↓: next • Shift+Tab/↑: prev • Enter: save • Esc: cancel")

	mainView := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		panel,
		"",
		hint,
	)

	return m.renderWithToast(mainView)
}

func (m Model) viewSSH() string {
	if m.width == 0 {
		return "Loading..."
	}

	var header string
	if m.sshHost != nil {
		logo := styleLogo.Render("◈ VECNA")
		connInfo := styleDim.Render(fmt.Sprintf(" / %s@%s", m.sshHost.User, m.sshHost.Hostname))
		header = styleHeader.Render(logo + connInfo)
	} else {
		header = styleHeader.Render(styleLogo.Render("◈ VECNA"))
	}

	var content string
	if m.connecting {
		content = styleDim.Render("Connecting...")
	} else if m.err != nil {
		content = styleError.Render(fmt.Sprintf("Error: %v", m.err))
	} else {
		output := m.sshOutput.String()
		if output == "" {
			content = styleDim.Render("Connected. Waiting for output...")
		} else {
			content = output
		}
	}

	terminalHeight := m.height - 4
	if terminalHeight < 5 {
		terminalHeight = 5
	}

	terminalWidth := m.width - 4

	terminal := stylePanel.
		Width(terminalWidth).
		Height(terminalHeight).
		Render(content)

	statusBar := styleStatusBar.Render(keyHint("esc", "disconnect") + "  " + keyHint("q", "quit"))

	mainView := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		terminal,
		"",
		statusBar,
	)

	return m.renderWithToast(mainView)
}

func (m Model) renderWithToast(content string) string {
	if m.toast == "" {
		return content
	}

	toastText := m.toast
	maxWidth := m.width / 3
	if len(toastText) > maxWidth {
		toastText = toastText[:maxWidth-3] + "..."
	}

	toast := styleToast.Render(fmt.Sprintf("✕ %s", toastText))
	toastWidth := lipgloss.Width(toast)

	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return content
	}

	toastLine := strings.Repeat(" ", m.width-toastWidth-2) + toast
	lines[1] = toastLine

	return strings.Join(lines, "\n")
}
