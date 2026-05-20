package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- WindowSizeMsg reaches the active screen ---

func TestApp_WindowSize_ReachesBrowser(t *testing.T) {
	a := newTestApp()
	a.screen = screenBrowser

	model, _ := a.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app := model.(*App)

	if app.browser.width != 100 || app.browser.height != 30 {
		t.Errorf("browser size = %dx%d, want 100x30", app.browser.width, app.browser.height)
	}
}

func TestApp_WindowSize_ReachesTrash(t *testing.T) {
	a := newTestApp()
	a.screen = screenTrash
	a.trash = NewTrashModel(nil)

	model, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app := model.(*App)

	if app.trash.width != 80 || app.trash.height != 24 {
		t.Errorf("trash size = %dx%d, want 80x24", app.trash.width, app.trash.height)
	}
}

func TestApp_WindowSize_ReachesDiskInfo(t *testing.T) {
	a := newTestApp()
	a.screen = screenDiskInfo
	a.diskInfo = NewDiskInfoModel(nil)

	model, _ := a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app := model.(*App)

	if app.diskInfo.width != 80 || app.diskInfo.height != 24 {
		t.Errorf("diskinfo size = %dx%d, want 80x24", app.diskInfo.width, app.diskInfo.height)
	}
}

// --- Messages are routed to the correct screen ---

func TestApp_BrowserReceivesLoadedMsg(t *testing.T) {
	a := newTestApp()
	a.screen = screenBrowser

	msg := loadedMsg{path: "disk:/", entries: makeEntries(5), total: 5}
	model, _ := a.Update(msg)
	app := model.(*App)

	if len(app.browser.entries) != 5 {
		t.Errorf("browser entries = %d, want 5", len(app.browser.entries))
	}
	if app.browser.loading {
		t.Error("browser should not be loading after loadedMsg")
	}
}

func TestApp_TrashReceivesTrashLoadedMsg(t *testing.T) {
	a := newTestApp()
	a.screen = screenTrash
	a.trash = NewTrashModel(nil)

	msg := trashLoadedMsg{items: nil}
	model, _ := a.Update(msg)
	app := model.(*App)

	if app.trash.loading {
		t.Error("trash should not be loading after trashLoadedMsg")
	}
}

// --- Fatal error propagates from sub-screen through App ---

func TestApp_TrashFatalError_SwitchesScreen(t *testing.T) {
	a := newTestApp()
	a.screen = screenTrash
	a.trash = NewTrashModel(nil)

	// Trash receives auth error → emits fatalErrorMsg → App intercepts
	msg := trashLoadedMsg{err: errTest("HTTP 401: Unauthorized")}
	model, cmd := a.Update(msg)
	app := model.(*App)

	// The fatalErrorMsg is returned as a Cmd, not handled in-place,
	// so we need to run the cmd and re-update
	if cmd == nil {
		t.Fatal("expected cmd from trash on auth error")
	}
	fatalMsg := cmd()
	model2, _ := app.Update(fatalMsg)
	app2 := model2.(*App)

	if app2.screen != screenFatalError {
		t.Errorf("screen = %v, want screenFatalError", app2.screen)
	}
}

// --- LastPath ---

func TestApp_LastPath_ReturnsBrowserPath(t *testing.T) {
	a := newTestApp()
	a.screen = screenBrowser
	a.browser.path = "disk:/photos/vacation"

	if got := a.LastPath(); got != "disk:/photos/vacation" {
		t.Errorf("LastPath() = %q, want disk:/photos/vacation", got)
	}
}

func TestApp_LastPath_EmptyOnSetupScreen(t *testing.T) {
	a := newTestApp()
	a.screen = screenSetup

	if got := a.LastPath(); got != "" {
		t.Errorf("LastPath() = %q, want empty on setup screen", got)
	}
}

func TestApp_LastPath_ReturnsBrowserPathOnFatalError(t *testing.T) {
	a := newTestApp()
	a.screen = screenFatalError
	a.browser.path = "disk:/docs"
	a.fatalErr = &fatalErrorMsg{title: "T", body: "B", hint: "H"}

	if got := a.LastPath(); got != "disk:/docs" {
		t.Errorf("LastPath() = %q, want disk:/docs", got)
	}
}

// --- View returns non-empty for each screen ---

func TestApp_View_Browser(t *testing.T) {
	a := newTestApp()
	a.width, a.height = 80, 24
	// browser needs size delivered
	a.browser.width, a.browser.height = 80, 24

	if v := a.View(); v == "" {
		t.Error("View() returned empty string for screenBrowser")
	}
}

func TestApp_View_FatalError(t *testing.T) {
	a := newTestApp()
	a.screen = screenFatalError
	a.fatalErr = &fatalErrorMsg{
		title: "Authentication Error",
		body:  "Token expired.",
		hint:  "Re-run yad.",
	}
	a.width, a.height = 80, 24

	if v := a.View(); v == "" {
		t.Error("View() returned empty string for screenFatalError")
	}
}
