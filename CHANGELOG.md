# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial public release
- TUI dashboard for monitoring AI agent sessions (`pg watch`)
- Permission prompt detection for Claude Code and Codex
- Quick approval with number keys (1-9)
- Approve + Remember with shift+number keys
- Auto-approve modes: OFF, SAFE, ALL
- SAFE mode blocks destructive commands (rm -rf, git push --force, etc.)
- YAML-based rules engine for custom auto-approval patterns
- SQLite audit logging for all decisions
- CLI commands: `list`, `approve`, `deny`, `auto`, `rules`
- Session metadata display (model, context size, git branch, working directory)
- Task timer for tracking work sessions
### Changed
- Clarified Go 1.24 build/install requirements and documented the explicit auto-approve policies (`SAFE`, `NODEL`, `ALL`, `BURST`) plus their CLI shortcuts.
- Documented best-effort Claude/Codex detection caveats and the difference between Claude's absolute `Ctx` value and Codex's `% context left`.

### Security
- SAFE mode provides protection against destructive commands
- Audit logging for compliance and review

## [0.1.0] - 2025-01-20

### Added
- Initial release
