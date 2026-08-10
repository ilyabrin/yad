# Security Policy

## Supported versions

Only the latest release receives security fixes. Please upgrade before reporting.

| Version        | Supported |
| -------------- | --------- |
| latest release | ✅        |
| older          | ❌        |

## Reporting a vulnerability

**Please do not open a public issue for security problems.**

Use GitHub's private reporting instead:
[**Report a vulnerability**](https://github.com/ilyabrin/yad/security/advisories/new)

Include what you have: affected version (`yad --version`), OS, reproduction steps, and the impact you believe it has. A proof of concept helps but is not required.

You can expect an acknowledgement within **7 days** and a status update within **30 days**. If a fix is warranted, you will be credited in the release notes unless you prefer otherwise.

## What is in scope

- Leakage of OAuth tokens or the contents of `~/.yad/config.yaml`
- Flaws in the OAuth code exchange or token refresh in [`internal/auth`](internal/auth/)
- Path handling that lets a crafted remote filename write outside the chosen download directory
- Command injection through filenames, paths or public URLs (e.g. the `o` "open in browser" action)
- Anything that causes `yad` to act on a different resource than the one shown to the user

## What is not in scope

- **Extracting the OAuth client secret from a release binary.** It is embedded at build time and is recoverable with `strings`. This is a known and accepted trade-off, [documented in the README](README.md#-security): the secret only protects the token-exchange endpoint and grants no access to user data. Register your own application if this matters to you.
- **The OAuth client ID being public.** It is public by design.
- Vulnerabilities in Yandex.Disk itself — report those to [Yandex](https://yandex.com/bugbounty/).
- Attacks requiring an already-compromised local account. `~/.yad/config.yaml` is written `0600`; anyone who can read it as your user can read your tokens by design.

## Handling your own credentials

`~/.yad/config.yaml` holds live credentials. Do not commit it, sync it to shared storage, or include it in a dotfiles repository. To revoke access, remove the application at [yandex.ru/id/security/applications](https://yandex.ru/id/security/applications) and delete the file.
