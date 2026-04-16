# Permission Guardian

A TUI dashboard for monitoring and managing permission prompts across multiple Claude Code (and Codex) sessions running in tmux.

## Features

- **Live Dashboard**: Monitor all AI agent sessions in real-time
- **Quick Approval**: Approve or deny permission prompts with single keystrokes
- **Approve + Remember**: Send "don't ask again" responses with shift+number keys
- **Per-Session Auto-Approve**: Toggle automatic approval for specific sessions
- **Safe Mode**: Auto-approve everything except destructive commands (rm -rf, git push --force, etc.)
- **Rules Engine**: Define custom auto-approval rules based on session, command, directory patterns
- **Session Metadata**: View model, context size, git branch, working directory for each session

## Installation

```bash
go install github.com/buddyh/permission-guardian/cmd/pg@latest
```

Or build from source:

```bash
git clone https://github.com/buddyh/permission-guardian.git
cd permission-guardian
go build -o pg ./cmd/pg
sudo mv pg /usr/local/bin/
```

## Usage

### Interactive Dashboard

```bash
pg watch
```

**Keyboard shortcuts:**
| Key | Action |
|-----|--------|
| `1-9` | Approve waiting session #N |
| `!-)`  | Approve + don't ask again for session #N |
| `a` | Approve selected session |
| `s` | Approve + don't ask again for selected |
| `d` | Deny selected session |
| `t` | Toggle auto-approve mode (OFF → SAFE → ALL) |
| `p` | Expand preview panel |
| `l` | View auto-approve log |
| `j/k` or `↑/↓` | Navigate sessions |
| `q` | Quit |

### CLI Commands

```bash
# List sessions waiting for approval
pg list
pg list --json
pg list --count  # Exit code = number waiting

# Approve/deny specific session
pg approve <session-name>
pg deny <session-name>

# Auto-approval daemon with rules
pg auto
pg auto --dry-run

# Manage rules
pg rules init          # Create default rules config
pg rules list          # Show all rules
pg rules enable <name> # Enable a rule
pg rules disable <name>
pg rules add my-rule --session "work-*" --type bash --action approve
pg rules delete <name>
```

## Auto-Approve Modes

When you press `t` on a session in the TUI, it cycles through:

- **OFF**: No auto-approval (default)
- **SAFE**: Auto-approve all except destructive commands
- **ALL**: Auto-approve everything (use with caution)

Destructive commands blocked in SAFE mode include:
- `rm -rf`, `rm -r`, `rm -f`
- `git push --force`, `git reset --hard`
- `DROP TABLE`, `DELETE FROM`, `TRUNCATE`
- `sudo rm`, `shutdown`, `reboot`
- And more...

## Rules Configuration

Rules are stored in `~/.config/permission-guardian/rules.yaml`:

```yaml
rules:
  - name: approve-git-status
    description: Auto-approve safe git commands
    enabled: true
    action: approve
    match:
      prompt_types: [bash]
      commands: ["^git (status|log|diff|branch)"]

  - name: approve-npm-install
    description: Auto-approve npm/yarn install
    enabled: true
    action: approve
    match:
      prompt_types: [bash]
      commands: ["^(npm|yarn|pnpm) install"]
```

## Requirements

- macOS or Linux
- tmux
- Go 1.24+ (for building)

## License

MIT
