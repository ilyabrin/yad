# Contributing to YaD

Thanks for taking the time. Bug reports, fixes and focused features are all welcome.

> Found a **security** problem? Do not open an issue — see [SECURITY.md](SECURITY.md).

## Getting set up

```sh
git clone https://github.com/ilyabrin/yad && cd yad
go test ./...
go run .
```

You need **Go 1.25+**. No other tooling is required; `golangci-lint` is optional locally (CI runs it for you).

To work on the app without touching your real Disk, point it at a throwaway account:

```sh
YANDEX_DISK_TOKEN=<token-for-a-test-account> go run .
```

## Before you open a pull request

Run what CI runs:

```sh
gofmt -l .                    # must print nothing
go vet ./...
go test -race -cover ./...
```

CI additionally runs `golangci-lint` (config in [.golangci.yml](.golangci.yml)), checks that `go mod tidy` produces no diff, and cross-compiles for Linux, macOS and Windows.

## Commit messages

We use [Conventional Commits](https://www.conventionalcommits.org/) — the release changelog is generated from them, so the prefix matters:

| Prefix      | Use for                                        | In changelog |
| ----------- | ---------------------------------------------- | ------------ |
| `feat:`     | new user-visible capability                    | ✅            |
| `fix:`      | bug fix                                        | ✅            |
| `perf:`     | performance improvement                        | ✅            |
| `refactor:` | internal change with no behaviour change        | ✅            |
| `docs:`     | documentation only                             | ❌            |
| `test:`     | tests only                                     | ✅            |
| `ci:`       | workflows, dependabot, release config           | ❌            |
| `chore:`    | housekeeping                                   | ❌            |

Write the subject in the imperative: `fix: clear the filter when leaving a directory`.

## How the TUI is put together

`yad` follows the Elm architecture that [Bubbletea](https://github.com/charmbracelet/bubbletea) implements:

```
tui/app.go          root model — owns every screen, routes messages, handles token refresh
tui/setup.go        first-run OAuth wizard
tui/browser_*.go    file browser, split into model / update / view
tui/trash.go        trash screen
tui/diskinfo.go     disk usage screen
tui/ops.go          async API calls, each returning a tea.Cmd
tui/dialog.go       confirm / input / progress overlays
tui/text.go         width-aware, ANSI-safe truncation
tui/keys.go         keybinding map
tui/styles.go       lipgloss palette
```

Three rules keep this codebase predictable — please follow them:

1. **`Update` never blocks.** Every network call lives inside a `tea.Cmd` and reports back as a message. If you find yourself calling the API from `Update`, wrap it in a command instead.
2. **`View` is pure.** It reads the model and returns a string. No mutation, no I/O.
3. **Never index `m.entries[m.cursor]`.** The list renders `visibleEntries()`, which honours the active `/` filter. Use `currentEntry()` — indexing the raw slice targets a resource the user cannot see. This has caused a wrong-file-deleted bug before; there are regression tests in [`tui/browser_filter_test.go`](tui/browser_filter_test.go) guarding it.

New keybindings go in `keys.go`, get handled in the relevant `handleKey`, and must be added to **three** places that users read: the status bar in `browser_view.go`, `helpText` in [`yad.go`](yad.go), and the keybinding tables in both READMEs.

## Text and layout

Terminal width is measured in **display cells**, not bytes or runes. Yandex.Disk filenames are frequently Cyrillic, so never slice strings to fit a column — use `truncateRight` / `truncateLeft` from [`tui/text.go`](tui/text.go).

## Translations

The project ships English and Russian READMEs. If you change one, please change the other — or say in the PR that you could not, and it will be handled.

## Licence

By contributing you agree that your work is dual-licensed under **MIT OR Apache-2.0**, matching the project.
