package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Mouse layout rows are 0-based from the top of the Bubble Tea root view (tab bar = row 0).
// These must stay aligned with viewHome / viewRunCommand vertical structure.
const (
	layoutMouseTabBarRow = 0

	// First screen row where the host list rows begin (after tab, header, blank, panel chrome, filter).
	layoutMouseHomeFirstHostRow = 6

	// First row of command labels inside the run-command panel (after header, blank, panel + filter block).
	layoutMouseRunCommandFirstListRow       = 5
	layoutMouseRunCommandMultiHostExtraRows = 2

	// Horizontal extent beyond RunCommandListPanelInnerWidth for panel borders (left + right).
	layoutMouseRunCommandPanelBorderCells = 2
)

// mouseTabBarBlockedByView returns true when tab-bar clicks should be ignored so we do not
// change currentTabIndex behind a full-screen or modal flow.
func mouseTabBarBlockedByView(v View) bool {
	switch v {
	case ViewAddHost, ViewAddCommand, ViewDeleteCommandConfirm, ViewDeleteConfirm,
		ViewPortForward, ViewFileTransfer, ViewImportSSH, ViewVersion:
		return true
	default:
		return false
	}
}

func runCommandListStartRow(multiHost bool) int {
	row := layoutMouseRunCommandFirstListRow
	if multiHost {
		row += layoutMouseRunCommandMultiHostExtraRows
	}
	return row
}

// runCommandPanelMaxInclusiveX is the largest column index (inclusive) inside the run-command list panel.
func runCommandPanelMaxInclusiveX(termWidth int) int {
	extent := RunCommandListPanelInnerWidth + layoutMouseRunCommandPanelBorderCells
	if termWidth > 0 && extent > termWidth {
		return termWidth - 1
	}
	return extent - 1
}

func (m Model) mouseTabRenderedWidth(tabIdx int) int {
	t := m.tabs[tabIdx]
	title := t.Title
	if t.Connecting {
		title += " …"
	}
	var rendered string
	if tabIdx == m.currentTabIndex {
		rendered = styleTabActive.Render(title)
	} else {
		rendered = styleTab.Render(title)
	}
	return lipgloss.Width(rendered)
}

func (m Model) mouseTabIndexAtX(x int) (int, bool) {
	col := 0
	for i := range m.tabs {
		w := m.mouseTabRenderedWidth(i)
		if x >= col && x < col+w {
			return i, true
		}
		col += w
	}
	return 0, false
}

func (m Model) mouseTryTabSwitch(x, y int, isClick bool) (Model, tea.Cmd, bool) {
	if !isClick || y != layoutMouseTabBarRow || len(m.tabs) <= 1 {
		return m, nil, false
	}
	if mouseTabBarBlockedByView(m.view) {
		return m, nil, false
	}
	i, ok := m.mouseTabIndexAtX(x)
	if !ok {
		return m, nil, false
	}
	m.currentTabIndex = i
	if i > 0 && m.tabs[i].Session != nil {
		return m, m.readSSHOutput(m.tabs[i].Id, m.tabs[i].Session), true
	}
	return m, nil, true
}

func (m Model) mouseOnHomeView(x, y int, isClick, isWheel, wheelUp bool) (Model, tea.Cmd, bool) {
	if m.currentTabIndex != 0 {
		return m, nil, false
	}
	listMaxX := HomeHostsPanelWidth(m.width) - 1
	if listMaxX < 0 {
		listMaxX = 0
	}
	if (isClick || isWheel) && (x < 0 || x > listMaxX) {
		return m, nil, false
	}
	entries := m.filteredHostEntries()
	if isClick {
		clicked := y - layoutMouseHomeFirstHostRow
		if clicked >= 0 && clicked < len(entries) {
			m.cursor = clicked
			m.fetchingActiveSSHForHost = entries[m.cursor].Host.Name
			return m, fetchActiveSSHCountCmd(entries[m.cursor].Host), true
		}
		return m, nil, false
	}
	if isWheel {
		if wheelUp && m.cursor > 0 {
			m.cursor--
		} else if !wheelUp && m.cursor < len(entries)-1 {
			m.cursor++
		}
		if len(entries) > 0 && m.cursor < len(entries) {
			m.fetchingActiveSSHForHost = entries[m.cursor].Host.Name
			return m, fetchActiveSSHCountCmd(entries[m.cursor].Host), true
		}
		return m, nil, false
	}
	return m, nil, false
}

func (m Model) mouseOnRunCommandView(x, y int, isClick, isWheel, wheelUp bool) (Model, tea.Cmd, bool) {
	if m.runCommandOutput != "" || len(m.runCommandMultiResults) > 0 || m.runCommandRunning {
		return m, nil, false
	}
	maxX := runCommandPanelMaxInclusiveX(m.width)
	if (isClick || isWheel) && (x < 0 || x > maxX) {
		return m, nil, false
	}
	entries := m.filteredCommandEntries()
	startRow := runCommandListStartRow(len(m.runCommandHosts) > 1)
	if isClick {
		clicked := y - startRow
		if clicked >= 0 && clicked < len(entries) {
			m.runCommandCursor = clicked
			return m, nil, true
		}
		return m, nil, false
	}
	if isWheel {
		if wheelUp && m.runCommandCursor > 0 {
			m.runCommandCursor--
		} else if !wheelUp && m.runCommandCursor < len(entries)-1 {
			m.runCommandCursor++
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m Model) mouseOnFileTransferView(x int, isClick, isWheel, wheelUp bool) (Model, tea.Cmd, bool) {
	if m.transferOutput != "" || m.transferRunning || m.transferMode == 1 {
		return m, nil, false
	}
	if isClick {
		panelW := (m.width - 4) / 2
		if x < panelW+1 {
			m.transferFocusPanel = 0
		} else {
			m.transferFocusPanel = 1
		}
		return m, nil, true
	}
	if isWheel {
		if m.transferFocusPanel == 0 {
			visLocal := filterTransferEntries(m.transferLocalEntries, m.transferLocalFilter)
			if wheelUp && m.transferLocalCursor > 0 {
				m.transferLocalCursor--
			} else if !wheelUp && m.transferLocalCursor < len(visLocal)-1 {
				m.transferLocalCursor++
			}
		} else {
			visRemote := filterTransferEntries(m.transferRemoteEntries, m.transferRemoteFilter)
			if wheelUp && m.transferRemoteCursor > 0 {
				m.transferRemoteCursor--
			} else if !wheelUp && m.transferRemoteCursor < len(visRemote)-1 {
				m.transferRemoteCursor++
			}
		}
		return m, nil, true
	}
	return m, nil, false
}

func (m Model) mouseOnImportSSHView(isWheel, wheelUp bool) (Model, tea.Cmd, bool) {
	if !isWheel {
		return m, nil, false
	}
	if wheelUp && m.importCursor > 0 {
		m.importCursor--
	} else if !wheelUp && m.importCursor < len(m.importCandidates)-1 {
		m.importCursor++
	}
	return m, nil, true
}

func (m Model) mouseOnPortForwardView(isWheel, wheelUp bool) (Model, tea.Cmd, bool) {
	if !isWheel || m.portForwardFocus != 3 {
		return m, nil, false
	}
	if wheelUp && m.portForwardCursor > 0 {
		m.portForwardCursor--
	} else if !wheelUp && m.portForwardCursor < len(m.activeForwards)-1 {
		m.portForwardCursor++
	}
	return m, nil, true
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	x, y := msg.X, msg.Y
	isClick := msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft
	wheelUp := msg.Button == tea.MouseButtonWheelUp
	wheelDown := msg.Button == tea.MouseButtonWheelDown
	isWheel := wheelUp || wheelDown

	if next, cmd, ok := m.mouseTryTabSwitch(x, y, isClick); ok {
		return next, cmd
	}

	switch m.view {
	case ViewHome:
		if next, cmd, ok := m.mouseOnHomeView(x, y, isClick, isWheel, wheelUp); ok {
			return next, cmd
		}
	case ViewRunCommand:
		if next, cmd, ok := m.mouseOnRunCommandView(x, y, isClick, isWheel, wheelUp); ok {
			return next, cmd
		}
	case ViewFileTransfer:
		if next, cmd, ok := m.mouseOnFileTransferView(x, isClick, isWheel, wheelUp); ok {
			return next, cmd
		}
	case ViewImportSSH:
		if next, cmd, ok := m.mouseOnImportSSHView(isWheel, wheelUp); ok {
			return next, cmd
		}
	case ViewPortForward:
		if next, cmd, ok := m.mouseOnPortForwardView(isWheel, wheelUp); ok {
			return next, cmd
		}
	}

	return m, nil
}
