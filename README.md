# Ticketsmith

Turn a few rough lines into a properly formatted ticket. Ticketsmith is a lightweight, cross-platform desktop utility that uses AI to draft Bugs, User Stories, and Tasks from your own templates, then files them straight into your project management tool via its API.

## Highlights

- **Provider-agnostic** — the tracker (OpenProject first; Jira and Azure DevOps planned) and the AI backend (Groq by default, any OpenAI-compatible endpoint) both sit behind clean interfaces, so neither is hardcoded into the core logic.
- **Your own templates** — define ticket types, extraction fields, and AI instructions yourself; nothing is baked into the binary.
- **Per-user attribution** — everyone supplies their own tracker API token, so tickets are attributed to whoever actually ran the tool, no shared service account.
- **Preview before you file** — every AI-generated ticket is editable before submission, and every generate/create/edit/error is logged and viewable in-app.
- **One codebase, two run modes** — a native desktop app (Wails) for end users, and a headless HTTP server (build-tag gated) for local or containerized development.

## Tech stack

Go backend · Wails v2 desktop shell · React + Tailwind CSS + shadcn/ui frontend · SQLite for local metadata · OS keychain for secrets.

## Installation

### macOS

Ticketsmith's macOS build is signed with a self-signed certificate rather
than a paid Apple Developer ID, so macOS Gatekeeper will show "Apple could
not verify that this app is free of malware" the first time you open it.
This is expected — to open it anyway:

1. Right-click (or Control-click) `Ticketsmith.app` and choose **Open**.
2. Click **Open** in the dialog that appears.

You only need to do this once per machine.

## Status

Early planning / pre-implementation. See [`docs/PLAN.md`](docs/PLAN.md) for the full architecture, data model, and build order.

## License

See [LICENSE](LICENSE).
