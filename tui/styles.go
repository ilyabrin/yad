package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Base palette
	colorPrimary  = lipgloss.Color("#7C3AED") // violet
	colorAccent   = lipgloss.Color("#A78BFA") // light violet
	colorMuted    = lipgloss.Color("#6B7280") // gray
	colorSuccess  = lipgloss.Color("#10B981") // green
	colorDanger   = lipgloss.Color("#EF4444") // red
	colorSelected = lipgloss.Color("#1E1B4B") // deep indigo bg for selected row

	// List item
	StyleItemNormal = lipgloss.NewStyle().
			PaddingLeft(1)

	StyleItemSelected = lipgloss.NewStyle().
				PaddingLeft(1).
				Background(colorSelected).
				Foreground(colorAccent).
				Bold(true)

	// Directory entries are displayed in a different color
	StyleDir = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	StyleFile = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	StyleSize = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(9).
			Align(lipgloss.Right)

	StyleDate = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(17)

	// Title bar
	StyleTitle = lipgloss.NewStyle().
			Background(colorPrimary).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 1)

	StylePath = lipgloss.NewStyle().
			Foreground(colorAccent).
			Padding(0, 1)

	// Status bar (bottom)
	StyleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#111827")).
			Foreground(colorMuted).
			Padding(0, 1)

	StyleStatusKey = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	// Dialogs / overlays
	StyleDialog = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	StyleDialogTitle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true).
				MarginBottom(1)

	StyleError = lipgloss.NewStyle().
			Foreground(colorDanger)

	StyleSuccess = lipgloss.NewStyle().
			Foreground(colorSuccess)

	// Progress bar
	StyleProgressFull  = lipgloss.NewStyle().Background(colorPrimary)
	StyleProgressEmpty = lipgloss.NewStyle().Background(lipgloss.Color("#374151"))
)
