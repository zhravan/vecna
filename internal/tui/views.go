package tui

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/shravan20/vecna/internal/config"
)

// wrapLines breaks text into lines and wraps each line to maxWidth runes (terminal column width).
// Returns a slice of lines each at most maxWidth runes; long lines are split into multiple lines.
func wrapLines(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		maxWidth = 80
	}
	raw := strings.Split(text, "\n")
	var out []string
	for _, line := range raw {
		line = strings.TrimRight(line, "\r")
		for len(line) > 0 {
			runeCount := utf8.RuneCountInString(line)
			if runeCount <= maxWidth {
				out = append(out, line)
				break
			}
			// take first maxWidth runes
			n := 0
			for i := 0; i < maxWidth && n < len(line); {
				_, size := utf8.DecodeRuneInString(line[n:])
				n += size
				i++
			}
			out = append(out, line[:n])
			line = line[n:]
		}
	}
	return out
}

// stripANSI removes VT100/ANSI escape sequences so we don't send clear-screen etc. to the terminal.
func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] != 0x1b {
			b.WriteByte(s[i])
			i++
			continue
		}
		i++
		if i >= len(s) {
			break
		}
		switch s[i] {
		case '[':
			// CSI: skip until letter
			i++
			for i < len(s) && (s[i] < 0x40 || s[i] > 0x7e) {
				i++
			}
			if i < len(s) {
				i++
			}
		case ']':
			// OSC: skip until BEL or ST
			i++
			for i < len(s) {
				if s[i] == 0x07 {
					i++
					break
				}
				if i+1 < len(s) && s[i] == 0x1b && s[i+1] == '\\' {
					i += 2
					break
				}
				i++
			}
		case '(', ')', '*', '+', '-', '.', '/':
			// two-byte sequence
			if i+1 < len(s) {
				i += 2
			} else {
				i = len(s)
			}
		default:
			i++
		}
	}
	return b.String()
}

func (m Model) viewHome() string {
	if m.width == 0 {
		return renderLoader(40, 12, m.animFrame, "Loading...")
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
	entries := m.filteredHostEntries()
	filterLine := "Filter: " + m.hostFilter.View()

	var items []string
	items = append(items, "", filterLine, "")
	if len(entries) == 0 {
		items = append(items, styleDim.Render("  No hosts yet"))
		items = append(items, styleDim.Render("  Press 'a' to add"))
		if m.hostFilter.Value() != "" {
			items = append(items, styleDim.Render("  (no match)"))
		}
	} else {
		for i, e := range entries {
			h := e.Host
			status := styleStatusOnline.Render("●")
			sel := " "
			if m.selectedHostNames[h.Name] {
				sel = "▣"
			}
			name := h.Name
			info := styleDim.Render(fmt.Sprintf("%s@%s", h.User, h.Hostname))

			var line string
			if i == m.cursor {
				name = styleListItemSelected.Render(name)
				line = fmt.Sprintf(" %s ▸ %s %s %s", sel, status, name, info)
			} else {
				name = styleListItem.Render(name)
				line = fmt.Sprintf(" %s   %s %s %s", sel, status, name, info)
			}
			items = append(items, line)
		}
	}

	listHeight := height - 2
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
	panelContent := lipgloss.JoinVertical(lipgloss.Left, title, list)

	panel := stylePanelActive.
		Width(width).
		Height(height).
		Render(panelContent)

	return panel
}

func (m Model) renderDetailPanel(width, height int) string {
	title := stylePanelTitle.Render("DETAILS")
	entries := m.filteredHostEntries()

	var content string
	if len(entries) == 0 || m.cursor >= len(entries) {
		content = styleDim.Render("Select a host to view details")
	} else {
		h := entries[m.cursor].Host
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
		if h.ProxyJump != "" {
			lines = append(lines, fmt.Sprintf("%s  %s", styleKey.Render("Jump"), h.ProxyJump))
		}
		// Active SSH count from server (who | wc -l)
		var activeSSHStr string
		if m.fetchingActiveSSHForHost == h.Name {
			activeSSHStr = "…"
		} else if count, ok := m.activeSSHCountByHost[h.Name]; ok && count >= 0 {
			activeSSHStr = fmt.Sprintf("%d", count)
		} else {
			activeSSHStr = "—"
		}
		lines = append(lines, fmt.Sprintf("%s  %s", styleKey.Render("Active SSH"), activeSSHStr))
		if at, ok := m.lastSSHAt[h.Name]; ok {
			ago := time.Since(at)
			var lastStr string
			if ago < time.Minute {
				lastStr = "just now"
			} else if ago < time.Hour {
				lastStr = fmt.Sprintf("%d min ago", int(ago.Minutes()))
			} else if ago < 24*time.Hour {
				lastStr = fmt.Sprintf("%d hr ago", int(ago.Hours()))
			} else {
				lastStr = at.Format("Jan 2 15:04")
			}
			lines = append(lines, fmt.Sprintf("%s  %s", styleKey.Render("Last SSH "), lastStr))
		}

		lines = append(lines, "")
		lines = append(lines, stylePanelTitle.Render("ACTIONS"))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  %s  Select (multi-run)", styleKey.Render("space")))
		lines = append(lines, fmt.Sprintf("  %s  SSH into host", styleKey.Render("c")))
		lines = append(lines, fmt.Sprintf("  %s  Edit host", styleKey.Render("e")))
		lines = append(lines, fmt.Sprintf("  %s  SFTP browser", styleKey.Render("f")))
		lines = append(lines, fmt.Sprintf("  %s  Port forward", styleKey.Render("p")))
		lines = append(lines, fmt.Sprintf("  %s  Run command (on selected)", styleKey.Render("r")))
		lines = append(lines, fmt.Sprintf("  %s  File transfer", styleKey.Render("t")))

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
		keyHint("1-9", "tab"),
		keyHint("ctrl+←/→", "switch"),
		keyHint("/", "filter"),
		keyHint("space", "select"),
		keyHint("↑↓", "nav"),
		keyHint("a", "add"),
		keyHint("i", "import"),
		keyHint("c", "connect"),
		keyHint("r", "run cmd"),
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
		return renderLoader(40, 12, m.animFrame, "Loading...")
	}

	logo := styleLogo.Render("◈ VECNA")
	headerLabel := " / Add Host"
	if m.editingHostIndex >= 0 {
		headerLabel = " / Edit Host"
	}
	header := styleHeader.Render(logo + styleDim.Render(headerLabel))

	formWidth := 50
	if formWidth > m.width-4 {
		formWidth = m.width - 4
	}

	var fields []string
	labels := []string{"Name", "Host", "User", "Port", "Password", "Auto-gen key", "Identity file", "Proxy jump"}

	for i, input := range m.inputs {
		labelText := labels[i]
		if i == 4 {
			if m.showPassword {
				labelText += " (visible)"
			} else {
				labelText += " (hidden)"
			}
		}

		field := input.View()

		if i == m.inputFocus {
			label := styleInputLabel.Render(fmt.Sprintf("%s:", labelText))
			fields = append(fields, fmt.Sprintf("%s\n%s", label, field))
		} else {
			fields = append(fields, fmt.Sprintf("%s\n%s", styleDim.Render(labelText+":"), field))
		}
	}

	form := strings.Join(fields, "\n\n")

	panel := stylePanelActive.
		Width(formWidth).
		Padding(1, 2).
		Render(form)

	hint := styleDim.Render("Tab/↓: next • Shift+Tab/↑: prev • Ctrl+P: toggle password • Enter: save • Esc: cancel")
	hint2 := styleDim.Render("Provide password and/or identity file path. No default key fallback.")

	mainView := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		panel,
		"",
		hint,
		hint2,
	)

	return m.renderWithToast(mainView)
}

func (m Model) viewPortForward() string {
	if m.width == 0 {
		return renderLoader(40, 12, m.animFrame, "Loading...")
	}

	logo := styleLogo.Render("◈ VECNA")
	header := styleHeader.Render(logo + styleDim.Render(" / Port Forward"))

	formWidth := 50
	if formWidth > m.width-4 {
		formWidth = m.width - 4
	}

	labels := []string{"Local port", "Remote host", "Remote port"}
	var fields []string
	for i, input := range m.portForwardInputs {
		label := labels[i]
		if i == m.portForwardFocus {
			fields = append(fields, styleInputLabel.Render(label+":")+"\n"+input.View())
		} else {
			fields = append(fields, styleDim.Render(label+":")+"\n"+input.View())
		}
	}
	form := strings.Join(fields, "\n\n")

	panel := stylePanelActive.Width(formWidth).Padding(1, 2).Render(form)

	activeTitle := stylePanelTitle.Render("ACTIVE FORWARDS")
	var listLines []string
	if len(m.activeForwards) == 0 {
		listLines = append(listLines, styleDim.Render("  None"))
	} else {
		for i, f := range m.activeForwards {
			line := fmt.Sprintf("  %s → %s", f.localAddr, f.remoteAddr)
			if i == m.portForwardCursor && m.portForwardFocus == 3 {
				line = " ▸ " + styleListItemSelected.Render(line)
			} else {
				line = "   " + styleListItem.Render(line)
			}
			listLines = append(listLines, line)
		}
	}
	listContent := strings.Join(listLines, "\n")
	listPanel := stylePanel.Width(formWidth).Padding(1, 2).Render(activeTitle + "\n\n" + listContent)

	var status string
	if m.portForwardStarting {
		status = styleDim.Render("Starting forward...")
	} else {
		status = keyHint("Tab", "next") + "  " + keyHint("Enter", "start") + "  " + keyHint("d", "stop") + "  " + keyHint("esc", "back")
	}

	mainView := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		panel,
		"",
		listPanel,
		"",
		styleStatusBar.Render(status),
	)
	return m.renderWithToast(mainView)
}

func (m Model) viewRunCommand() string {
	if m.width == 0 {
		return renderLoader(40, 12, m.animFrame, "Loading...")
	}

	logo := styleLogo.Render("◈ VECNA")
	header := styleHeader.Render(logo + styleDim.Render(" / Run command"))
	commands := config.GetCommands()

	// Multi-host results: show each host's output with a header
	if len(m.runCommandMultiResults) > 0 {
		var blocks []string
		for _, r := range m.runCommandMultiResults {
			block := "═══ " + r.Host + " ═══"
			if r.Err != nil {
				block += " (error)"
			}
			block += "\n" + r.Output
			blocks = append(blocks, block)
		}
		content := strings.Join(blocks, "\n\n")
		lines := strings.Split(content, "\n")
		maxH := m.height - 6
		if maxH < 5 {
			maxH = 5
		}
		if len(lines) > maxH {
			lines = lines[len(lines)-maxH:]
		}
		content = strings.Join(lines, "\n")
		status := styleStatusBar.Render(keyHint("esc", "back"))
		return lipgloss.JoinVertical(lipgloss.Left, header, "", content, "", status)
	}

	if m.runCommandOutput != "" {
		lines := strings.Split(m.runCommandOutput, "\n")
		maxH := m.height - 6
		if maxH < 5 {
			maxH = 5
		}
		if len(lines) > maxH {
			lines = lines[len(lines)-maxH:]
		}
		content := strings.Join(lines, "\n")
		status := styleStatusBar.Render(keyHint("esc", "back"))
		return lipgloss.JoinVertical(lipgloss.Left, header, "", content, "", status)
	}

	if m.runCommandRunning {
		msg := "Running..."
		if len(m.runCommandHosts) > 1 {
			msg = fmt.Sprintf("Running on %d hosts...", len(m.runCommandHosts))
		}
		loader := renderLoader(40, 8, m.animFrame, msg)
		return lipgloss.JoinVertical(lipgloss.Left, header, "", loader)
	}

	entries := m.filteredCommandEntries()
	var listLines []string
	if len(m.runCommandHosts) > 1 {
		names := make([]string, 0, len(m.runCommandHosts))
		for _, h := range m.runCommandHosts {
			names = append(names, h.Name)
		}
		listLines = append(listLines, styleDim.Render("  On: "+strings.Join(names, ", ")))
		listLines = append(listLines, "")
	}
	listLines = append(listLines, "", "Filter: "+m.runCommandFilter.View(), "")
	if len(entries) == 0 {
		listLines = append(listLines, styleDim.Render("  No commands (type in filter to search)"))
		if len(commands) == 0 {
			listLines = append(listLines, styleDim.Render("  Add to config: commands: [{ label: \"...\", command: \"...\" }]"))
		}
	} else {
		for i, e := range entries {
			c := e.Command
			active := i == m.runCommandCursor
			line := fmt.Sprintf("  %s", c.Label)
			if active && c.Command != "" {
				line += styleDim.Render("  → "+c.Command)
			}
			if active {
				line = " ▸ " + styleListItemSelected.Render(line)
			} else {
				line = "   " + styleListItem.Render(line)
			}
			listLines = append(listLines, line)
		}
	}
	list := strings.Join(listLines, "\n")
	panel := stylePanelActive.Width(60).Padding(1, 2).Render(list)
	status := styleStatusBar.Render(keyHint("/", "filter") + "  " + keyHint("↑↓", "nav") + "  " + keyHint("⏎", "run") + "  " + keyHint("esc", "back"))
	return lipgloss.JoinVertical(lipgloss.Left, header, "", panel, "", status)
}

func (m Model) viewImportSSH() string {
	logo := styleLogo.Render("◈ VECNA")
	header := styleHeader.Render(logo + styleDim.Render(" / Import from SSH config"))

	if m.importErr != "" {
		msg := styleDim.Render("Could not load ~/.ssh/config: " + m.importErr)
		status := styleStatusBar.Render(keyHint("esc", "back"))
		return lipgloss.JoinVertical(lipgloss.Left, header, "", msg, "", status)
	}

	if len(m.importCandidates) == 0 {
		msg := styleDim.Render("No hosts found in ~/.ssh/config (or file is empty).")
		status := styleStatusBar.Render(keyHint("esc", "back"))
		return lipgloss.JoinVertical(lipgloss.Left, header, "", msg, "", status)
	}

	entries := m.importCandidates
	width := 72
	if width > m.width-4 {
		width = m.width - 4
	}
	var lines []string
	lines = append(lines, "", styleDim.Render("Select hosts to import, then press Enter. Space = toggle."), "")
	for i, c := range entries {
		sel := " "
		if m.importSelected[c.Name] {
			sel = "▣"
		}
		info := fmt.Sprintf("%s@%s:%d", c.User, c.Hostname, c.Port)
		if c.IdentityFile != "" {
			base := filepath.Base(c.IdentityFile)
			if len(base) > 20 {
				base = base[:17] + "..."
			}
			info += "  " + styleDim.Render(base)
		}
		if c.ProxyJumpRaw != "" {
			info += "  " + styleDim.Render("→ "+c.ProxyJumpRaw)
		}
		nameAndInfo := c.Name + "  " + info
		if i == m.importCursor {
			lines = append(lines, " ▸ "+sel+" "+styleListItemSelected.Render(nameAndInfo))
		} else {
			lines = append(lines, "   "+sel+" "+styleListItem.Render(nameAndInfo))
		}
	}
	body := strings.Join(lines, "\n")
	panel := stylePanelActive.Width(width).Padding(1, 2).Render(body)
	status := styleStatusBar.Render(
		keyHint("space", "toggle") + "  " +
			keyHint("↑↓", "nav") + "  " +
			keyHint("⏎", "import selected") + "  " +
			keyHint("esc", "back"))
	return lipgloss.JoinVertical(lipgloss.Left, header, "", panel, "", status)
}

func (m Model) viewFileTransfer() string {
	if m.width == 0 {
		return renderLoader(40, 12, m.animFrame, "Loading...")
	}

	logo := styleLogo.Render("◈ VECNA")
	header := styleHeader.Render(logo + styleDim.Render(" / File transfer"))

	if m.transferOutput != "" {
		lines := strings.Split(m.transferOutput, "\n")
		maxH := m.height - 6
		if maxH < 5 {
			maxH = 5
		}
		if len(lines) > maxH {
			lines = lines[len(lines)-maxH:]
		}
		content := strings.Join(lines, "\n")
		status := styleStatusBar.Render(keyHint("esc", "back"))
		return lipgloss.JoinVertical(lipgloss.Left, header, "", content, "", status)
	}

	if m.transferRunning {
		loader := renderLoader(40, 8, m.animFrame, "Transferring...")
		return lipgloss.JoinVertical(lipgloss.Left, header, "", loader)
	}

	// Host→Host mode: source host + path, dest host + path
	if m.transferMode == 1 {
		return m.viewFileTransferHostToHost(header)
	}

	// Two-pane browser: Local | Remote (with scroll and filter)
	panelW := (m.width - 4) / 2
	if panelW < 20 {
		panelW = 20
	}
	maxRows := m.height - 8
	if maxRows < 3 {
		maxRows = 3
	}

	visLocal := filterTransferEntries(m.transferLocalEntries, m.transferLocalFilter)
	visRemote := filterTransferEntries(m.transferRemoteEntries, m.transferRemoteFilter)

	// Scroll window: show entries [start : start+maxRows], keep cursor in view
	leftStart := m.transferLocalOffset
	if m.transferLocalCursor < leftStart {
		leftStart = m.transferLocalCursor
	}
	if m.transferLocalCursor >= leftStart+maxRows {
		leftStart = m.transferLocalCursor - maxRows + 1
	}
	if leftStart > len(visLocal)-maxRows && len(visLocal) > maxRows {
		leftStart = len(visLocal) - maxRows
	}
	if leftStart < 0 {
		leftStart = 0
	}
	rightStart := m.transferRemoteOffset
	if m.transferRemoteCursor < rightStart {
		rightStart = m.transferRemoteCursor
	}
	if m.transferRemoteCursor >= rightStart+maxRows {
		rightStart = m.transferRemoteCursor - maxRows + 1
	}
	if rightStart > len(visRemote)-maxRows && len(visRemote) > maxRows {
		rightStart = len(visRemote) - maxRows
	}
	if rightStart < 0 {
		rightStart = 0
	}

	leftTitle := " LOCAL "
	if m.transferFocusPanel == 0 {
		leftTitle = " LOCAL ◀ "
	}
	if m.transferLocalFilter != "" {
		leftTitle += styleDim.Render(" [/" + m.transferLocalFilter + "]")
	}
	rightTitle := " REMOTE "
	if m.transferFocusPanel == 1 {
		rightTitle = " REMOTE ◀ "
	}
	if m.transferRemoteFilter != "" {
		rightTitle += styleDim.Render(" [/" + m.transferRemoteFilter + "]")
	}

	leftLines := []string{stylePanelTitle.Render(leftTitle), styleDim.Render(m.transferLocalCwd), ""}
	leftEnd := leftStart + maxRows
	if leftEnd > len(visLocal) {
		leftEnd = len(visLocal)
	}
	for j := leftStart; j < leftEnd; j++ {
		e := visLocal[j]
		line := "  "
		if e.IsDir {
			line += "📁 " + e.Name + "/"
		} else {
			line += "📄 " + e.Name
			if e.Size >= 0 {
				line += styleDim.Render(fmt.Sprintf("  (%d)", e.Size))
			}
		}
		full := m.transferLocalCwd + string(filepath.Separator) + e.Name
		if m.transferSelectedLocal != nil && m.transferSelectedLocal[full] {
			line = "▣ " + line
		} else {
			line = "  " + line
		}
		if j == m.transferLocalCursor {
			line = styleListItemSelected.Render(line)
		} else {
			line = styleListItem.Render(line)
		}
		leftLines = append(leftLines, line)
	}
	leftPanel := stylePanel.Width(panelW).Height(maxRows + 3).Render(strings.Join(leftLines, "\n"))

	rightLines := []string{stylePanelTitle.Render(rightTitle), styleDim.Render(m.transferRemoteCwd), ""}
	if m.transferRemoteLoading {
		rightLines = append(rightLines, styleDim.Render("  Loading..."))
	} else {
		rightEnd := rightStart + maxRows
		if rightEnd > len(visRemote) {
			rightEnd = len(visRemote)
		}
		for j := rightStart; j < rightEnd; j++ {
			e := visRemote[j]
			line := "  "
			if e.IsDir {
				line += "📁 " + e.Name + "/"
			} else {
				line += "📄 " + e.Name
				if e.Size >= 0 {
					line += styleDim.Render(fmt.Sprintf("  (%d)", e.Size))
				}
			}
			full := path.Join(m.transferRemoteCwd, e.Name)
			if m.transferSelectedRemote != nil && m.transferSelectedRemote[full] {
				line = "▣ " + line
			} else {
				line = "  " + line
			}
			if j == m.transferRemoteCursor {
				line = styleListItemSelected.Render(line)
			} else {
				line = styleListItem.Render(line)
			}
			rightLines = append(rightLines, line)
		}
	}
	rightPanel := stylePanel.Width(panelW).Height(maxRows + 3).Render(strings.Join(rightLines, "\n"))

	if m.transferFocusPanel == 1 {
		rightPanel = stylePanelActive.Width(panelW).Height(maxRows + 3).Render(strings.Join(rightLines, "\n"))
	} else {
		leftPanel = stylePanelActive.Width(panelW).Height(maxRows + 3).Render(strings.Join(leftLines, "\n"))
	}

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
	statusStr := keyHint("Tab", "switch") + "  " +
		keyHint("↑↓", "nav") + "  " +
		keyHint("/", "search") + "  " +
		keyHint("~", "home") + "  " +
		keyHint("\\", "root") + "  " +
		keyHint("⌫", "parent") + "  " +
		keyHint("space", "select") + "  " +
		keyHint("⏎", "open/transfer") + "  " +
		keyHint("m", "Host→Host") + "  " +
		keyHint("esc", "back")
	if m.transferFilterFocused {
		statusStr = styleKey.Render("Filter: ") + m.transferFilterInput.View() + "  " + styleDim.Render("(Enter apply, Esc cancel)")
	}
	status := styleStatusBar.Render(statusStr)
	return lipgloss.JoinVertical(lipgloss.Left, header, "", panels, "", status)
}

func (m Model) viewFileTransferHostToHost(header string) string {
	hosts := config.GetHosts()
	panelW := (m.width - 4) / 2
	if panelW < 24 {
		panelW = 24
	}
	leftLines := []string{stylePanelTitle.Render(" SOURCE "), ""}
	for i, h := range hosts {
		line := "  " + h.Name + "  " + styleDim.Render(fmt.Sprintf("%s@%s", h.User, h.Hostname))
		if i == m.transferSourceHostIdx {
			line = "▸ " + styleListItemSelected.Render(line)
		} else {
			line = "  " + styleListItem.Render(line)
		}
		leftLines = append(leftLines, line)
	}
	leftLines = append(leftLines, "", styleInputLabel.Render("Path:"), m.transferSourcePathInput.View())
	leftPanel := stylePanel.Width(panelW).Render(strings.Join(leftLines, "\n"))
	if m.transferHostHostFocus == 0 || m.transferHostHostFocus == 1 {
		leftPanel = stylePanelActive.Width(panelW).Render(strings.Join(leftLines, "\n"))
	}
	rightLines := []string{stylePanelTitle.Render(" DEST "), ""}
	for i, h := range hosts {
		line := "  " + h.Name + "  " + styleDim.Render(fmt.Sprintf("%s@%s", h.User, h.Hostname))
		if i == m.transferDestHostIdx {
			line = "▸ " + styleListItemSelected.Render(line)
		} else {
			line = "  " + styleListItem.Render(line)
		}
		rightLines = append(rightLines, line)
	}
	rightLines = append(rightLines, "", styleInputLabel.Render("Path:"), m.transferDestPathInput.View())
	rightPanel := stylePanel.Width(panelW).Render(strings.Join(rightLines, "\n"))
	if m.transferHostHostFocus == 2 || m.transferHostHostFocus == 3 {
		rightPanel = stylePanelActive.Width(panelW).Render(strings.Join(rightLines, "\n"))
	}
	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
	status := styleStatusBar.Render(
		keyHint("m", "Local↔Host") + "  " +
			keyHint("Tab", "focus") + "  " +
			keyHint("↑↓", "host") + "  " +
			keyHint("⏎", "run") + "  " +
			keyHint("esc", "back"))
	return lipgloss.JoinVertical(lipgloss.Left, header, "", panels, "", status)
}

// viewTabContent returns the main area content when in tab view (Hosts or current SSH tab). No tab bar.
func (m Model) viewTabContent() string {
	if m.currentTabIndex == 0 {
		return m.viewHome()
	}
	if m.currentTabIndex < len(m.tabs) {
		return m.viewSSHTab(m.tabs[m.currentTabIndex])
	}
	return m.viewHome()
}

func (m Model) renderTabBar() string {
	var parts []string
	for i, t := range m.tabs {
		title := t.Title
		if t.Connecting {
			title = t.Title + " …"
		}
		if i == m.currentTabIndex {
			parts = append(parts, styleTabActive.Render(title))
		} else {
			parts = append(parts, styleTab.Render(title))
		}
	}
	line := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	// Keep tab bar to one line so it never pushes off screen; truncate if too wide.
	if m.width > 4 && utf8.RuneCountInString(line) > m.width {
		runes := []rune(line)
		line = string(runes[:m.width-3]) + "…"
	}
	return line
}

func (m Model) viewSSHTab(t tab) string {
	if m.width == 0 {
		return renderLoader(40, 12, m.animFrame, "Starting up...")
	}
	if t.Connecting {
		terminalHeight := m.height - 4
		if terminalHeight < 5 {
			terminalHeight = 5
		}
		msg := "Connecting..."
		if t.Host.Hostname != "" {
			msg = fmt.Sprintf("Connecting to %s@%s ...", t.Host.User, t.Host.Hostname)
		}
		loader := renderLoaderFullscreen(m.width, m.height, m.animFrame, terminalHeight, msg)
		statusBar := styleStatusBar.Render(keyHint("esc", "cancel") + "  " + keyHint("ctrl+q", "cancel"))
		return lipgloss.JoinVertical(lipgloss.Left, loader, "", statusBar)
	}
	if t.Session != nil {
		raw := t.Output
		output := stripANSI(raw)
		output = strings.ReplaceAll(output, "\r\n", "\n")
		output = strings.ReplaceAll(output, "\r", "\n")
		if output == "" {
			output = styleDim.Render("Waiting for output... (Esc or Ctrl+Q to close tab)")
		}
		termWidth := m.width
		if termWidth < 40 {
			termWidth = 40
		}
		// Reserve space for tab bar (1) + status bar (1) + headroom so tab bar is never scrolled off.
		termHeight := m.height - 4
		if termHeight < 5 {
			termHeight = 5
		}
		lines := wrapLines(output, termWidth-2)
		if len(lines) > termHeight {
			lines = lines[len(lines)-termHeight:]
		}
		screen := strings.Join(lines, "\n")
		statusBar := styleStatusBar.Render(keyHint("1-9", "tab") + "  " + keyHint("ctrl+←/→", "switch") + "  " + keyHint("esc", "close"))
		terminalBox := stylePanelActive.
			Width(termWidth).
			Height(termHeight).
			Render(screen)
		return terminalBox + "\n" + statusBar
	}
	return styleDim.Render("No active session")
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

	var toast string
	if m.toastSuccess {
		toast = styleToastSuccess.Render(fmt.Sprintf("✓ %s", toastText))
	} else {
		toast = styleToast.Render(fmt.Sprintf("✕ %s", toastText))
	}
	toastWidth := lipgloss.Width(toast)

	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return content
	}

	toastLine := strings.Repeat(" ", m.width-toastWidth-2) + toast
	lines[1] = toastLine

	return strings.Join(lines, "\n")
}
