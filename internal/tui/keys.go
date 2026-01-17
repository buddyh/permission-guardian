package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keyboard bindings for the TUI
type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Approve       key.Binding
	ApproveAlways key.Binding
	Deny          key.Binding
	Refresh       key.Binding
	ToggleAuto    key.Binding
	ToggleView    key.Binding
	ViewLog       key.Binding
	Preview       key.Binding
	Quit          key.Binding
}

// Keys is the default key binding configuration
var Keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Approve: key.NewBinding(
		key.WithKeys("a", "y", "enter"),
		key.WithHelp("a", "approve"),
	),
	ApproveAlways: key.NewBinding(
		key.WithKeys("s", "A"),
		key.WithHelp("s", "approve+remember"),
	),
	Deny: key.NewBinding(
		key.WithKeys("d", "n", "escape"),
		key.WithHelp("d", "deny"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	ToggleAuto: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "toggle auto"),
	),
	ToggleView: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "toggle view"),
	),
	ViewLog: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "log"),
	),
	Preview: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "preview"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
