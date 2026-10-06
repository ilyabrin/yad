# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [v1.2.0] - 2026-10-07

### Added

- **Signing in needs no client secret.** It uses PKCE, so a copy built with `go install` or from a clone signs in exactly like a release download. Before, a build from source could only accept a pasted token
- Nested folders in one step: `n` accepts a path such as `trips/2026/iceland` and creates the missing levels
- Live `/` filter by name over the current page
- The current path is shown in the title bar
- Quitting while an upload or download is running asks for confirmation first
- Auto token refresh mid-session: when an access token expires during a long session, `yad` silently refreshes it without requiring a restart
- Default sort order from config (`default_sort` field in `~/.yad/config.yaml`)
- Last visited directory restored on next launch (`last_path` field in config)
- Pagination indicators `▲`/`▼` showing when more items exist above or below the visible page
- Dedicated overdraft error screen with a clear remediation hint when Yandex Disk API is disabled due to storage quota exceeded
- `SECURITY.md`, `CONTRIBUTING.md`, a code of conduct, issue forms and a pull request template

### Fixed

- **Uploads and downloads stopped after 30 seconds.** The underlying client applied its request timeout to the whole transfer, so any file that took longer, roughly anything over 35 MB on a 10 Mbit/s link, was cut off midway. Fixed in `ilyabrin/disk` v1.2.1
- **Upload from URL reported success too early.** Yandex fetches the file in the background, and `yad` said it was done before it was. It now waits for the transfer to finish and reports a failure if Yandex could not fetch the file
- **Actions targeted the wrong file while a filter was active.** The list rendered filtered entries but delete, rename, download, publish, copy-URL and metadata all indexed the unfiltered page, so they operated on a resource the user could not see. All cursor-based actions now resolve through a single filter-aware accessor
- Cursor could move past the end of a filtered list, and `↑`/`↓` triggered page loads while filtering
- `Ctrl+A` selected hidden entries instead of only the visible (filtered) ones
- Filenames and paths containing non-ASCII characters (Cyrillic, CJK, emoji) were truncated mid-rune, producing mojibake in the list, title bar and trash view
- Dialog overlays could slice through ANSI escape sequences of the row behind them, corrupting colours
- Progress callbacks no longer block the transfer goroutine when the UI stops reading (quit mid-transfer)
- `--help` now lists the `m` (metadata) and `R` (refresh) keys
- Inline fallback to `/` when the saved `last_path` no longer exists on Disk (directory deleted remotely)

### Changed

- Release builds no longer embed an OAuth client secret, and none is stored anywhere
- Dependencies updated: `ilyabrin/disk` v1.2.2, the charmbracelet stack, `golang.org/x/*`; Go 1.25
- CI runs on Linux, macOS and Windows with `-race`, coverage, `gofmt`, `go vet`, `go mod tidy` and `golangci-lint` (pinned to v2.13.2), plus a cross-compile matrix
- Dependabot watches GitHub Actions in addition to Go modules, with grouped weekly PRs
- Both READMEs rewritten so people who are not developers can install and use `yad`: download first, Windows instructions, plain-language sign-in steps and a troubleshooting section
- **Licensing clarified: the project is now explicitly dual-licensed `MIT OR Apache-2.0`.** The repository previously shipped an Apache-2.0 `LICENSE` file while both READMEs claimed MIT; there are now `LICENSE-MIT` and `LICENSE-APACHE`, and the copyright placeholder in the Apache text is filled in
- `.gitignore` patterns `*.a**`/`*.b**`/`*.c**`/`*.d**` replaced with explicit rules, because they silently excluded unrelated paths and would have swallowed files like `.codecov.yml`
- Line endings are pinned to LF in the repository

### Removed

- `.yad.yaml`, a leftover from an earlier CLI design describing an `accounts:`/`language:` config format the application never read
- Unused internal commands and message types

## [v1.1.2] - 2026-05-09

### Added

- Upload from URL: press `U` to upload a remote file directly to Yandex Disk by URL, with a two-step prompt (URL then destination filename)
- Bulk download: press `d` with selected items to download multiple files to a local directory
- Published file indicator: files with a public link are highlighted with a green background and `⇡` marker
- Open public link in browser: press `o` in the publish overlay to open the URL directly
- `--version` / `-v` flag

### Fixed

- Sort cycle (name → date → size, ascending/descending) via `s` key

## [v1.1.1]

### Changed

- Bumped `github.com/ilyabrin/disk` dependency to v1.1.1

## [v0.1.0] - initial release

### Added

- Full-screen file browser with pagination and navigation
- Upload and download files with real-time progress bars
- Create directories, rename and delete files (including bulk delete)
- Publish files and copy public URLs to clipboard
- Trash management: restore or permanently delete items, empty trash
- Disk usage info screen with visual storage bar
- OAuth 2.0 guided first-run setup; tokens stored in `~/.yad/config.yaml`
- Token auto-refresh on startup when the stored token has expired
- `YANDEX_DISK_TOKEN` environment variable for CI / scripting
- Support for user-supplied OAuth credentials via config (`oauth.client_id`, `oauth.client_secret`)

[Unreleased]: https://github.com/ilyabrin/yad/compare/v1.2.0...HEAD
[v1.2.0]: https://github.com/ilyabrin/yad/compare/v1.1.2...v1.2.0
[v1.1.2]: https://github.com/ilyabrin/yad/compare/v1.1.1...v1.1.2
[v1.1.1]: https://github.com/ilyabrin/yad/compare/v0.1.0...v1.1.1
[v0.1.0]: https://github.com/ilyabrin/yad/releases/tag/v0.1.0
