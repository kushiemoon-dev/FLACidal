# Contributing to FLACidal

Thanks for your interest in contributing!

## Getting Started

1. Fork the repository
2. Clone your fork locally
3. Install dependencies:
   - [Go 1.22+](https://go.dev/dl/)
   - [Node.js 18+](https://nodejs.org/)
   - [Wails v2](https://wails.io/docs/gettingstarted/installation)
4. Run `wails dev` to start in development mode

## Project Structure

- **Backend (Go):** `main.go`, `app.go`, `internal/`, `cmd/`
- **Frontend (Svelte + TypeScript):** `frontend/src/`
- **Wails config:** `wails.json`
- **Build output:** `build/bin/`

## Development

- Build: `wails build`
- Dev mode with hot reload: `wails dev`
- Frontend type-check: `cd frontend && npm run check`
- Frontend tests: `cd frontend && npm test`

## Pull Requests

- Create a feature branch from `main`
- Keep changes focused and atomic
- Ensure `wails build` passes before submitting
- Describe what changed and why in the PR description

## Reporting Issues

- Use GitHub Issues
- Include your OS, FLACidal version, and steps to reproduce
- Attach logs from the Terminal tab in the app if applicable

## Code Style

- Go: follow standard `gofmt` formatting
- TypeScript/Svelte: match existing code style
- Commit format: `type: description` (feat, fix, refactor, docs, test, chore)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
