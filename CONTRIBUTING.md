# Contributing to YaD

Thanks for taking the time. Bug reports, fixes and focused features are all welcome.

> Found a **security** problem? Do not open an issue. See [SECURITY.md](SECURITY.md).

By taking part you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Getting set up

```sh
git clone https://github.com/ilyabrin/yad && cd yad
go test ./...
go run .
```

You need **Go 1.25+**. Nothing else is required to get going.

To work on the app without touching your real Disk, point it at a throwaway
account:

```sh
YANDEX_DISK_TOKEN=<token-for-a-test-account> go run .
```

Signing in needs no client secret, so a build from your own clone authenticates
exactly like a release does. There is nothing to configure first.

## Before you open a pull request

Please run the full set locally before pushing. CI minutes are finite, and
every one of these catches something a reviewer should not have to:

```sh
gofmt -l .                    # must print nothing
go vet ./...
go test -race -cover ./...
go mod tidy                   # must leave go.mod and go.sum unchanged
```

CI also runs `golangci-lint`, pinned to the version in
[`.github/workflows/ci.yml`](.github/workflows/ci.yml) so that a new linter
release cannot turn the build red on its own. To run the same version yourself:

```sh
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./...
```

The linter config lives in [.golangci.yml](.golangci.yml). CI additionally
cross-compiles for Linux, macOS and Windows.

> [!TIP]
> On Windows, run the test command through `bash` or WSL. PowerShell splits
> `-coverprofile=coverage.out` at the dot and `go test` then fails on a package
> called `.out`.

## Commit messages

We use [Conventional Commits](https://www.conventionalcommits.org/), and the
release changelog is generated from them, so the prefix matters:

| Prefix      | Use for                                  | In changelog |
| ----------- | ---------------------------------------- | ------------ |
| `feat:`     | new user-visible capability              | ✅           |
| `fix:`      | bug fix                                  | ✅           |
| `perf:`     | performance improvement                  | ✅           |
| `refactor:` | internal change with no behaviour change | ✅           |
| `docs:`     | documentation only                       | ❌           |
| `test:`     | tests only                               | ✅           |
| `ci:`       | workflows, dependabot, release config    | ❌           |
| `chore:`    | housekeeping                             | ❌           |

Write the subject in the imperative: `fix: clear the filter when leaving a directory`.

## How the TUI is put together

`yad` follows the Elm architecture that [Bubbletea](https://github.com/charmbracelet/bubbletea)
implements:

```text
tui/app.go          root model, owns every screen, routes messages, refreshes tokens
tui/setup.go        first-run sign-in screen (OAuth 2.0 with PKCE)
tui/browser_*.go    file browser, split into model / update / view
tui/trash.go        trash screen
tui/diskinfo.go     disk usage screen
tui/ops.go          async API calls, each returning a tea.Cmd
tui/dialog.go       confirm / input / progress overlays
tui/text.go         width-aware, ANSI-safe truncation
tui/keys.go         keybinding map
tui/styles.go       lipgloss palette
```

Three rules keep this codebase predictable. Please follow them:

1. **`Update` never blocks.** Every network call lives inside a `tea.Cmd` and
   reports back as a message. If you find yourself calling the API from
   `Update`, wrap it in a command instead.
2. **`View` is pure.** It reads the model and returns a string. No mutation,
   no I/O.
3. **Never index `m.entries[m.cursor]`.** The list renders `visibleEntries()`,
   which honours the active `/` filter. Use `currentEntry()`, because indexing
   the raw slice targets a resource the user cannot see. This has caused a
   wrong-file-deleted bug before, and there are regression tests in
   [`tui/browser_filter_test.go`](tui/browser_filter_test.go) guarding it.

The models are passed by value, which is what makes the architecture safe:
every `Update` returns fresh state, so a closure captured inside a `tea.Cmd`
can never observe a later mutation. Please keep it that way. The copies are
measured in the low hundreds of nanoseconds and are not worth optimising.

New keybindings go in `keys.go`, get handled in the relevant `handleKey`, and
must be added to **three** places that users read: the status bar in
`browser_view.go`, `helpText` in [`yad.go`](yad.go), and the keybinding tables
in both READMEs.

## Text and layout

Terminal width is measured in **display cells**, not bytes or runes.
Yandex.Disk filenames are frequently Cyrillic, so never slice strings to fit a
column. Use `truncateRight` and `truncateLeft` from
[`tui/text.go`](tui/text.go).

Every piece of text the user sees comes from a translation file, never from
a string in the code. Call `i18n.T("trash.is_empty")`, `i18n.F` to format,
or `i18n.N` for a count, and add the key to every file in
[`internal/i18n/locales`](internal/i18n/locales). Keys must be written out as
literals: the tests read them from the code.

## Translating YaD

Translations live in [`internal/i18n/locales`](internal/i18n/locales), one
YAML file per language, named by its code: `en.yaml`, `ru.yaml`. They are
built into the binary, so users never install anything extra. `en.yaml` is
the source; whatever a translation lacks is shown in English.

To add a language:

1. Copy `en.yaml` to `<code>.yaml`, for example `uk.yaml` or `kk.yaml`.
2. Translate the values. Keep the keys, the `%d`, `%s` and `%q` in the same
   order, and the spaces at the start or end of a value: they separate a key
   from its hint in the status bar.
3. Counts have plural forms. English needs `one` and `other`. Russian,
   Ukrainian and Belarusian need `one`, `few` and `many`. A language that
   counts differently needs its rule in `pluralCategory` in
   [`internal/i18n/i18n.go`](internal/i18n/i18n.go).
4. Run `go test ./...`. It fails on a missing or extra key, a changed `%`
   verb, or a missing plural form, and the layout tests check that every
   screen still fits a narrow window.

YaD picks the language from `YAD_LANG`, then `language:` in the config, then
`LC_ALL`, `LC_MESSAGES` and `LANG`, then the Windows display language.
Try yours with `YAD_LANG=<code> yad`.

## Documentation

The project ships English and Russian READMEs. If you change one, please change
the other, or say in the PR that you could not and it will be handled.

Two conventions for anything you write in Markdown:

- **No em dashes.** They disrupt the layout GitHub renders and pull attention
  to themselves. A comma, a colon or a full stop reads better.
- **Keep it readable for non-developers.** YaD is for anyone comfortable in a
  terminal, not only for programmers. The README should stay that way, with
  contributor material kept separate.

Markdown is checked with `markdownlint`, configured in
[.markdownlint.json](.markdownlint.json):

```sh
npx markdownlint-cli *.md
```

## Licence

By contributing you agree that your work is dual-licensed under
**MIT OR Apache-2.0**, matching the project.
