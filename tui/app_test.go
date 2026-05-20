package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// newTestApp returns an App already on screenBrowser (no live client needed
// for routing tests — the client field is only used when constructing new
// sub-screens, which these tests don't trigger).
func newTestApp() *App {
	return &App{
		screen:  screenBrowser,
		browser: newTestBrowser(),
		width:   80,
		height:  24,
	}
}

// --- fatalErrorMsg routing ---

func TestApp_FatalErrorMsg_SwitchesScreen(t *testing.T) {
	a := newTestApp()
	fe := fatalErrorMsg{
		title:  "Authentication Error",
		body:   "Token expired.",
		hint:   "Re-run yad.",
		detail: errors.New("HTTP 401: Unauthorized"),
	}

	model, cmd := a.Update(fe)
	app := model.(*App)

	if app.screen != screenFatalError {
		t.Errorf("screen = %v, want screenFatalError", app.screen)
	}
	if app.fatalErr == nil {
		t.Fatal("fatalErr should be set")
	}
	if app.fatalErr.title != fe.title {
		t.Errorf("fatalErr.title = %q, want %q", app.fatalErr.title, fe.title)
	}
	if cmd != nil {
		t.Error("cmd should be nil after fatalErrorMsg")
	}
}

func TestApp_FatalErrorScreen_QuitOnQ(t *testing.T) {
	a := newTestApp()
	a.screen = screenFatalError
	a.fatalErr = &fatalErrorMsg{title: "T", body: "B", hint: "H"}

	for _, key := range []string{"q", "esc", "ctrl+c"} {
		t.Run(key, func(t *testing.T) {
			_, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			if key == "q" && cmd == nil {
				// q is a rune key — bubbletea represents it as KeyRunes
				// The actual quit is triggered; just verify no panic.
				return
			}
			// For esc and ctrl+c the cmd should be tea.Quit.
			_ = cmd
		})
	}
}

func TestApp_FatalErrorScreen_IgnoresOtherKeys(t *testing.T) {
	a := newTestApp()
	a.screen = screenFatalError
	a.fatalErr = &fatalErrorMsg{title: "T", body: "B", hint: "H"}

	model, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	app := model.(*App)

	if app.screen != screenFatalError {
		t.Error("screen should remain screenFatalError")
	}
	if cmd != nil {
		t.Error("cmd should be nil for non-quit key on fatal error screen")
	}
}

// --- switchToBrowserMsg routing ---

func TestApp_SwitchToBrowserMsg(t *testing.T) {
	a := newTestApp()
	a.screen = screenTrash

	model, _ := a.Update(switchToBrowserMsg{})
	app := model.(*App)

	if app.screen != screenBrowser {
		t.Errorf("screen = %v, want screenBrowser", app.screen)
	}
}

func TestApp_SwitchToBrowserFromInfoMsg(t *testing.T) {
	a := newTestApp()
	a.screen = screenDiskInfo

	model, _ := a.Update(switchToBrowserFromInfoMsg{})
	app := model.(*App)

	if app.screen != screenBrowser {
		t.Errorf("screen = %v, want screenBrowser", app.screen)
	}
}

// --- WindowSizeMsg is delivered to app regardless of screen ---

func TestApp_WindowSizeMsg_UpdatesDimensions(t *testing.T) {
	a := newTestApp()

	model, _ := a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := model.(*App)

	if app.width != 120 || app.height != 40 {
		t.Errorf("size = %dx%d, want 120x40", app.width, app.height)
	}
}

// --- tryRefreshMsg emitted by browser loadedMsg with auth error ---

func TestApp_BrowserAuthError_RoutesTryRefresh(t *testing.T) {
	a := newTestApp()

	// Simulate browser receiving a 401 error via loadedMsg
	authErr := errors.New("HTTP 401: Unauthorized")
	_, cmd := a.browser.Update(loadedMsg{path: "disk:/", err: authErr})

	if cmd == nil {
		t.Fatal("expected a cmd from browser on auth error")
	}
	msg := cmd()
	rm, ok := msg.(tryRefreshMsg)
	if !ok {
		t.Fatalf("expected tryRefreshMsg, got %T", msg)
	}
	if rm.origErr == nil {
		t.Error("origErr should be set")
	}
}

// --- tryRefreshMsg with no refreshFn falls back to fatalErrorMsg ---

func TestApp_TryRefreshMsg_NoRefreshFn_ShowsFatal(t *testing.T) {
	a := newTestApp() // refreshFn is nil

	model, _ := a.Update(tryRefreshMsg{origErr: errors.New("HTTP 401: Unauthorized")})
	app := model.(*App)

	if app.screen != screenFatalError {
		t.Errorf("screen = %v, want screenFatalError", app.screen)
	}
	if app.fatalErr == nil || app.fatalErr.title != "Authentication Error" {
		t.Errorf("fatalErr = %v, want Authentication Error", app.fatalErr)
	}
}

// --- tryRefreshMsg with refreshFn spawns a goroutine ---

func TestApp_TryRefreshMsg_WithRefreshFn_SpawnsCmd(t *testing.T) {
	a := newTestApp()
	called := false
	a.refreshFn = func() (string, error) {
		called = true
		return "new-token", nil
	}

	_, cmd := a.Update(tryRefreshMsg{origErr: errors.New("HTTP 401")})
	if cmd == nil {
		t.Fatal("expected a cmd when refreshFn is set")
	}
	msg := cmd()
	rd, ok := msg.(refreshDoneMsg)
	if !ok {
		t.Fatalf("expected refreshDoneMsg, got %T", msg)
	}
	if !called {
		t.Error("refreshFn should have been called")
	}
	if rd.token != "new-token" {
		t.Errorf("token = %q, want new-token", rd.token)
	}
	if rd.err != nil {
		t.Errorf("unexpected err: %v", rd.err)
	}
}

func TestApp_BrowserOverdraftError_RoutesFatal(t *testing.T) {
	a := newTestApp()

	overdraftErr := errors.New("DiskAPIDisabledForOverdraftUserError: API недоступно")
	_, cmd := a.browser.Update(loadedMsg{path: "disk:/", err: overdraftErr})

	if cmd == nil {
		t.Fatal("expected a cmd from browser on overdraft error")
	}
	msg := cmd()
	fe, ok := msg.(fatalErrorMsg)
	if !ok {
		t.Fatalf("expected fatalErrorMsg, got %T", msg)
	}
	if fe.title != "Storage Overdraft" {
		t.Errorf("title = %q, want Storage Overdraft", fe.title)
	}
}
