package tui

// RunCommandListPanelInnerWidth is the content width of the run-command list panel.
// Must match mouse hit-testing in mouse.go.
const RunCommandListPanelInnerWidth = 60

// HomeHostsPanelWidth returns the left column width used by viewHome for the hosts panel.
// Keep in sync with renderHostsPanel(leftWidth, ...) and mouse hit-testing for that list.
func HomeHostsPanelWidth(termWidth int) int {
	if termWidth <= 0 {
		return 20
	}
	w := termWidth*2/5 - 4
	if w < 20 {
		w = 20
	}
	return w
}
