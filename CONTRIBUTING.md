# Contributing to Permission Guardian

Thank you for your interest in contributing to Permission Guardian!

## Development Setup

Prerequisite: Go 1.24+ and tmux installed locally.

1. Clone the repository:
   ```bash
   git clone https://github.com/buddyh/permission-guardian.git
   cd permission-guardian
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build:
   ```bash
   go build -o pg ./cmd/pg
   ```

4. Run tests:
   ```bash
   go test ./...
   ```

5. To exercise the UI while developing:
   ```bash
   ./pg watch
   ```
   Resize the terminal to evaluate how the refreshed header and policy indicators behave under Compact vs Expanded layouts.

# Auto-Approve Modes

Permission Guardian supports the same policies users see in the UI:

- **OFF**: Manual approvals only.
- **SAFE**: Auto-approve all prompts except a curated list of destructive commands.
- **NODEL**: Auto-approve everything that does not delete files, directories, or tables.
- **ALL**: Approve every prompt automatically.
- **BURST**: Temporarily follows the active policy until the session goes idle; useful for short-lived workflows.

Each mode is backed by explicit pattern matching and can be extended via the rules engine. In the TUI, `NODEL` is the compact label used for the delete-blocking policy.

## Detection Notes

Detection relies on tmux pane titles, `ps`, and heuristic keywords, so some wrappers or nested shells might prevent Permission Guardian from identifying a Claude or Codex agent. Treat detection as best-effort and double-check the session label before enabling auto-approve policies. Claude reports an absolute `Ctx: ##k` value, while Codex exposes `% context left`; the UI now renders those raw signals, so tests and instrumentation that inspect the context indicators should assert against the same raw values.

## Code Style

- Run `go fmt ./...` before committing
- Run `go vet ./...` to check for issues
- Follow standard Go conventions

## Releases

Tagged releases are built with GoReleaser and published to GitHub Releases. The same workflow also updates the Homebrew tap at `buddyh/homebrew-tap`.

Release automation requires one repo secret in `buddyh/permission-guardian`:

- `TAP_GITHUB_TOKEN`: a GitHub personal access token with `repo` scope and write access to `buddyh/homebrew-tap`

Without that secret, tag builds can still create release artifacts, but the Homebrew formula update will fail when the workflow tries to write to the tap repository.

Recommended install paths for end users are:

- Homebrew: `brew install buddyh/tap/permission-guardian`
- One-line installer: `curl -fsSL https://raw.githubusercontent.com/buddyh/permission-guardian/main/install.sh | bash`
- `go install` for Go users
- Direct downloads from GitHub Releases

## Testing

- Add tests for new functionality
- Ensure all tests pass before submitting a PR
- Tests are located alongside source files as `*_test.go`

## Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Format code (`go fmt ./...`)
6. Commit with a descriptive message
7. Push to your fork
8. Open a Pull Request

## Reporting Issues

When reporting bugs, please include:
- Go version (`go version`)
- Operating system and version
- Steps to reproduce
- Expected vs actual behavior

## Project Structure

```
permission-guardian/
  cmd/pg/           # Main application entry point
  internal/
    detector/       # Permission prompt detection
    rules/          # Auto-approval rules engine
    db/             # SQLite audit logging
    tmux/           # Tmux session interaction
    tui/            # Terminal UI components
```

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
