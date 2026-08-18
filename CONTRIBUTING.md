# Contributing to FLACidal

We appreciate you taking the time to contribute!

## Getting Started

1. Fork the repo
2. Clone your fork to your machine
3. Install the required tooling:
   - [Go 1.22+](https://go.dev/dl/)
   - [Node.js 18+](https://nodejs.org/)
   - [Wails v2](https://wails.io/docs/gettingstarted/installation)
4. Launch development mode with `wails dev`

## Project Structure

- **Backend (Go):** `main.go`, `app.go`, `internal/`, `cmd/`
- **Frontend (Svelte + TypeScript):** `frontend/src/`
- **Wails config:** `wails.json`
- **Build output:** `build/bin/`

## Development

- Build: `wails build`
- Hot-reload dev mode: `wails dev`
- Type-check the frontend: `cd frontend && npm run check`
- Run frontend tests: `cd frontend && npm test`

## Pull Requests

- Branch off `main` for your feature
- Keep each change focused and self-contained
- Confirm `wails build` succeeds before you submit
- Explain what changed and why in the description

## Reporting Issues

- File it through GitHub Issues
- Note your OS, the FLACidal version, and how to reproduce it
- If relevant, attach logs from the app's Terminal tab

## Code Style

- Go: stick to standard `gofmt` formatting
- TypeScript/Svelte: follow the conventions already in place
- Commit format: `type: description` (feat, fix, refactor, docs, test, chore)

## License

By submitting a contribution, you agree it will be released under the MIT License.
