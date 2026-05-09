package tui

import "github.com/charmbracelet/bubbles/key"

// BrowserKeyMap defines all keybindings for the file browser.
type BrowserKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding // open dir
	Back      key.Binding // go to parent directory
	Select    key.Binding // toggle selection
	SelectAll key.Binding // select / deselect all
	Upload    key.Binding
	UploadURL key.Binding
	Download  key.Binding
	NewDir    key.Binding
	Delete    key.Binding
	Rename    key.Binding
	Meta      key.Binding // show file metadata
	Sort      key.Binding // cycle sort order
	Publish   key.Binding // toggle public link
	CopyURL   key.Binding // copy public URL to clipboard
	Refresh   key.Binding
	Info      key.Binding // show disk info
	Trash     key.Binding // open trash view
	Quit      key.Binding
}

// DefaultBrowserKeyMap returns the default keybindings.
func DefaultBrowserKeyMap() BrowserKeyMap {
	return BrowserKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter", "right", "l"),
			key.WithHelp("↵/→", "open"),
		),
		Back: key.NewBinding(
			key.WithKeys("backspace", "left", "h"),
			key.WithHelp("←/h", "back"),
		),
		Select: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "select"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "select all"),
		),
		Upload: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "upload"),
		),
		UploadURL: key.NewBinding(
			key.WithKeys("U"),
			key.WithHelp("U", "upload URL"),
		),
		Download: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "download"),
		),
		NewDir: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new dir"),
		),
		Delete: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "delete"),
		),
		Rename: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rename"),
		),
		Meta: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "info"),
		),
		Sort: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort"),
		),
		Publish: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "publish"),
		),
		CopyURL: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy URL"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("R", "ctrl+r"),
			key.WithHelp("R", "refresh"),
		),
		Info: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "disk info"),
		),
		Trash: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "trash"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns keybindings shown in the compact help line.
func (k BrowserKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Back, k.Select, k.Upload, k.Download, k.Delete, k.Quit}
}

// FullHelp returns the full two-column keybinding list.
func (k BrowserKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.Select, k.SelectAll, k.Upload, k.Download},
		{k.NewDir, k.Delete, k.Rename, k.Meta},
		{k.Publish, k.CopyURL, k.Refresh, k.Info},
		{k.Trash, k.Quit},
	}
}
