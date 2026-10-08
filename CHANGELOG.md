# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [v1.8.0] - 2026-10-08

### Added

- Protect a shared link: before sharing, `p` asks how long the link should work (forever, 1, 7 or 30 days) and for an optional password. Enter alone shares an ordinary link, as before. If the protection cannot be applied, the file is not left shared

### Fixed

- Custom properties and batch operations, through `ilyabrin/disk` v1.5.0

### Changed

- Dependencies: `ilyabrin/disk` v1.5.0

## [v1.7.1] - 2026-10-08

### Fixed

- Deleting a file showed an error although the file was gone
- Uploading a file that already exists failed instead of replacing it
- Moving a large folder could not follow the move to the end

### Changed

- Dependencies: `ilyabrin/disk` v1.4.0

## [v1.7.0] - 2026-10-07

### Added

- Android, in Termux: an `android-arm64` archive, and a README install that picks it, or `linux-armv7` on 32-bit phones and TV boxes
- Links open in the device's browser on Android, through `termux-open-url`

### Fixed

- On Android, name lookups failed because there is no `/etc/resolv.conf`. yad now reads Termux's own copy, or uses the servers Termux ships with

## [v1.6.0] - 2026-10-07

### Added

- Linux packages: `.deb`, `.rpm` and `.apk` for amd64, arm64, 386, armv6, armv7 and riscv64. They are named `yadisk`, because Debian, Ubuntu and Fedora already ship an unrelated `yad` (Yet Another Dialog); the command is still `yad`, and the package manager asks before replacing the other program
- Homebrew on macOS and Linux: `brew install --cask ilyabrin/tap/yad`
- Scoop on Windows: `scoop bucket add ilyabrin https://github.com/ilyabrin/scoop-bucket`, then `scoop install ilyabrin/yad`

## [v1.5.0] - 2026-10-07

### Added

- Downloads for more computers: Windows on ARM, 32-bit Linux PCs, older Raspberry Pi (`armv6` and `armv7`), RISC-V, and FreeBSD, OpenBSD and NetBSD. Fifteen archives instead of five
- The README tells you which archive to pick, with the `uname -m` values for Linux

### Fixed

- 32-bit builds can read a Disk or a file over 2 GB. Fixed in `ilyabrin/disk` v1.3.0

### Changed

- Dependencies: `ilyabrin/disk` v1.3.0
- CI builds every release target and also runs the tests as 32-bit

## [v1.4.0] - 2026-10-07

### Added

- `YANDEX_DISK_API_URL` points yad at another API address: a proxy, or a local server for tests. Your access token is sent there too, so only use one you trust
- The README opens with a short animation of yad at work, recorded against sample files

### Fixed

- **The trash screen pushed its title off the top.** Each row was two characters wider than the window and wrapped onto a second line
- **The file list lost its title in narrow windows** or while filtering: the key hints at the bottom wrapped onto a second line. Hints now drop from the end until they fit, and "q quit" always stays

## [v1.3.0] - 2026-10-07

### Added

- The sign-in screen shows the link as a QR code when the window is big enough, so it can be opened on a phone. Useful over SSH or anywhere without a browser. The encoder is built in, with no new dependency

### Fixed

- **The trash screen was always empty.** The underlying client read the trash listing from the wrong place in Yandex's response, so "Trash is empty" showed even with files in the trash. Fixed in `ilyabrin/disk` v1.2.3

### Changed

- The README walks through installing step by step: which archive suits which computer, commands for each system including adding `yad` to `PATH` on Windows, and how to check a download against `checksums.txt`
- The README documents that copying a share link on Linux needs `xclip`, `xsel` or `wl-clipboard`, and that sign-in works without a browser, over SSH for example. Troubleshooting is a list of collapsible questions
- Dependencies: `ilyabrin/disk` v1.2.3. `kr/pretty`, `go-spew` and `go-difflib` leave the module graph
- The default branch is now `main`

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

[Unreleased]: https://github.com/ilyabrin/yad/compare/v1.8.0...HEAD
[v1.8.0]: https://github.com/ilyabrin/yad/compare/v1.7.1...v1.8.0
[v1.7.1]: https://github.com/ilyabrin/yad/compare/v1.7.0...v1.7.1
[v1.7.0]: https://github.com/ilyabrin/yad/compare/v1.6.0...v1.7.0
[v1.6.0]: https://github.com/ilyabrin/yad/compare/v1.5.0...v1.6.0
[v1.5.0]: https://github.com/ilyabrin/yad/compare/v1.4.0...v1.5.0
[v1.4.0]: https://github.com/ilyabrin/yad/compare/v1.3.0...v1.4.0
[v1.3.0]: https://github.com/ilyabrin/yad/compare/v1.2.0...v1.3.0
[v1.2.0]: https://github.com/ilyabrin/yad/compare/v1.1.2...v1.2.0
[v1.1.2]: https://github.com/ilyabrin/yad/compare/v1.1.1...v1.1.2
[v1.1.1]: https://github.com/ilyabrin/yad/compare/v0.1.0...v1.1.1
[v0.1.0]: https://github.com/ilyabrin/yad/releases/tag/v0.1.0
