package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ilyabrin/disk"
	"github.com/ilyabrin/yad/internal/auth"
)

type screen int

const (
	screenSetup screen = iota
	screenBrowser
	screenTrash
	screenDiskInfo
)

// App is the root Bubbletea model. It owns all screens and routes messages
// between them.
type App struct {
	screen   screen
	setup    SetupModel
	browser  BrowserModel
	trash    TrashModel
	diskInfo DiskInfoModel

	// Written when setup completes; read by main() to persist to config.
	tokenResult *SetupDoneMsg

	width  int
	height int
	client *disk.Client
}

// New creates the App.
//   - client non-nil  → skip setup, open browser directly
//   - client nil      → show OAuth setup screen
//   - oauthCfg        → optional user-supplied OAuth credentials (may be nil)
func New(client *disk.Client, oauthCfg *auth.Config) *App {
	app := &App{}

	if client != nil {
		app.client = client
		app.screen = screenBrowser
		app.browser = NewBrowserModel(client)
		return app
	}

	app.screen = screenSetup
	app.setup = NewSetupModel(oauthCfg)
	return app
}

func (a *App) Init() tea.Cmd {
	if a.screen == screenBrowser {
		return a.browser.Init()
	}
	return a.setup.Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Deliver window size to the active screen
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		a.width, a.height = ws.Width, ws.Height
	}

	// App-level routing messages
	switch m := msg.(type) {
	case SetupDoneMsg:
		a.tokenResult = &m
		client, err := disk.New(m.AccessToken)
		if err != nil {
			// Token rejected - go back to setup
			a.setup = NewSetupModel(nil)
			a.screen = screenSetup
			return a, a.setup.Init()
		}
		a.client = client
		a.screen = screenBrowser
		a.browser = NewBrowserModel(client)
		return a, tea.Batch(
			a.browser.Init(),
			func() tea.Msg { return tea.WindowSizeMsg{Width: a.width, Height: a.height} },
		)

	case switchToBrowserMsg, switchToBrowserFromInfoMsg:
		a.screen = screenBrowser
		return a, func() tea.Msg { return tea.WindowSizeMsg{Width: a.width, Height: a.height} }
	}

	// Delegate to active screen
	switch a.screen {
	case screenSetup:
		var cmd tea.Cmd
		a.setup, cmd = a.setup.Update(msg)
		return a, cmd

	case screenBrowser:
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "t":
				a.screen = screenTrash
				a.trash = NewTrashModel(a.client)
				return a, tea.Batch(
					a.trash.Init(),
					func() tea.Msg { return tea.WindowSizeMsg{Width: a.width, Height: a.height} },
				)
			case "i":
				a.screen = screenDiskInfo
				a.diskInfo = NewDiskInfoModel(a.client)
				return a, tea.Batch(
					a.diskInfo.Init(),
					func() tea.Msg { return tea.WindowSizeMsg{Width: a.width, Height: a.height} },
				)
			}
		}
		var cmd tea.Cmd
		a.browser, cmd = a.browser.Update(msg)
		return a, cmd

	case screenTrash:
		var cmd tea.Cmd
		a.trash, cmd = a.trash.Update(msg)
		return a, cmd

	case screenDiskInfo:
		var cmd tea.Cmd
		a.diskInfo, cmd = a.diskInfo.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a *App) View() string {
	switch a.screen {
	case screenSetup:
		return a.setup.View()
	case screenBrowser:
		return a.browser.View()
	case screenTrash:
		return a.trash.View()
	case screenDiskInfo:
		return a.diskInfo.View()
	}
	return ""
}

// TokenResult returns the OAuth result from a completed setup flow, or nil
// if the user was already authenticated. The caller (main) uses this to
// persist the tokens to config.
func (a *App) TokenResult() *SetupDoneMsg {
	return a.tokenResult
}
