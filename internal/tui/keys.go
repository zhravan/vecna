package tui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	Help     key.Binding
	Add      key.Binding
	Delete   key.Binding
	Edit     key.Binding
	Connect  key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	SFTP     key.Binding
	Forward    key.Binding
	RunCommand   key.Binding
	Transfer     key.Binding
	ToggleSelect key.Binding
	Import       key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("⏎", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add host"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Connect: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "connect"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev field"),
		),
		SFTP: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "sftp"),
		),
		Forward: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "port forward"),
		),
		RunCommand: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "run cmd"),
		),
		Transfer: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "transfer"),
		),
		ToggleSelect: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "select"),
		),
		Import: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "import from ssh config"),
		),
	}
}
