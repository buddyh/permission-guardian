# Permission Guardian

A TUI dashboard for monitoring and managing permission prompts across multiple Claude Code (and Codex) sessions running in tmux.

## Features

- **Live Dashboard**: Monitor all AI agent sessions in real-time
- **Quick Approval**: Approve or deny permission prompts with single keystrokes
- **Approve + Remember**: Send "don't ask again" responses with shift+number keys
- **Per-Session Auto-Approve**: Toggle automatic approval for specific sessions
- **Explicit Auto Modes**: `SAFE`, `NODEL`, `ALL`, and `BURST` modes describe exactly what the agent can approve automatically
- **Rules Engine**: Define custom auto-approval rules based on session, command, directory patterns
- **Session Metadata**: View model, context size, git branch, working directory for each session

## Installation

Go 1.24+ is only required if you are building from source. For end users, use one of these install paths:

### Homebrew tap

```bash
brew install buddyh/tap/permission-guardian
```

Or, if you prefer the explicit two-step tap flow:

```bash
brew tap buddyh/tap
brew install permission-guardian
```

### One-line installer

macOS and Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/buddyh/permission-guardian/main/install.sh | bash
```

Optional environment variables:

```bash
curl -fsSL https://raw.githubusercontent.com/buddyh/permission-guardian/main/install.sh | VERSION=0.1.1 INSTALL_DIR="$HOME/.local/bin" bash
```

Use this when Homebrew is not available or you want a pinned version without building from source.

### Direct release tarballs

Download a prebuilt archive from the latest release and install `pg` manually:

```bash
https://github.com/buddyh/permission-guardian/releases/latest
```

Release artifacts are published for:

- `darwin/arm64`
- `darwin/amd64`
- `linux/arm64`
- `linux/amd64`

### Go install

```bash
go install github.com/buddyh/permission-guardian/cmd/pg@latest
```

Use this if you already have a Go toolchain and prefer Go-managed installs.

### Build from source

```bash
git clone https://github.com/buddyh/permission-guardian.git
cd permission-guardian
go build -o pg ./cmd/pg
sudo install pg /usr/local/bin/pg
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
| `t` | Toggle auto approval (`OFF` ↔ `SAFE`) |
| `T` or `m` | Cycle auto policy (`SAFE` → `NODEL` → `ALL`) |
| `x` | Switch the selected session to `NODEL` |
| `b` | Toggle burst mode until the session goes idle |
| `p` | Expand preview panel |
| `v` | Cycle view mode |
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

Press `t` in the TUI to advance the active session through:

- **OFF**: No prompts are auto-approved.
- **SAFE**: Auto-approve everything except known destructive commands (`rm -rf`, `git push --force`, `DROP TABLE`, etc.).
- **NODEL**: Auto-approve all prompts that do not delete files, directories, or tables.
- **ALL**: Auto-approve every prompt with no filtering.
- **BURST**: Mirrors whichever auto mode is currently selected, but only until the session idles; useful for short-lived flows.

SAFE and NODEL both rely on curated command-pattern lists, and you can strengthen either policy through the rules engine.

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

## Detection and Context

- **Claude/Codex Detection**: Permission Guardian inspects tmux panes and process metadata to detect Claude Code and Codex agents. Some wrappers, custom shells, or heavily nested tmux trees can hide the agent from `ps`, so treat detection as best-effort and verify the session identity before enabling auto modes.
- **Context Display**: Claude explicitly prints `Ctx: ##k`, and the UI now renders that exact value instead of trying to normalize against a hard-coded max. Codex sessions expose `% context left`, which is shown as a percentage bar. Knowing whether you are on Claude or Codex helps interpret the display correctly.

In the TUI, `NODEL` is the compact on-screen label for the delete-blocking policy.
## Requirements

- macOS or Linux
- tmux
- Go 1.24+ (only for building from source)

## License

MIT
