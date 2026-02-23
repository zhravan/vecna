package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorBg        = lipgloss.Color("#1a1b26")
	colorFg        = lipgloss.Color("#c0caf5")
	colorSubtle    = lipgloss.Color("#565f89")
	colorHighlight = lipgloss.Color("#7aa2f7")
	colorGreen     = lipgloss.Color("#9ece6a")
	colorRed       = lipgloss.Color("#f7768e")
	colorYellow    = lipgloss.Color("#e0af68")
	colorCyan      = lipgloss.Color("#7dcfff")
	colorPurple    = lipgloss.Color("#bb9af7")

	baseBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	styleApp = lipgloss.NewStyle()

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorHighlight).
			Padding(0, 1)

	styleLogo = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurple)

	stylePanel = lipgloss.NewStyle().
			Border(baseBorder).
			BorderForeground(colorSubtle).
			Padding(0, 1)

	stylePanelActive = lipgloss.NewStyle().
				Border(baseBorder).
				BorderForeground(colorHighlight).
				Padding(0, 1)

	stylePanelTitle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Bold(true)

	styleListItem = lipgloss.NewStyle().
			Foreground(colorFg)

	styleListItemSelected = lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true)

	styleListItemDim = lipgloss.NewStyle().
				Foreground(colorSubtle)

	styleStatusOnline = lipgloss.NewStyle().
				Foreground(colorGreen)

	styleStatusOffline = lipgloss.NewStyle().
				Foreground(colorRed)

	styleStatusBar = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Padding(0, 1)

	styleKey = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	styleKeyDesc = lipgloss.NewStyle().
			Foreground(colorSubtle)

	styleInputLabel = lipgloss.NewStyle().
			Foreground(colorHighlight).
			Bold(true)

	styleInputField = lipgloss.NewStyle().
			Foreground(colorFg)

	styleDim = lipgloss.NewStyle().
			Foreground(colorSubtle)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorGreen)

	styleError = lipgloss.NewStyle().
			Foreground(colorRed)

	styleWarning = lipgloss.NewStyle().
			Foreground(colorYellow)

	styleToast = lipgloss.NewStyle().
			Background(colorRed).
			Foreground(lipgloss.Color("#1a1b26")).
			Bold(true).
			Padding(0, 2).
			MarginTop(1)

	styleToastSuccess = lipgloss.NewStyle().
			Background(colorGreen).
			Foreground(lipgloss.Color("#1a1b26")).
			Bold(true).
			Padding(0, 2).
			MarginTop(1)

	stylePurple = lipgloss.NewStyle().
			Foreground(colorPurple).
			Bold(true)

	styleTab = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder(), false, false, true, false).
			BorderForeground(colorSubtle)

	styleTabActive = lipgloss.NewStyle().
			Foreground(colorHighlight).
			Bold(true).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder(), false, false, true, false).
			BorderForeground(colorHighlight)
)

func keyHint(k, desc string) string {
	return styleKey.Render(k) + styleKeyDesc.Render(":"+desc)
}
