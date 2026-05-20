// Package tui implements the terminal UI for YaD (Yandex.Disk TUI).
//
// # Architecture
//
// The root model is [App], which owns all screens and routes Bubbletea
// messages between them. Screens are independent models that implement
// the Bubbletea Model interface (Init / Update / View):
//
//   - [SetupModel]    — OAuth wizard shown on first launch
//   - [BrowserModel]  — main file browser (split across browser_model.go,
//     browser_update.go, browser_view.go)
//   - [TrashModel]    — trash management
//   - [DiskInfoModel] — disk usage overview
//
// # Async operations
//
// All network calls are wrapped in [tea.Cmd] functions defined in ops.go.
// Long-running transfers (upload / download) use a goroutine + channel pair:
// the goroutine is started lazily inside a Cmd (cmdStartUpload /
// cmdStartDownload), which returns an uploadStartedMsg / downloadStartedMsg
// carrying the channels. The model stores those channels and polls them via
// cmdWaitUpload / cmdWaitDownload on every progress tick.
//
// # Keybindings
//
// Browser keybindings are declared in [BrowserKeyMap] (keys.go). Global
// shortcuts (t = trash, i = disk info) are handled in [App.Update]; they
// are suppressed when the browser reports [BrowserModel.IsInputActive].
package tui
