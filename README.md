# Ticketsmith

Turn a few rough lines into a properly formatted ticket. Ticketsmith is a lightweight, cross-platform desktop utility that uses AI to draft Bugs, User Stories, and Tasks from your own templates, then files them straight into your project management tool via its API.

## Features

- **AI ticket generation** — paste freeform text, pick a template, and get a structured, editable ticket. Refine the result with follow-up instructions before filing it, and attach files or clipboard screenshots on the way.
- **Provider-agnostic** — the tracker (OpenProject today; Jira and Azure DevOps planned) and the AI backend (any OpenAI-compatible endpoint, e.g. Groq) both sit behind clean interfaces, so neither is hardcoded into the core logic.
- **Your own templates** — define ticket types, extraction fields, and AI instructions yourself; nothing is baked into the binary. Built-in starter templates are seeded on first launch, and the app can suggest tuning improvements to a template based on how you've edited its output.
- **Quick capture** — a global hotkey (Ctrl+T on Windows) raises the app from anywhere so you can jot down a thought without breaking flow.
- **Notes** — capture quick notes, then later merge them and convert them into tickets, or discard them.
- **Per-user attribution** — everyone supplies their own tracker API token, so tickets are attributed to whoever actually ran the tool, no shared service account.
- **Full audit log** — every generate, refine, create, and error is logged and viewable in-app; a single log entry tracks a ticket from raw input through to the filed result.
- **Secrets stay local** — API tokens live in the OS keychain, never in the database or on disk in plaintext.
- **Built-in updates** — the app checks for new releases, shows release notes, and can download and install updates itself.

## Tech stack

Go backend · Wails v2 desktop shell · React + Tailwind CSS + shadcn/ui frontend · SQLite for local metadata · OS keychain for secrets.

## Installation

Download the latest Windows installer from [GitHub Releases](../../releases/latest). Once installed, the app keeps itself up to date via its built-in updater.

The installer is signed with a self-signed certificate rather than a paid code-signing certificate, so Windows SmartScreen may show "Windows protected your PC" on first run. Click **More info → Run anyway** to proceed.

macOS builds are not currently published — on macOS, build from source (see Development below).

## Development

Prerequisites: Go, Node.js, and the [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation).

```sh
cd frontend && npm install && cd ..
wails dev      # full-stack dev loop with hot reload
wails build    # native binary in build/bin/
go test ./...  # backend tests
```

Optionally, copy `.env.example` to `.env` with your OpenProject URL and API token to auto-seed a "Default" connection on first launch (dev convenience only — connections and AI providers are normally configured in-app on the Connect screen).

## License

See [LICENSE](LICENSE).
