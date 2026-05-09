# YaD - Yandex.Disk TUI

> [Русская версия](README.ru.md)

A terminal UI for [Yandex.Disk](https://disk.yandex.ru) built with [Bubbletea](https://github.com/charmbracelet/bubbletea).

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/github/license/ilyabrin/yad)

## Features

- Full-screen file browser - navigate, open directories
- Upload and download files with real-time progress bars
- Create directories, rename and delete files
- Publish files and copy public URLs to clipboard
- Trash management - restore or permanently delete items
- Disk usage info - storage breakdown with a visual bar
- OAuth 2.0 authentication - guided first-run setup, tokens stored locally
- Token auto-refresh - silent background refresh when the token expires

## Installation

### go install

```sh
go install github.com/ilyabrin/yad@latest
```

> The binary built this way uses manual token paste for authentication (no client secret embedded). See [Authentication](#authentication) for details.

### Build from source

```sh
git clone https://github.com/ilyabrin/yad
cd yad
go build -o yad .
```

### Release build (with OAuth client secret)

Official releases embed the client secret at build time so the full OAuth code-exchange flow works automatically:

```sh
go build \
  -ldflags "-X github.com/ilyabrin/yad/internal/auth.clientSecret=YOUR_SECRET" \
  -o yad .
```

## Authentication

On first launch `yad` starts a setup wizard:

1. A Yandex OAuth URL is displayed and opened in your default browser.
2. You authorise the app and Yandex shows you a short code on screen.
3. Paste the code into the terminal - `yad` exchanges it for tokens and saves them to `~/.yad/config.yaml`.

From that point on tokens are loaded automatically. If the access token expires, `yad` refreshes it silently using the stored refresh token.

### Manual token (dev builds / CI)

If no client secret is available, the wizard asks you to paste a long-lived token directly. Alternatively, set the environment variable:

```sh
export YANDEX_DISK_TOKEN=your_token_here
yad
```

The environment variable always takes priority over the stored token - useful for scripts and CI.

## Configuration

Config is stored at `~/.yad/config.yaml` (mode `0600`):

```yaml
# Stored automatically after OAuth setup - do not edit manually
access_token: "..."
refresh_token: "..."
token_expiry: "2026-06-01T12:00:00Z"

# Optional: use your own registered Yandex application
oauth:
  client_id: "your_client_id"
  client_secret: "your_client_secret"
```

To register your own application visit [oauth.yandex.ru](https://oauth.yandex.ru) and request the `cloud_api:disk.read` and `cloud_api:disk.write` scopes.

## Keybindings

### File browser

| Key                     | Action                                         |
| ----------------------- | ---------------------------------------------- |
| `↑` / `k`               | Move up                                        |
| `↓` / `j`               | Move down                                      |
| `↵` / `→` / `l`         | Open directory                                 |
| `←` / `h` / `Backspace` | Go to parent                                   |
| `Space`                 | Toggle selection                               |
| `Ctrl+A`                | Select all / deselect all                      |
| `Esc`                   | Clear selection                                |
| `u`                     | Upload local file (2-step: path then filename) |
| `U`                     | Upload from URL (2-step: URL then filename)    |
| `d`                     | Download file (bulk if items selected)         |
| `n`                     | Create new directory                           |
| `r`                     | Rename selected item                           |
| `D`                     | Delete (bulk if items selected)                |
| `s`                     | Cycle sort: name / date / size, asc and desc   |
| `p`                     | Publish / show public link                     |
| `c`                     | Copy public URL to clipboard                   |
| `m`                     | Show file metadata                             |
| `R` / `Ctrl+R`          | Refresh listing                                |
| `i`                     | Disk usage info                                |
| `t`                     | Open trash                                     |
| `q` / `Ctrl+C`          | Quit                                           |

Published files are highlighted with a green background and a `⇡` marker.
Pressing `p` on a published file shows the link with options: `c` copy, `u` unpublish.

### Trash

| Key               | Action                           |
| ----------------- | -------------------------------- |
| `↑` / `k`         | Move up                          |
| `↓` / `j`         | Move down                        |
| `r`               | Restore selected item            |
| `D`               | Permanently delete selected item |
| `E`               | Empty trash (delete all)         |
| `R` / `Ctrl+R`    | Refresh                          |
| `q` / `←` / `Esc` | Back to browser                  |

### Disk info

| Key               | Action          |
| ----------------- | --------------- |
| `q` / `←` / `Esc` | Back to browser |

## Security note

The OAuth client secret is injected at release build time via `-ldflags` and lives only in the compiled binary. This is standard practice for open-source CLI tools (see: `gh`, `heroku`). The secret protects the token exchange endpoint - it does not grant access to any user data. If you prefer full control, register your own Yandex application and supply credentials via `~/.yad/config.yaml`.

## Dependencies

- [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) - UI components
- [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) - terminal styling
- [ilyabrin/disk](https://github.com/ilyabrin/disk) - Yandex.Disk API client
- [atotto/clipboard](https://github.com/atotto/clipboard) - clipboard access

## License

MIT
