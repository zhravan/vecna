package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorFg        = lipgloss.Color("#c0caf5")
	colorSubtle    = lipgloss.Color("#565f89")
	colorHighlight = lipgloss.Color("#7aa2f7")
	colorGreen     = lipgloss.Color("#9ece6a")
	colorRed       = lipgloss.Color("#f7768e")
	colorYellow    = lipgloss.Color("#e0af68")
	colorCyan      = lipgloss.Color("#7dcfff")
	colorPurple    = lipgloss.Color("#bb9af7")
	colorOrange    = lipgloss.Color("#ff9e64")
	colorTeal      = lipgloss.Color("#73daca")
	colorMagenta   = lipgloss.Color("#ff007c")
	colorSurface   = lipgloss.Color("#24283b")
	colorBorderDim = lipgloss.Color("#3b4261")

	// brandLetterColors cycles across "◈ VECNA" for a richer header.
	brandLetterColors = []lipgloss.Color{
		colorPurple, colorHighlight, colorCyan, colorGreen, colorYellow, colorOrange,
	}

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

	// Header wraps the brand; brand carries its own per-rune colors.
	styleHeader = lipgloss.NewStyle().
			Padding(0, 1)

	stylePanel = lipgloss.NewStyle().
			Border(baseBorder).
			BorderForeground(colorBorderDim).
			Padding(0, 1)

	stylePanelActive = lipgloss.NewStyle().
				Border(baseBorder).
				BorderForeground(colorHighlight).
				Padding(0, 1)

	// SSH session output: teal border to distinguish from generic panels.
	stylePanelSSH = lipgloss.NewStyle().
			Border(baseBorder).
			BorderForeground(colorTeal).
			Padding(0, 1)

	stylePanelTitleHosts = lipgloss.NewStyle().
				Foreground(colorCyan).
				Bold(true)

	stylePanelTitleDetails = lipgloss.NewStyle().
				Foreground(colorPurple).
				Bold(true)

	stylePanelTitleActions = lipgloss.NewStyle().
				Foreground(colorYellow).
				Bold(true)

	stylePanelTitleForwards = lipgloss.NewStyle().
				Foreground(colorTeal).
				Bold(true)

	stylePanelTitleLocal = lipgloss.NewStyle().
				Foreground(colorCyan).
				Bold(true)

	stylePanelTitleRemote = lipgloss.NewStyle().
				Foreground(colorOrange).
				Bold(true)

	stylePanelTitleSource = lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true)

	stylePanelTitleDest = lipgloss.NewStyle().
				Foreground(colorMagenta).
				Bold(true)

	styleBreadcrumb = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Italic(true)

	styleRunHostHeader = lipgloss.NewStyle().
			Foreground(colorMagenta).
			Bold(true)

	styleDetailValue = lipgloss.NewStyle().
			Foreground(colorTeal)

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
			Background(colorSurface).
			Foreground(colorSubtle).
			BorderTop(true).
			BorderStyle(lipgloss.Border{Top: "─"}).
			BorderForeground(colorPurple).
			Padding(0, 1)

	styleKey = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	styleKeyDesc = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#565f89", Dark: "#a9b1d6"})

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
			BorderForeground(colorBorderDim)

	styleTabActive = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true).
			Background(colorSurface).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder(), false, false, true, false).
			BorderForeground(colorHighlight)
)

// renderBrand returns "◈ VECNA" with cycling accent colors (Tokyo Night–style palette).
func renderBrand() string {
	const text = "◈ VECNA"
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}
	var b strings.Builder
	b.Grow(len(text) * 12)
	for i, r := range runes {
		c := brandLetterColors[i%len(brandLetterColors)]
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(c).Render(string(r)))
	}
	return b.String()
}

func keyHint(k, desc string) string {
	return styleKey.Render(k) + styleKeyDesc.Render(":"+desc)
}
