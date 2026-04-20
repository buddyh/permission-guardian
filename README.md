# Permission Guardian

A TUI dashboard for monitoring and managing permission prompts across multiple Claude Code and Codex sessions running in tmux.

## What It Requires

Permission Guardian is not a generic terminal wrapper. It expects:

- macOS
- `tmux`
- Claude Code and/or Codex running inside tmux panes

If your agent sessions are not running in tmux, `pg` has nothing to monitor or route.

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

Install methods below only install `pg`. You still need `tmux` installed and your Claude Code/Codex sessions running inside tmux for the app to do anything useful.

### Install tmux first

If you do not already have `tmux`:

```bash
brew install tmux
```

Verify it:

```bash
tmux -V
```

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

macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/buddyh/permission-guardian/main/install.sh | bash
```

Optional environment variables:

```bash
curl -fsSL https://raw.githubusercontent.com/buddyh/permission-guardian/main/install.sh | VERSION=0.1.1 INSTALL_DIR="$HOME/.local/bin" bash
```

Use this when Homebrew is not available or you want a pinned version without building from source.

### Direct release tarballs

Download the archive that matches your Mac and install `pg` manually:

```bash
curl -LO https://github.com/buddyh/permission-guardian/releases/latest/download/permission-guardian_0.1.1_darwin_<arm64|amd64>.tar.gz
tar -xzf permission-guardian_0.1.1_darwin_<arm64|amd64>.tar.gz
sudo install pg /usr/local/bin/pg
```

Release artifacts are published for:

- `darwin/arm64`
- `darwin/amd64`

### Go install

```bash
go install github.com/buddyh/permission-guardian/cmd/pg@latest
```

Use this if you already have a Go toolchain and prefer Go-managed installs.

### Verify the install

After any install path:

```bash
pg --version
```

If you installed to `~/.local/bin`, make sure that directory is on your `PATH`.

### Build from source

```bash
git clone https://github.com/buddyh/permission-guardian.git
cd permission-guardian
go build -o pg ./cmd/pg
sudo install pg /usr/local/bin/pg
```

## Usage

### Quick Start

1. Install `tmux`.
2. Install `pg`.
3. Start a tmux session:

```bash
tmux new -s agents
```

4. Inside tmux, start Claude Code or Codex.
5. In another shell, verify tmux is running:

```bash
tmux ls
```

6. Check whether Permission Guardian can see any waiting sessions:

```bash
pg list
```

7. Launch the dashboard:

```bash
pg watch
```

If `pg list` shows nothing, that can still be normal until one of your agent sessions is actually waiting on a permission prompt.

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

## Troubleshooting

### `tmux is not running`

Start tmux first:

```bash
tmux new -s agents
```

Then launch Claude Code or Codex inside that tmux session.

### `pg watch` opens but shows no sessions

Check these in order:

1. `tmux ls` shows at least one running session.
2. Claude Code or Codex is running inside a tmux pane, not in a normal shell tab.
3. The session is actually waiting on a permission prompt if you are using `pg list`.

### `pg: command not found`

Your install directory is not on `PATH`. Check where the installer placed `pg`, then either add that directory to `PATH` or reinstall to a directory already on `PATH`.

### Detection looks wrong

Detection is best-effort and based on tmux pane content plus process inspection. Nested shells, wrappers, and custom launch flows can make detection less reliable. Keep the agent launch as direct as possible inside tmux.

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

- macOS
- tmux
- Claude Code and/or Codex sessions running inside tmux
- Go 1.24+ (only for building from source)

## License

MIT
