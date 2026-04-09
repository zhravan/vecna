package tui

import (
	"reflect"
	"regexp"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// Kitty keyboard protocol: optional push/pop so Cmd/Win (super) + digit can arrive as CSI u sequences.
// See https://sw.kovidgoyal.net/kitty/keyboard-protocol/
//
// Default is OFF: iTerm2 and several other terminals still mishandle CSI >Pu (even without flag 4),
// so keys are reported in a way Bubble Tea v1 does not decode → no KeyMsg, “dead keyboard”.
// Enable only where you need ⌘/Win+1–9 tab jump and the terminal supports this stack (e.g. Kitty,
// Ghostty, WezTerm, often VS Code integrated terminal).
//
// Must NOT use flag 4 (report all keys as escape codes): that breaks normal KeyMsg delivery.
// Flags here are 1|2|8 = 11 (disambiguate, alternate, text).
const (
	EnvKittyKeyboard = "VECNA_KITTY_KEYBOARD"
	kittyKeyboardPushFlags = "\x1b[>11u"
	kittyKeyboardPop       = "\x1b[<u"
)

// modSuper is bit 3 in kitty modifier encoding (Command on macOS, Windows key on many layouts).
const modSuper = 8

// kittyCSIKeyMods matches CSI u key reports; optional :ev is kitty event type (1=press, 2=repeat, 3=release).
var kittyCSIKeyMods = regexp.MustCompile(`^\x1b\[([0-9]+);([0-9]+)(?::([0-9]+))?u$`)

// tabIndexFromKittyUnknownCSI parses Bubble Tea's unexported unknownCSISequenceMsg for super+digit (cmd/Win+1–9).
func tabIndexFromKittyUnknownCSI(msg tea.Msg) (idx int, ok bool) {
	t := reflect.TypeOf(msg)
	if t == nil || t.String() != "tea.unknownCSISequenceMsg" {
		return 0, false
	}
	v := reflect.ValueOf(msg)
	if v.Kind() != reflect.Slice || v.Type().Elem().Kind() != reflect.Uint8 {
		return 0, false
	}
	return parseKittySuperDigitTab(v.Bytes())
}

func parseKittySuperDigitTab(b []byte) (idx int, ok bool) {
	m := kittyCSIKeyMods.FindSubmatch(b)
	if m == nil {
		return 0, false
	}
	if len(m) >= 4 && len(m[3]) > 0 && string(m[3]) == "3" {
		return 0, false
	}
	keyCode, errK := strconv.Atoi(string(m[1]))
	mods, errM := strconv.Atoi(string(m[2]))
	if errK != nil || errM != nil {
		return 0, false
	}
	if mods&modSuper == 0 {
		return 0, false
	}
	if keyCode < '1' || keyCode > '9' {
		return 0, false
	}
	return keyCode - '1', true
}
