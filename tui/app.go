package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ilyabrin/disk"
	"github.com/ilyabrin/yad/internal/auth"
	"github.com/ilyabrin/yad/internal/i18n"
)

type screen int

const (
	screenSetup screen = iota
	screenBrowser
	screenTrash
	screenDiskInfo
	screenFatalError
)

// fatalErrorMsg is emitted by any screen when an unrecoverable API error occurs
// (expired token, storage overdraft, etc.). App intercepts it, switches to the
// fatal error screen, and only allows the user to quit.
type fatalErrorMsg struct {
	title  string // screen subtitle, e.g. "Authentication Error"
	body   string // primary message shown in the dialog
	hint   string // remediation instruction shown below the body
	detail error  // raw API error, shown in muted style (may be nil)
}

// tryRefreshMsg is emitted when a 401/403 API error is detected mid-session.
// App intercepts it and attempts a silent token refresh via refreshFn.
type tryRefreshMsg struct{ origErr error }

// refreshDoneMsg is the result of a token refresh attempt.
type refreshDoneMsg struct {
	token string
	err   error
}

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

	fatalErr  *fatalErrorMsg         // set when a fatal error is received from any screen
	refreshFn func() (string, error) // nil when no refresh is possible (env token / no refresh token)

	width  int
	height int
	client *disk.Client

	// languageChosen is set when the user switches the language with L,
	// so main() saves it to the config.
	languageChosen bool
}

// New creates the App.
//   - client non-nil  → skip setup, open browser directly
//   - client nil      → show OAuth setup screen
//   - oauthCfg        → optional user-supplied OAuth credentials (may be nil)
//   - defaultSort     → initial sort order (e.g. "-modified"); empty → "name"
//   - lastPath        → directory to open on start; empty → "/"
//   - refreshFn       → called when a mid-session 401 is detected; nil disables silent refresh
func New(client *disk.Client, oauthCfg *auth.Config, defaultSort, lastPath string, refreshFn func() (string, error)) *App {
	app := &App{refreshFn: refreshFn}

	if client != nil {
		app.client = client
		app.screen = screenBrowser
		app.browser = NewBrowserModel(client, defaultSort, lastPath)
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
	case fatalErrorMsg:
		a.fatalErr = &m
		a.screen = screenFatalError
		return a, nil

	case tryRefreshMsg:
		if a.refreshFn == nil {
			return a.goFatalAuth(m.origErr)
		}
		refreshFn := a.refreshFn
		return a, func() tea.Msg {
			token, err := refreshFn()
			return refreshDoneMsg{token: token, err: err}
		}

	case refreshDoneMsg:
		if m.err != nil {
			return a.goFatalAuth(m.err)
		}
		newClient, err := NewClient(m.token)
		if err != nil {
			return a.goFatalAuth(err)
		}
		a.client = newClient
		a.browser.setClient(newClient)
		a.trash.setClient(newClient)
		a.diskInfo.setClient(newClient)
		switch a.screen {
		case screenTrash:
			return a, a.trash.Init()
		case screenDiskInfo:
			return a, a.diskInfo.Init()
		default:
			a.browser.loading = true
			return a, a.browser.reloadCmd()
		}

	case SetupDoneMsg:
		a.tokenResult = &m
		client, err := NewClient(m.AccessToken)
		if err != nil {
			// Token rejected - go back to setup
			a.setup = NewSetupModel(nil)
			a.screen = screenSetup
			return a, a.setup.Init()
		}
		a.client = client
		a.screen = screenBrowser
		a.browser = NewBrowserModel(client, "", "")
		return a, tea.Batch(
			a.browser.Init(),
			func() tea.Msg { return tea.WindowSizeMsg{Width: a.width, Height: a.height} },
		)

	case switchToBrowserMsg, switchToBrowserFromInfoMsg:
		a.screen = screenBrowser
		return a, func() tea.Msg { return tea.WindowSizeMsg{Width: a.width, Height: a.height} }
	}

	// Hotkeys work in a Cyrillic layout too, except where the user types
	// text: the setup screen's code field and the browser's text fields.
	if km, ok := msg.(tea.KeyMsg); ok && a.screen != screenSetup &&
		(a.screen != screenBrowser || !a.browser.TypingText()) {
		msg = latinKey(km)
	}

	// Fatal error screen: only q/esc/ctrl+c to quit
	if a.screen == screenFatalError {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "q", "esc", "ctrl+c":
				return a, tea.Quit
			}
		}
		return a, nil
	}

	// Delegate to active screen
	switch a.screen {
	case screenSetup:
		var cmd tea.Cmd
		a.setup, cmd = a.setup.Update(msg)
		return a, cmd

	case screenBrowser:
		if km, ok := msg.(tea.KeyMsg); ok && !a.browser.IsInputActive() {
			switch km.String() {
			case "t":
				a.screen = screenTrash
				a.trash = NewTrashModel(a.client)
				return a, tea.Batch(
					a.trash.Init(),
					func() tea.Msg { return tea.WindowSizeMsg{Width: a.width, Height: a.height} },
				)
			case "L":
				i18n.Set(i18n.Next())
				a.languageChosen = true
				// Built when the browser was; everything else is drawn anew.
				a.browser.filterInput.Placeholder = i18n.T("browser.filter_placeholder")
				if os.Getenv("YAD_LANG") != "" {
					a.browser = a.browser.showMessage(i18n.T("app.language_env_overrides"), false)
				}
				return a, nil
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
	case screenFatalError:
		return a.viewFatalError()
	}
	return ""
}

func (a *App) viewFatalError() string {
	fe := a.fatalErr
	title := StyleTitle.Render("  YaD  ·  " + fe.title)
	body := StyleError.Render(iconErr + " " + fe.body)
	detail := ""
	if fe.detail != nil {
		detail = "\n" + StyleMuted(fe.detail.Error())
	}
	hint := "\n\n" + StyleMuted(fe.hint) +
		"\n\n" + StyleStatusKey.Render("q") + StyleMuted(" quit")
	content := StyleDialog.Render(body + detail + hint)
	inner := lipgloss.PlaceHorizontal(a.width, lipgloss.Center,
		lipgloss.Place(a.width, a.height-2, lipgloss.Center, lipgloss.Center, content))
	return title + "\n" + inner
}

func (a *App) goFatalAuth(err error) (tea.Model, tea.Cmd) {
	a.fatalErr = authFatalMsg(err)
	a.screen = screenFatalError
	return a, nil
}

// TokenResult returns the OAuth result from a completed setup flow, or nil
// if the user was already authenticated. The caller (main) uses this to
// persist the tokens to config.
func (a *App) TokenResult() *SetupDoneMsg {
	return a.tokenResult
}

// LastPath returns the directory the browser was in when the app exited.
// Empty when the browser was never shown (e.g. setup screen).
// ChosenLanguage is the language the user switched to with L, for main()
// to save, or "" when they did not switch.
func (a *App) ChosenLanguage() string {
	if !a.languageChosen {
		return ""
	}
	return i18n.Current().String()
}

func (a *App) LastPath() string {
	if a.screen == screenBrowser || a.screen == screenFatalError {
		return a.browser.path
	}
	return ""
}
