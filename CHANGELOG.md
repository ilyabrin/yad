# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- Auto token refresh mid-session — when an access token expires during a long session, `yad` silently refreshes it without requiring a restart
- Default sort order from config (`default_sort` field in `~/.yad/config.yaml`)
- Last visited directory restored on next launch (`last_path` field in config)
- Pagination indicators `▲`/`▼` showing when more items exist above or below the visible page
- Dedicated overdraft error screen with a clear remediation hint when Yandex Disk API is disabled due to storage quota exceeded

### Fixed

- Inline fallback to `/` when the saved `last_path` no longer exists on Disk (directory deleted remotely)

## [v1.1.2] — 2026-05-09

### Added

- Upload from URL — press `U` to upload a remote file directly to Yandex Disk by URL, with a two-step prompt (URL then destination filename)
- Bulk download — press `d` with selected items to download multiple files to a local directory
- Published file indicator — files with a public link are highlighted with a green background and `⇡` marker
- Open public link in browser — press `o` in the publish overlay to open the URL directly
- `--version` / `-v` flag

### Fixed

- Sort cycle (name → date → size, ascending/descending) via `s` key

## [v1.1.1] — 2026-04-xx

### Changed

- Bumped `github.com/ilyabrin/disk` dependency to v1.1.1

## [v0.1.0] — initial release

### Added

- Full-screen file browser with pagination and navigation
- Upload and download files with real-time progress bars
- Create directories, rename and delete files (including bulk delete)
- Publish files and copy public URLs to clipboard
- Trash management — restore or permanently delete items, empty trash
- Disk usage info screen with visual storage bar
- OAuth 2.0 guided first-run setup; tokens stored in `~/.yad/config.yaml`
- Token auto-refresh on startup when the stored token has expired
- `YANDEX_DISK_TOKEN` environment variable for CI / scripting
- Support for user-supplied OAuth credentials via config (`oauth.client_id`, `oauth.client_secret`)

[Unreleased]: https://github.com/ilyabrin/yad/compare/v1.1.2...HEAD
[v1.1.2]: https://github.com/ilyabrin/yad/compare/v1.1.1...v1.1.2
[v1.1.1]: https://github.com/ilyabrin/yad/compare/v0.1.0...v1.1.1
[v0.1.0]: https://github.com/ilyabrin/yad/releases/tag/v0.1.0
