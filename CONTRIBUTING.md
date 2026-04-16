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

## Code Style

- Run `go fmt ./...` before committing
- Run `go vet ./...` to check for issues
- Follow standard Go conventions

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
