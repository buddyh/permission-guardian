// Package tui provides the terminal user interface for permission-guardian
package tui

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/buddyh/permission-guardian/internal/db"
	"github.com/buddyh/permission-guardian/internal/detector"
	"github.com/buddyh/permission-guardian/internal/tmux"
	"github.com/charmbracelet/x/ansi"
)

// ═══════════════════════════════════════════════════════════════════════════
// Color Palette - Nord + Dracula inspired
// ═══════════════════════════════════════════════════════════════════════════
var (
	// Base colors
	colorBg       = lipgloss.Color("#1e2139")
	colorPanel    = lipgloss.Color("#232744") // Elevated panel surface
	colorPanelAlt = lipgloss.Color("#1a1e34") // Deeper inset / zebra background
	colorSelected = lipgloss.Color("#314a76") // Cooler, stronger selection highlight
	colorBorder   = lipgloss.Color("#5a6178") // Brighter separators
	colorTextPri  = lipgloss.Color("#e8eaee")
	colorTextSec  = lipgloss.Color("#b8bcc8") // Brighter secondary text
	colorTextDim  = lipgloss.Color("#8890a0") // Brighter dim text

	// Semantic colors
	colorSuccess = lipgloss.Color("#5eead4") // Approved/healthy
	colorWarning = lipgloss.Color("#fbbf24") // Waiting/pending
	colorError   = lipgloss.Color("#ff6b6b") // Denied/error
	colorActive  = lipgloss.Color("#a78bfa") // Working/active
	colorInfo    = lipgloss.Color("#60a5fa") // Info/neutral
	colorAccent  = lipgloss.Color("#f472b6") // Highlight/accent
	colorCodex   = lipgloss.Color("#58a6ff") // Distinct blue for Codex labels
	colorClaude  = lipgloss.Color("#ff7a1a") // True orange for Claude labels
	colorHero    = lipgloss.Color("#63f3ff") // Neon cyan for v2 composited header
	colorHeroAlt = lipgloss.Color("#3a86ff") // Electric blue shadow
	colorHeroFX  = lipgloss.Color("#b517ff") // Purple glow accent
)

// ═══════════════════════════════════════════════════════════════════════════
// Styles
// ═══════════════════════════════════════════════════════════════════════════
var (
	// Logo styles
	logoStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	logoSubStyle = lipgloss.NewStyle().
			Foreground(colorTextSec)

	// Panel styles
	panelStyle = lipgloss.NewStyle().
			Background(colorPanel).
			Border(lipgloss.RoundedBorder()).
			BorderForegroundBlend(colorHeroAlt, colorBorder, colorAccent).
			Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
			Foreground(colorHero).
			Bold(true).
			Padding(0, 1)

	// Session list styles
	sessionNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9fe870")).
				Bold(true)

	// Status styles
	statusWaiting = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true)

	statusWorking = lipgloss.NewStyle().
			Foreground(colorActive)

	statusIdle = lipgloss.NewStyle().
			Foreground(colorTextDim)

	statusApproved = lipgloss.NewStyle().
			Foreground(colorSuccess)

	statusDenied = lipgloss.NewStyle().
			Foreground(colorError)

	statusAccent = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	statusAuto = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	// Detail panel styles
	detailLabelStyle = lipgloss.NewStyle().
				Foreground(colorTextSec)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(colorTextPri)

	// Help bar styles
	helpStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Padding(0, 2)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorInfo)

	// Divider
	dividerStyle = lipgloss.NewStyle().
			Foreground(colorBorder)

	headerRuleStyle = lipgloss.NewStyle().
			Foreground(colorHero).
			Faint(true)

	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(colorHeroAlt).
				Bold(true)

	selectedIndicatorStyle = lipgloss.NewStyle().
				Foreground(colorHero).
				Bold(true)
)

// ASCII Logo
const logo = `██████╗  ██████╗
██╔══██╗██╔════╝
██████╔╝██║  ███╗
██╔═══╝ ██║   ██║
██║     ╚██████╔╝
╚═╝      ╚═════╝`

// ═══════════════════════════════════════════════════════════════════════════
// Key bindings
// ═══════════════════════════════════════════════════════════════════════════
type keyMap struct {
	Up             key.Binding
	Down           key.Binding
	Approve        key.Binding
	ApproveAlways  key.Binding
	Deny           key.Binding
	Refresh        key.Binding
	ToggleAuto     key.Binding
	ToggleAutoMode key.Binding
	ToggleNoDelete key.Binding
	ToggleBurst    key.Binding
	ToggleView     key.Binding
	ViewLog        key.Binding
	Preview        key.Binding
	SendText       key.Binding
	RenameSession  key.Binding
	KillSession    key.Binding
	ToggleGit      key.Binding
	ToggleCtx      key.Binding
	ToggleModel    key.Binding
	Quit           key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Approve: key.NewBinding(
		key.WithKeys("a", "y", "enter"),
		key.WithHelp("a", "approve"),
	),
	ApproveAlways: key.NewBinding(
		key.WithKeys("s", "A"),
		key.WithHelp("s", "approve+remember"),
	),
	Deny: key.NewBinding(
		key.WithKeys("d", "n", "escape"),
		key.WithHelp("d", "deny"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	ToggleAuto: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "off/safe"),
	),
	ToggleAutoMode: key.NewBinding(
		key.WithKeys("T", "m"),
		key.WithHelp("T", "cycle modes"),
	),
	ToggleNoDelete: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "no-delete"),
	),
	ToggleBurst: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "burst"),
	),
	ToggleView: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "view mode"),
	),
	ViewLog: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "log"),
	),
	Preview: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "preview"),
	),
	SendText: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "send text"),
	),
	RenameSession: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "rename"),
	),
	KillSession: key.NewBinding(
		key.WithKeys("K"),
		key.WithHelp("K", "kill"),
	),
	ToggleGit: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "git col"),
	),
	ToggleCtx: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "ctx col"),
	),
	ToggleModel: key.NewBinding(
		key.WithKeys("M"),
		key.WithHelp("M", "agent col"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// ═══════════════════════════════════════════════════════════════════════════
// Messages
// ═══════════════════════════════════════════════════════════════════════════
type tickMsg time.Time
type sessionsMsg struct {
	sessions []detector.WaitingSession
	err      error
}
type actionMsg struct {
	session string
	action  string
	err     error
}

// ═══════════════════════════════════════════════════════════════════════════
// Auto-approve modes
// ═══════════════════════════════════════════════════════════════════════════
type AutoMode int

const (
	AutoOff      AutoMode = iota // No auto-approve
	AutoSafe                     // Approve all except destructive commands
	AutoNoDelete                 // Approve all except delete commands (files, folders, db records)
	AutoAll                      // Approve everything (yolo)
)

func (m AutoMode) String() string {
	switch m {
	case AutoSafe:
		return "SAFE"
	case AutoNoDelete:
		return "NODEL"
	case AutoAll:
		return "ALL"
	default:
		return ""
	}
}

func (m AutoMode) Description() string {
	switch m {
	case AutoSafe:
		return "approve except destructive ops"
	case AutoNoDelete:
		return "approve except delete ops"
	case AutoAll:
		return "approve every prompt"
	default:
		return "auto approve off"
	}
}

func (m AutoMode) Next() AutoMode {
	switch m {
	case AutoOff:
		return AutoSafe
	case AutoSafe:
		return AutoNoDelete
	case AutoNoDelete:
		return AutoAll
	case AutoAll:
		return AutoOff
	default:
		return AutoOff
	}
}

func countAutoModes(modes map[string]AutoMode) (safeCount, noDeleteCount, allCount int) {
	for _, mode := range modes {
		switch mode {
		case AutoSafe:
			safeCount++
		case AutoNoDelete:
			noDeleteCount++
		case AutoAll:
			allCount++
		}
	}
	return safeCount, noDeleteCount, allCount
}

func autoModeStatus(mode AutoMode, sessionName string) string {
	if mode == AutoOff {
		return fmt.Sprintf("AUTO OFF: %s", sessionName)
	}
	return fmt.Sprintf("AUTO %s: %s (%s)", mode.String(), sessionName, mode.Description())
}

func burstModeStatus(mode AutoMode, sessionName string) string {
	return fmt.Sprintf("BURST %s: %s (%s; until idle)", mode.String(), sessionName, mode.Description())
}

// Destructive command patterns - commands that should NOT be auto-approved in SAFE mode
var destructivePatterns = []string{
	// File deletion
	"rm -rf", "rm -fr", "rm -r", "rm -f",
	"rmdir",
	// Disk operations
	"dd if=", "dd of=",
	"mkfs", "fdisk", "parted",
	// Permission changes (broad)
	"chmod -R", "chmod 777", "chmod 666",
	"chown -R",
	// Process killing
	"kill -9", "killall", "pkill",
	// Git destructive
	"git push --force", "git push -f",
	"git reset --hard",
	"git clean -fd", "git clean -f",
	// Database destructive
	"DROP TABLE", "DROP DATABASE", "TRUNCATE",
	"DELETE FROM", "DELETE * FROM",
	// System
	"sudo rm", "sudo dd", "sudo mkfs",
	"shutdown", "reboot", "halt",
	// Dangerous redirects
	"> /dev/", ">/dev/",
	// npm/package destructive
	"npm unpublish",
	// Docker destructive
	"docker system prune", "docker rm -f", "docker rmi -f",
}

// isDestructiveCommand checks if a request contains destructive patterns
func isDestructiveCommand(request string) bool {
	reqLower := strings.ToLower(request)
	for _, pattern := range destructivePatterns {
		if strings.Contains(reqLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// Delete-only patterns - subset that specifically destroy files, folders, or database records
var deletePatterns = []string{
	// File/folder deletion
	"rm -rf", "rm -fr", "rm -r", "rm -f",
	"rmdir",
	"sudo rm",
	// Database deletion
	"DROP TABLE", "DROP DATABASE", "TRUNCATE",
	"DELETE FROM", "DELETE * FROM",
	// Docker deletion
	"docker system prune", "docker rm -f", "docker rmi -f",
}

// isDeleteCommand checks if a request contains delete-specific patterns (files, folders, db records)
func isDeleteCommand(request string) bool {
	reqLower := strings.ToLower(request)
	for _, pattern := range deletePatterns {
		if strings.Contains(reqLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════════════
// Model
// ═══════════════════════════════════════════════════════════════════════════
type Model struct {
	sessions     []detector.WaitingSession
	cursor       int
	spinner      spinner.Model
	refreshRate  time.Duration
	lastRefresh  time.Time
	err          error
	width        int
	height       int
	actionStatus string
	actionTime   time.Time

	// Auto-approve per session (mode)
	autoApprove map[string]AutoMode
	// Burst mode per session - auto-approve until idle
	burstMode map[string]bool
	// Track pending approvals to prevent double-sending
	pendingApproval map[string]time.Time
	// Task timer tracking
	taskStartTime  map[string]time.Time // When current task run started
	taskApprovals  map[string]int       // Number of approvals in current task
	lastActiveTime map[string]time.Time // Last time session was active (for idle grace period)
	// Log viewing
	showLog  bool
	logLines []string
	// Expanded preview mode
	showPreview bool
	// Text input mode
	inputMode bool
	textInput textinput.Model
	// Kill confirmation (K-K within 2 seconds)
	killConfirmSession string
	killConfirmTime    time.Time
	// View mode (compact/expanded)
	viewMode ViewMode
	// Hidden columns (user can toggle)
	hiddenColumns map[Column]bool
	// Audit database
	auditDB *db.DB
}

// LogEntry represents an auto-approval log entry
type LogEntry struct {
	Time       time.Time
	Session    string
	PromptType string
	Request    string
}

// New creates a new TUI model
func New(refreshRate time.Duration) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorAccent)

	// Open audit database (non-fatal if it fails)
	auditDB, _ := db.Open()

	ti := textinput.New()
	ti.Placeholder = "Type text to send (Enter to send, Esc to cancel)"
	ti.CharLimit = 500
	ti.SetWidth(60)
	tiStyles := textinput.DefaultStyles(true)
	tiStyles.Focused.Prompt = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	tiStyles.Focused.Text = lipgloss.NewStyle().Foreground(colorTextPri)
	tiStyles.Blurred.Prompt = tiStyles.Focused.Prompt
	tiStyles.Blurred.Text = tiStyles.Focused.Text
	ti.SetStyles(tiStyles)

	return Model{
		spinner:         s,
		refreshRate:     refreshRate,
		lastRefresh:     time.Now(),
		width:           120,
		height:          40,
		autoApprove:     make(map[string]AutoMode),
		burstMode:       make(map[string]bool),
		pendingApproval: make(map[string]time.Time),
		taskStartTime:   make(map[string]time.Time),
		taskApprovals:   make(map[string]int),
		lastActiveTime:  make(map[string]time.Time),
		hiddenColumns:   make(map[Column]bool),
		textInput:       ti,
		auditDB:         auditDB,
	}
}

// logFilePath returns the path to the auto-approve log file
func logFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "permission-guardian", "auto.log")
}

// writeLog appends an entry to the auto-approve log
func writeLog(session string, promptType string, request string) error {
	path := logFilePath()
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	// Truncate request for log
	req := strings.ReplaceAll(request, "\n", " ")
	if len(req) > 100 {
		req = req[:97] + "..."
	}

	entry := fmt.Sprintf("%s | %s | %s | %s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		session,
		promptType,
		req,
	)
	_, err = f.WriteString(entry)
	return err
}

// taskLogFilePath returns the path to the task run log file
func taskLogFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "permission-guardian", "tasks.log")
}

// writeTaskLog logs a completed task run
func writeTaskLog(session string, duration time.Duration, approvals int) error {
	path := taskLogFilePath()
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	entry := fmt.Sprintf("%s | %s | duration=%s | approvals=%d\n",
		time.Now().Format("2006-01-02 15:04:05"),
		session,
		formatDuration(duration),
		approvals,
	)
	_, err = f.WriteString(entry)
	return err
}

// formatDuration formats a duration as human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", mins, secs)
	} else {
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
}

// readLog reads the last N lines from the log file
func readLog(maxLines int) []string {
	path := logFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{"No log entries yet"}
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	// Reverse to show newest first
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}

	return lines
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return m.spinner.Tick() },
		fetchSessions,
		tickCmd(m.refreshRate),
	)
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

const paneCaptureLines = 120

func fetchSessions() tea.Msg {
	sessions, err := detector.GetAllAgentSessions(paneCaptureLines)
	return sessionsMsg{sessions: sessions, err: err}
}

func approveSession(sessionName string) tea.Cmd {
	return func() tea.Msg {
		if err := tmux.SendKeys(sessionName, "1"); err != nil {
			return actionMsg{session: sessionName, action: "approve", err: err}
		}
		time.Sleep(100 * time.Millisecond)
		if err := tmux.SendEnter(sessionName); err != nil {
			return actionMsg{session: sessionName, action: "approve", err: err}
		}
		return actionMsg{session: sessionName, action: "approved", err: nil}
	}
}

func approveSessionAlways(sessionName string) tea.Cmd {
	return func() tea.Msg {
		// Send "2" for "Yes, and don't ask again"
		if err := tmux.SendKeys(sessionName, "2"); err != nil {
			return actionMsg{session: sessionName, action: "approve+remember", err: err}
		}
		time.Sleep(100 * time.Millisecond)
		if err := tmux.SendEnter(sessionName); err != nil {
			return actionMsg{session: sessionName, action: "approve+remember", err: err}
		}
		return actionMsg{session: sessionName, action: "approved+remembered", err: nil}
	}
}

func denySession(sessionName string) tea.Cmd {
	return func() tea.Msg {
		if err := tmux.SendKeys(sessionName, "Escape"); err != nil {
			return actionMsg{session: sessionName, action: "deny", err: err}
		}
		return actionMsg{session: sessionName, action: "denied", err: nil}
	}
}

func sendTextToSession(sessionName, text string) tea.Cmd {
	return func() tea.Msg {
		if text != "" {
			if err := tmux.SendText(sessionName, text); err != nil {
				return actionMsg{session: sessionName, action: "send text", err: err}
			}
			time.Sleep(50 * time.Millisecond)
		}
		if err := tmux.SendEnter(sessionName); err != nil {
			return actionMsg{session: sessionName, action: "send text", err: err}
		}
		return actionMsg{session: sessionName, action: "sent to", err: nil}
	}
}

func killSessionCmd(sessionName string) tea.Cmd {
	return func() tea.Msg {
		if err := tmux.KillSession(sessionName); err != nil {
			return actionMsg{session: sessionName, action: "kill", err: err}
		}
		return actionMsg{session: sessionName, action: "killed", err: nil}
	}
}

func renameSessionCmd(oldName, newName string) tea.Cmd {
	return func() tea.Msg {
		if err := tmux.RenameSession(oldName, newName); err != nil {
			return actionMsg{session: oldName, action: "rename", err: err}
		}
		return actionMsg{session: newName, action: "renamed", err: nil}
	}
}

func generateSessionName(cwd string, existingNames []string) string {
	base := filepath.Base(cwd)
	timestamp := time.Now().Format("01021504") // MMddHHmm
	candidate := base + "-" + timestamp

	nameSet := make(map[string]bool)
	for _, n := range existingNames {
		nameSet[n] = true
	}

	if !nameSet[candidate] {
		return candidate
	}

	for i := 2; i <= 99; i++ {
		suffixed := fmt.Sprintf("%s-%d", candidate, i)
		if !nameSet[suffixed] {
			return suffixed
		}
	}
	return candidate
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// If in text input mode, route all keys to textinput
		if m.inputMode {
			switch msg.String() {
			case "enter":
				text := m.textInput.Value()
				m.inputMode = false
				m.textInput.Reset()
				m.textInput.Blur()
				if m.cursor < len(m.sessions) {
					sessionName := m.sessions[m.cursor].Session.Name
					return m, sendTextToSession(sessionName, text)
				}
				return m, nil
			case "esc":
				m.inputMode = false
				m.textInput.Reset()
				m.textInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		// If viewing log, any key exits log view
		if m.showLog {
			m.showLog = false
			return m, nil
		}

		// Preview mode stays open - toggle with 'p' key only

		// Handle number keys 1-9 for quick approve
		if len(msg.String()) == 1 && msg.String() >= "1" && msg.String() <= "9" {
			idx := int(msg.String()[0] - '1')
			waitingSessions := m.getWaitingSessions()
			if idx < len(waitingSessions) {
				return m, approveSession(waitingSessions[idx].Session.Name)
			}
		}

		// Handle shift+number keys (!@#$%^&*() for quick approve+remember
		shiftNumbers := "!@#$%^&*()"
		if len(msg.String()) == 1 {
			if idx := strings.Index(shiftNumbers, msg.String()); idx != -1 {
				waitingSessions := m.getWaitingSessions()
				if idx < len(waitingSessions) {
					return m, approveSessionAlways(waitingSessions[idx].Session.Name)
				}
			}
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Up):
			if len(m.sessions) > 0 {
				if m.cursor > 0 {
					m.cursor--
				} else {
					m.cursor = len(m.sessions) - 1 // Wrap to bottom
				}
			}

		case key.Matches(msg, keys.Down):
			if len(m.sessions) > 0 {
				if m.cursor < len(m.sessions)-1 {
					m.cursor++
				} else {
					m.cursor = 0 // Wrap to top
				}
			}

		case key.Matches(msg, keys.Approve):
			if m.cursor < len(m.sessions) {
				session := m.sessions[m.cursor]
				if session.PromptType != detector.PromptUnknown {
					if m.auditDB != nil {
						_ = m.auditDB.LogDecision(db.Decision{
							Timestamp:  time.Now(),
							Session:    session.Session.Name,
							Decision:   "approved",
							Mode:       "manual",
							PromptType: string(session.PromptType),
							Request:    session.Request,
							ProjectDir: session.CWD,
							GitBranch:  session.Info.GitBranch,
						})
					}
					return m, approveSession(session.Session.Name)
				}
			}

		case key.Matches(msg, keys.ApproveAlways):
			if m.cursor < len(m.sessions) {
				session := m.sessions[m.cursor]
				if session.PromptType != detector.PromptUnknown {
					if m.auditDB != nil {
						_ = m.auditDB.LogDecision(db.Decision{
							Timestamp:  time.Now(),
							Session:    session.Session.Name,
							Decision:   "approved_always",
							Mode:       "manual",
							PromptType: string(session.PromptType),
							Request:    session.Request,
							ProjectDir: session.CWD,
							GitBranch:  session.Info.GitBranch,
						})
					}
					return m, approveSessionAlways(session.Session.Name)
				}
			}

		case key.Matches(msg, keys.Deny):
			if m.cursor < len(m.sessions) {
				session := m.sessions[m.cursor]
				if session.PromptType != detector.PromptUnknown {
					if m.auditDB != nil {
						_ = m.auditDB.LogDecision(db.Decision{
							Timestamp:  time.Now(),
							Session:    session.Session.Name,
							Decision:   "denied",
							Mode:       "manual",
							PromptType: string(session.PromptType),
							Request:    session.Request,
							ProjectDir: session.CWD,
							GitBranch:  session.Info.GitBranch,
						})
					}
					return m, denySession(session.Session.Name)
				}
			}

		case key.Matches(msg, keys.Refresh):
			return m, fetchSessions

		case key.Matches(msg, keys.ToggleAuto):
			// Toggle auto-approve ON (SAFE) / OFF for selected session
			if m.cursor < len(m.sessions) {
				name := m.sessions[m.cursor].Session.Name
				currentMode := m.autoApprove[name]

				if currentMode == AutoOff {
					// Turn on with SAFE mode (safe default)
					m.autoApprove[name] = AutoSafe
					m.actionStatus = autoModeStatus(AutoSafe, name)
				} else {
					// Turn off
					m.autoApprove[name] = AutoOff
					delete(m.burstMode, name)
					m.actionStatus = autoModeStatus(AutoOff, name)
				}
				m.actionTime = time.Now()
			}

		case key.Matches(msg, keys.ToggleAutoMode):
			// Cycle between SAFE → NO-DELETE → ALL (only when auto is already on)
			if m.cursor < len(m.sessions) {
				name := m.sessions[m.cursor].Session.Name
				currentMode := m.autoApprove[name]

				switch currentMode {
				case AutoSafe:
					m.autoApprove[name] = AutoNoDelete
					m.actionStatus = autoModeStatus(AutoNoDelete, name)
				case AutoNoDelete:
					m.autoApprove[name] = AutoAll
					m.actionStatus = autoModeStatus(AutoAll, name)
				case AutoAll:
					m.autoApprove[name] = AutoSafe
					m.actionStatus = autoModeStatus(AutoSafe, name)
				default:
					// Auto is off - inform user
					m.actionStatus = fmt.Sprintf("AUTO OFF: %s (press [t] to start in SAFE)", name)
				}
				m.actionTime = time.Now()
			}

		case key.Matches(msg, keys.ToggleNoDelete):
			// Toggle NO-DELETE mode ON/OFF for selected session
			if m.cursor < len(m.sessions) {
				name := m.sessions[m.cursor].Session.Name
				currentMode := m.autoApprove[name]

				if currentMode == AutoNoDelete {
					// Turn off
					m.autoApprove[name] = AutoOff
					delete(m.burstMode, name)
					m.actionStatus = autoModeStatus(AutoOff, name)
				} else {
					// Turn on NO-DELETE (or switch from SAFE/ALL)
					m.autoApprove[name] = AutoNoDelete
					m.actionStatus = autoModeStatus(AutoNoDelete, name)
				}
				m.actionTime = time.Now()
			}

		case key.Matches(msg, keys.ToggleBurst):
			// Toggle burst mode for selected session (requires SAFE or ALL mode)
			if m.cursor < len(m.sessions) {
				name := m.sessions[m.cursor].Session.Name
				mode := m.autoApprove[name]

				if mode == AutoOff {
					// Can't enable burst without an auto mode - enable SAFE+burst
					m.autoApprove[name] = AutoSafe
					m.burstMode[name] = true
					m.actionStatus = burstModeStatus(AutoSafe, name)
				} else if m.burstMode[name] {
					// Turn off burst
					delete(m.burstMode, name)
					m.actionStatus = fmt.Sprintf("BURST OFF: %s (still AUTO %s)", name, mode.String())
				} else {
					// Turn on burst
					m.burstMode[name] = true
					m.actionStatus = burstModeStatus(mode, name)
				}
				m.actionTime = time.Now()
			}

		case key.Matches(msg, keys.ViewLog):
			// Toggle log view
			m.showLog = true
			m.logLines = readLog(20)

		case key.Matches(msg, keys.Preview):
			// Toggle expanded preview for current session
			if m.cursor < len(m.sessions) {
				m.showPreview = !m.showPreview
			}

		case key.Matches(msg, keys.SendText):
			// Enter text input mode for selected session
			if m.cursor < len(m.sessions) {
				m.inputMode = true
				cmd := m.textInput.Focus()
				m.textInput.SetValue("")
				return m, cmd
			}

		case key.Matches(msg, keys.RenameSession):
			// Rename selected session to basename(cwd)-MMddHHmm
			if m.cursor < len(m.sessions) {
				session := m.sessions[m.cursor]
				cwd := session.CWD
				if cwd == "" || cwd == "unknown" {
					m.actionStatus = "Cannot rename: no CWD"
					m.actionTime = time.Now()
					return m, nil
				}
				existingNames, err := tmux.ListSessionNames()
				if err != nil {
					m.actionStatus = fmt.Sprintf("Rename error: %v", err)
					m.actionTime = time.Now()
					return m, nil
				}
				newName := generateSessionName(cwd, existingNames)
				oldName := session.Session.Name
				m.actionStatus = fmt.Sprintf("renaming: %s -> %s", oldName, newName)
				m.actionTime = time.Now()
				return m, renameSessionCmd(oldName, newName)
			}

		case key.Matches(msg, keys.KillSession):
			if m.cursor < len(m.sessions) {
				sessionName := m.sessions[m.cursor].Session.Name
				if m.killConfirmSession == sessionName && time.Since(m.killConfirmTime) < 2*time.Second {
					// Second K within 2s — kill it
					m.killConfirmSession = ""
					m.actionStatus = fmt.Sprintf("killing: %s", sessionName)
					m.actionTime = time.Now()
					return m, killSessionCmd(sessionName)
				}
				// First K — arm confirmation
				m.killConfirmSession = sessionName
				m.killConfirmTime = time.Now()
				m.actionStatus = fmt.Sprintf("Press K again to kill: %s", sessionName)
				m.actionTime = time.Now()
			}

		case key.Matches(msg, keys.ToggleView):
			// Cycle view mode
			m.viewMode = m.viewMode.Next()
			m.actionStatus = fmt.Sprintf("View: %s", m.viewMode.String())
			m.actionTime = time.Now()

		case key.Matches(msg, keys.ToggleGit):
			m.hiddenColumns[ColGit] = !m.hiddenColumns[ColGit]
			if m.hiddenColumns[ColGit] {
				m.actionStatus = "Git column: hidden"
			} else {
				m.actionStatus = "Git column: visible"
			}
			m.actionTime = time.Now()

		case key.Matches(msg, keys.ToggleCtx):
			m.hiddenColumns[ColCtx] = !m.hiddenColumns[ColCtx]
			if m.hiddenColumns[ColCtx] {
				m.actionStatus = "Context column: hidden"
			} else {
				m.actionStatus = "Context column: visible"
			}
			m.actionTime = time.Now()

		case key.Matches(msg, keys.ToggleModel):
			m.hiddenColumns[ColModel] = !m.hiddenColumns[ColModel]
			if m.hiddenColumns[ColModel] {
				m.actionStatus = "Agent column: hidden"
			} else {
				m.actionStatus = "Agent column: visible"
			}
			m.actionTime = time.Now()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.lastRefresh = time.Now()
		// Clear old action status
		if time.Since(m.actionTime) > 3*time.Second {
			m.actionStatus = ""
		}

		// Clean up old pending approvals (older than 5 seconds)
		for name, t := range m.pendingApproval {
			if time.Since(t) > 5*time.Second {
				delete(m.pendingApproval, name)
			}
		}

		// Build a map of current session states for quick lookup
		sessionStates := make(map[string]detector.WaitingSession)
		for _, s := range m.sessions {
			sessionStates[s.Session.Name] = s
		}

		// Process any control files (IPC from Claude Code skills)
		m.processControlFiles()

		// Task timer tracking - track active sessions and detect idle
		const idleGracePeriod = 10 * time.Second
		for _, session := range m.sessions {
			name := session.Session.Name
			isActive := session.Info.IsWorking || session.PromptType != detector.PromptUnknown

			if isActive {
				// Session is active - start timer if not started, update last active time
				if _, exists := m.taskStartTime[name]; !exists {
					m.taskStartTime[name] = time.Now()
					m.taskApprovals[name] = 0
				}
				m.lastActiveTime[name] = time.Now()
			} else {
				// Session is idle - check if we should end the task
				if startTime, exists := m.taskStartTime[name]; exists {
					lastActive := m.lastActiveTime[name]
					if time.Since(lastActive) > idleGracePeriod {
						// Task run complete - log it
						duration := lastActive.Sub(startTime)
						approvals := m.taskApprovals[name]
						_ = writeTaskLog(name, duration, approvals)
						// Log to SQLite
						if m.auditDB != nil {
							_ = m.auditDB.LogTaskRun(db.TaskRun{
								StartTime: startTime,
								EndTime:   lastActive,
								Session:   name,
								Duration:  duration,
								Approvals: approvals,
							})
						}
						// Clean up
						delete(m.taskStartTime, name)
						delete(m.taskApprovals, name)
						delete(m.lastActiveTime, name)
					}
				}
			}
		}

		// Clean up task tracking for sessions that no longer exist
		for name := range m.taskStartTime {
			if _, exists := sessionStates[name]; !exists {
				delete(m.taskStartTime, name)
				delete(m.taskApprovals, name)
				delete(m.lastActiveTime, name)
			}
		}

		// Check for burst mode sessions that have gone idle
		// Disable burst for idle sessions (not working AND not waiting for approval)
		for name := range m.burstMode {
			if session, exists := sessionStates[name]; exists {
				isIdle := !session.Info.IsWorking && session.PromptType == detector.PromptUnknown
				if isIdle {
					delete(m.burstMode, name)
					// Also disable auto-approve when burst ends
					delete(m.autoApprove, name)
					m.actionStatus = fmt.Sprintf("BURST COMPLETE: %s (now idle)", name)
					m.actionTime = time.Now()
				}
			} else {
				// Session no longer exists, clean up
				delete(m.burstMode, name)
				delete(m.autoApprove, name)
			}
		}

		// Auto-approve any waiting sessions based on their mode
		var cmds []tea.Cmd
		for _, session := range m.sessions {
			if session.PromptType == detector.PromptUnknown {
				continue // Not waiting
			}

			// Never auto-approve folder trust prompts - they're one-time security decisions
			if session.PromptType == detector.PromptTrust {
				continue
			}

			// Never auto-approve plan mode - it's an interview, not a permission prompt
			if session.PromptType == detector.PromptPlan {
				continue
			}

			mode := m.autoApprove[session.Session.Name]
			if mode == AutoOff {
				continue // Auto-approve disabled
			}

			// Check if we already sent an approval recently (debounce)
			if lastApproval, exists := m.pendingApproval[session.Session.Name]; exists {
				if time.Since(lastApproval) < 3*time.Second {
					continue // Already approved recently, wait for it to process
				}
			}

			// Check if we should approve based on mode
			shouldApprove := false
			skipReason := ""

			switch mode {
			case AutoAll:
				// Approve everything
				shouldApprove = true
			case AutoSafe:
				// Approve unless it's a destructive command
				if isDestructiveCommand(session.Request) {
					skipReason = "destructive command blocked"
				} else {
					shouldApprove = true
				}
			case AutoNoDelete:
				// Approve unless it deletes files/folders/db records
				if isDeleteCommand(session.Request) {
					skipReason = "delete command blocked"
				} else {
					shouldApprove = true
				}
			}

			if shouldApprove {
				_ = writeLog(session.Session.Name, string(session.PromptType), session.Request)
				// Log to SQLite audit database
				if m.auditDB != nil {
					modeStr := mode.String()
					if m.burstMode[session.Session.Name] {
						modeStr += "+BURST"
					}
					_ = m.auditDB.LogDecision(db.Decision{
						Timestamp:  time.Now(),
						Session:    session.Session.Name,
						Decision:   "auto_approved",
						Mode:       modeStr,
						PromptType: string(session.PromptType),
						Request:    session.Request,
						ProjectDir: session.CWD,
						GitBranch:  session.Info.GitBranch,
					})
				}
				cmds = append(cmds, approveSession(session.Session.Name))
				m.pendingApproval[session.Session.Name] = time.Now()
				// Increment task approval counter
				m.taskApprovals[session.Session.Name]++
				// Show burst indicator in status if in burst mode
				if m.burstMode[session.Session.Name] {
					m.actionStatus = fmt.Sprintf("BURST %s approved: %s", mode.String(), session.Session.Name)
				} else {
					m.actionStatus = fmt.Sprintf("AUTO %s approved: %s", mode.String(), session.Session.Name)
				}
				m.actionTime = time.Now()
			} else if skipReason != "" {
				// Log skipped decisions too
				if m.auditDB != nil {
					_ = m.auditDB.LogDecision(db.Decision{
						Timestamp:  time.Now(),
						Session:    session.Session.Name,
						Decision:   "auto_skipped",
						Mode:       mode.String(),
						PromptType: string(session.PromptType),
						Request:    session.Request,
						ProjectDir: session.CWD,
						GitBranch:  session.Info.GitBranch,
					})
				}
				m.actionStatus = fmt.Sprintf("AUTO %s held: %s (%s)", mode.String(), session.Session.Name, skipReason)
				m.actionTime = time.Now()
			}
		}

		cmds = append(cmds, fetchSessions, tickCmd(m.refreshRate))
		return m, tea.Batch(cmds...)

	case sessionsMsg:
		m.sessions = msg.sessions
		m.err = msg.err
		if m.cursor >= len(m.sessions) && len(m.sessions) > 0 {
			m.cursor = len(m.sessions) - 1
		}

	case actionMsg:
		m.actionTime = time.Now()
		if msg.err != nil {
			m.actionStatus = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.actionStatus = fmt.Sprintf("%s %s", msg.action, msg.session)
		}
		return m, fetchSessions

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) getWaitingSessions() []detector.WaitingSession {
	var waiting []detector.WaitingSession
	for _, s := range m.sessions {
		if s.PromptType != detector.PromptUnknown {
			waiting = append(waiting, s)
		}
	}
	return waiting
}

// getMiniColumns returns compact columns optimized for narrow terminals
func (m Model) getMiniColumns(width int) []ColumnDef {
	// Start with mini column definitions
	columns := make([]ColumnDef, len(MiniColumns))
	copy(columns, MiniColumns)

	// Calculate fixed width used
	fixedWidth := 2 // padding
	for _, col := range columns {
		if col.ID != ColRequest {
			fixedWidth += col.Width + MiniSeparatorWidth
		}
	}

	// Remove optional columns if too narrow
	if width < 70 {
		// Remove DIR column
		for i, col := range columns {
			if col.ID == ColDir {
				columns = append(columns[:i], columns[i+1:]...)
				fixedWidth -= col.Width + MiniSeparatorWidth
				break
			}
		}
	}
	if width < 60 {
		// Remove TIME column
		for i, col := range columns {
			if col.ID == ColTime {
				columns = append(columns[:i], columns[i+1:]...)
				fixedWidth -= col.Width + MiniSeparatorWidth
				break
			}
		}
	}

	minName := 6
	minStatus := 4
	minRequest := 6
	if width < 32 {
		minName = 4
		minStatus = 3
		minRequest = 4
	}

	// Tighten fixed columns if the request column would be too small.
	remaining := width - fixedWidth - MiniSeparatorWidth
	deficit := minRequest - remaining
	if deficit > 0 {
		shrink := func(id Column, minWidth int) {
			if deficit <= 0 {
				return
			}
			for i := range columns {
				if columns[i].ID == id && columns[i].Width > minWidth {
					delta := min(deficit, columns[i].Width-minWidth)
					columns[i].Width -= delta
					fixedWidth -= delta
					deficit -= delta
					return
				}
			}
		}
		shrink(ColName, minName)
		shrink(ColStatus, minStatus)
		remaining = width - fixedWidth - MiniSeparatorWidth
	}
	if remaining < 1 {
		remaining = 1
	}

	// Set request column to remaining width
	for i := range columns {
		if columns[i].ID == ColRequest {
			columns[i].Width = remaining
		}
	}

	return columns
}

// ═══════════════════════════════════════════════════════════════════════════
// View
// ═══════════════════════════════════════════════════════════════════════════

// shouldUseMiniMode returns true if mini mode should be active
// (either manually selected or auto-triggered by narrow terminal)
func (m Model) shouldUseMiniMode(width int) bool {
	// If user explicitly selected mini mode, always use it
	if m.viewMode == ViewMini {
		return true
	}
	// Auto-trigger mini mode for very narrow terminals
	return width < 90
}

// shouldUseFullHeader returns true when we have room for the full header.
func (m Model) shouldUseFullHeader(width, height int) bool {
	if m.viewMode == ViewMini {
		return false
	}
	return width >= 90 && height >= 16
}

func (m Model) View() tea.View {
	var v tea.View
	v.AltScreen = true

	if m.err != nil {
		v.SetContent(fmt.Sprintf("\n  Error: %v\n\n  Press q to quit.\n", m.err))
		return v
	}

	width := m.width
	height := m.height
	if width < 24 {
		width = 24
	}
	if height < 8 {
		height = 8
	}

	// If showing log, render log view instead
	if m.showLog {
		v.SetContent(m.renderLogView(width, height))
		return v
	}

	isMini := m.shouldUseMiniMode(width)
	var header string
	if m.shouldUseFullHeader(width, height) && !isMini {
		header = m.renderHeader(width)
	} else {
		header = m.renderMiniHeader(width)
	}
	header = lipgloss.NewStyle().Width(width).Render(header)
	headerHeight := lipgloss.Height(header)

	helpMode := helpModeForSize(width, height, isMini)
	helpBar := m.renderHelpBar(width, helpMode, isMini)
	helpHeight := 0
	if helpBar != "" {
		helpHeight = lipgloss.Height(helpBar)
	}

	minContentHeight := 6
	availableHeight := height - headerHeight - helpHeight
	if availableHeight < minContentHeight && helpMode != helpModeMinimal {
		helpMode = helpModeMinimal
		helpBar = m.renderHelpBar(width, helpMode, isMini)
		helpHeight = 0
		if helpBar != "" {
			helpHeight = lipgloss.Height(helpBar)
		}
		availableHeight = height - headerHeight - helpHeight
	}
	if availableHeight < minContentHeight && helpBar != "" {
		helpBar = ""
		availableHeight = height - headerHeight
	}
	if availableHeight < 1 {
		availableHeight = 1
	}

	helpAtTop := helpBar != "" && height < 14
	var sections []string
	sections = append(sections, header)
	if helpAtTop {
		sections = append(sections, helpBar)
	}

	if m.showPreview && m.cursor < len(m.sessions) {
		if availableHeight < 10 {
			sections = append(sections, m.renderPreviewView(width, availableHeight))
		} else {
			// Split view: sessions on top (~40%), preview on bottom (~60%)
			sessionsHeight := availableHeight * 2 / 5
			if sessionsHeight < 6 {
				sessionsHeight = 6
			}
			previewHeight := availableHeight - sessionsHeight
			if previewHeight < 4 {
				previewHeight = 4
				sessionsHeight = availableHeight - previewHeight
			}

			// Render compact sessions table (no detail panel)
			sections = append(sections, m.renderSessionTable(width, sessionsHeight))

			// Render preview panel below
			if previewHeight > 0 {
				sections = append(sections, m.renderSplitPreview(width, previewHeight))
			}
		}
	} else {
		// Normal view: full sessions with optional detail panel
		sections = append(sections, m.renderMainContent(width, availableHeight))
	}

	if m.inputMode {
		// Render text input bar instead of help bar
		sessionName := ""
		if m.cursor < len(m.sessions) {
			sessionName = m.sessions[m.cursor].Session.Name
		}
		inputLabel := statusAccent.Bold(true).Render(" INPUT ") +
			detailLabelStyle.Render(fmt.Sprintf(" -> %s ", sessionName))
		inputBar := inputLabel + m.textInput.View()
		sections = append(sections, helpStyle.Render(inputBar))
	} else if !helpAtTop && helpBar != "" {
		sections = append(sections, helpBar)
	}

	v.SetContent(lipgloss.JoinVertical(lipgloss.Left, sections...))
	return v
}

func (m Model) renderLogView(width, height int) string {
	title := statusAccent.Render(" AUTO-APPROVE LOG (press any key to close) ")

	var lines []string
	lines = append(lines, "")
	lines = append(lines, title)
	lines = append(lines, "")
	lines = append(lines, dividerStyle.Render(strings.Repeat("─", width-4)))

	for _, line := range m.logLines {
		if len(line) > width-4 {
			line = line[:width-7] + "..."
		}
		lines = append(lines, "  "+line)
	}

	// Count auto-approve sessions by mode
	safeCount, noDeleteCount, allCount := countAutoModes(m.autoApprove)

	lines = append(lines, "")
	lines = append(lines, dividerStyle.Render(strings.Repeat("─", width-4)))
	lines = append(lines, fmt.Sprintf("  Auto policies: %d SAFE, %d NODEL, %d ALL", safeCount, noDeleteCount, allCount))
	lines = append(lines, fmt.Sprintf("  Log file: %s", logFilePath()))
	lines = append(lines, "")
	lines = append(lines, detailLabelStyle.Render("  SAFE: approve except destructive ops (rm -rf, git push --force, chmod -R, kill -9, etc.)"))
	lines = append(lines, detailLabelStyle.Render("  NODEL: approve except delete ops (rm, rmdir, DROP/TRUNCATE/DELETE, docker prune/remove, etc.)"))
	lines = append(lines, detailLabelStyle.Render("  ALL: approve every prompt except trust-folder and plan-mode prompts, which are never auto-approved"))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderPreviewView(width, height int) string {
	if m.cursor >= len(m.sessions) {
		return ""
	}

	session := m.sessions[m.cursor]
	compact := height < 14

	// Title bar
	title := statusAccent.Render(fmt.Sprintf(" PREVIEW: %s (press any key to close) ", session.Session.Name))

	var lines []string
	if !compact {
		lines = append(lines, "")
	}
	lines = append(lines, title)
	if !compact {
		lines = append(lines, "")
	}

	// Session metadata
	metaStyle := detailLabelStyle
	valStyle := detailValueStyle

	lines = append(lines, metaStyle.Render("  Session: ")+valStyle.Render(session.Session.Name))
	lines = append(lines, metaStyle.Render("  Agent:   ")+valStyle.Render(string(session.Agent)))
	if session.Info.Model != "" {
		lines = append(lines, metaStyle.Render("  Model:   ")+valStyle.Render(session.Info.Model))
	}
	if session.CWD != "" && session.CWD != "unknown" {
		cwd := session.CWD
		if strings.HasPrefix(cwd, "/Users/") {
			parts := strings.SplitN(cwd, "/", 4)
			if len(parts) >= 4 {
				cwd = "~/" + parts[3]
			}
		}
		lines = append(lines, metaStyle.Render("  Dir:     ")+valStyle.Render(cwd))
	}

	if !compact {
		lines = append(lines, "")
	}
	lines = append(lines, dividerStyle.Render(strings.Repeat("─", width-4)))

	// Show request prominently if waiting
	if session.PromptType != detector.PromptUnknown {
		promptType := strings.ToUpper(string(session.PromptType))
		if !compact {
			lines = append(lines, "")
		}
		lines = append(lines, statusWaiting.Bold(true).Render(fmt.Sprintf("  %s REQUEST:", promptType)))
		if !compact {
			lines = append(lines, "")
		}

		// Show the extracted request (this is the good parsed content)
		reqLines := strings.Split(session.Request, "\n")
		for _, line := range reqLines {
			if len(line) > width-6 {
				line = line[:width-9] + "..."
			}
			lines = append(lines, "  "+statusWaiting.Render(line))
		}

		if !compact {
			lines = append(lines, "")
		}
		lines = append(lines, dividerStyle.Render(strings.Repeat("─", width-4)))
	}

	// Raw content - show as much as fits (with terminal styling preserved)
	if !compact {
		lines = append(lines, "")
	}
	lines = append(lines, detailLabelStyle.Render("  TERMINAL CONTENT:"))
	if !compact {
		lines = append(lines, "")
	}

	// Use StyledContent if available (preserves ANSI colors), fall back to RawContent
	contentToShow := session.StyledContent
	if contentToShow == "" {
		contentToShow = session.RawContent
	}

	rawLines := strings.Split(contentToShow, "\n")
	maxRawLines := height - len(lines)
	if maxRawLines < 0 {
		maxRawLines = 0
	}
	if maxRawLines > 0 {
		if len(rawLines) > maxRawLines {
			rawLines = rawLines[len(rawLines)-maxRawLines:]
		}
		for _, line := range rawLines {
			// Don't truncate styled content - ANSI codes make length calculation wrong
			// Just add padding and render directly to preserve colors
			lines = append(lines, "  "+line)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderSplitPreview(width, height int) string {
	if m.cursor >= len(m.sessions) {
		return ""
	}

	session := m.sessions[m.cursor]

	// Title bar with session name and close hint
	title := statusAccent.Render(fmt.Sprintf(" PREVIEW: %s ", session.Session.Name)) +
		detailLabelStyle.Render("[p] close")

	var lines []string
	lines = append(lines, title)

	// Use StyledContent if available (preserves ANSI colors)
	contentToShow := session.StyledContent
	if contentToShow == "" {
		contentToShow = session.RawContent
	}

	rawLines := strings.Split(contentToShow, "\n")

	// Calculate how many lines we can show
	maxLines := height - 3 // Account for title and panel borders
	if maxLines < 1 {
		maxLines = 1
	}

	// Show the last N lines (most recent content)
	if len(rawLines) > maxLines {
		rawLines = rawLines[len(rawLines)-maxLines:]
	}

	for _, line := range rawLines {
		lines = append(lines, "  "+line)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	return panelStyle.Width(width).Render(content)
}

func solidBlock(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	line := strings.Repeat(" ", width)
	lines := make([]string, height)
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func renderCompositedBox(content string, boxStyle, shadowStyle lipgloss.Style, shadowOffsetX, shadowOffsetY int) string {
	box := boxStyle.Render(content)
	shadow := shadowStyle.Render(solidBlock(lipgloss.Width(box), lipgloss.Height(box)))
	comp := lipgloss.NewCompositor(
		lipgloss.NewLayer(shadow).X(shadowOffsetX).Y(shadowOffsetY).Z(0),
		lipgloss.NewLayer(box).Z(1),
	)
	return comp.Render()
}

func (m Model) renderHeader(width int) string {
	const (
		headerGap     = 4
		titleGap      = 2
		minTitleWidth = 34
	)

	logoBlock := lipgloss.NewStyle().
		Padding(1, 0, 0, 2).
		Render(logoStyle.Render(logo))

	logoWidth := lipgloss.Width(logoBlock)
	titleMainStyle := lipgloss.NewStyle().
		Foreground(colorHero).
		Bold(true)
	titleMain := titleMainStyle.Render("Permission Guardian")
	titleLines := []string{
		titleMain,
		logoSubStyle.Render("tmux approval router for Claude Code + Codex"),
	}
	if width >= 96 {
		ruleWidth := width / 5
		if ruleWidth > 26 {
			ruleWidth = 26
		}
		if ruleWidth < 12 {
			ruleWidth = 12
		}
		titleLines = append(titleLines, headerRuleStyle.Render(strings.Repeat("═", ruleWidth)))
	}
	// Stats section
	waiting := m.getWaitingSessions()
	totalSessions := len(m.sessions)
	waitingCount := len(waiting)
	safeCount, noDeleteCount, allCount := countAutoModes(m.autoApprove)

	var statusText string
	if waitingCount > 0 {
		statusText = statusWaiting.Render(fmt.Sprintf("%d WAITING", waitingCount))
	} else {
		statusText = statusApproved.Render("ALL CLEAR")
	}

	statsTop := lipgloss.JoinHorizontal(
		lipgloss.Left,
		detailValueStyle.Render(fmt.Sprintf("%d", totalSessions)),
		" ",
		detailLabelStyle.Render("sessions"),
		"   ",
		statusText,
		" ",
		m.spinner.View(),
	)

	// Action status
	actionLine := ""
	if m.actionStatus != "" {
		actionStatus := m.actionStatus
		if len(actionStatus) > 34 {
			actionStatus = smartTruncate(actionStatus, 34)
		}
		switch {
		case strings.HasPrefix(m.actionStatus, "approved+remembered"):
			actionLine = statusAccent.Render(actionStatus + " (won't ask again)")
		case strings.HasPrefix(m.actionStatus, "approved"):
			actionLine = statusApproved.Render(actionStatus)
		case strings.HasPrefix(m.actionStatus, "denied"):
			actionLine = statusDenied.Render(actionStatus)
		case strings.HasPrefix(m.actionStatus, "AUTO OFF"):
			actionLine = statusIdle.Render(actionStatus)
		case strings.HasPrefix(m.actionStatus, "AUTO "), strings.HasPrefix(m.actionStatus, "BURST "):
			actionLine = statusAuto.Render(actionStatus)
		default:
			actionLine = detailValueStyle.Render(actionStatus)
		}
	}

	statsLines := []string{
		panelTitleStyle.Render("STATUS"),
		statsTop,
		detailLabelStyle.Render(fmt.Sprintf("Auto policies: %d SAFE · %d NODEL · %d ALL", safeCount, noDeleteCount, allCount)),
	}
	if actionLine != "" {
		statsLines = append(statsLines, actionLine)
	}
	statsContent := lipgloss.JoinVertical(lipgloss.Left, statsLines...)
	statsCardStyle := lipgloss.NewStyle().
		Background(colorPanel).
		Border(lipgloss.RoundedBorder()).
		BorderForegroundBlend(colorHeroAlt, colorHero, colorAccent).
		Padding(0, 1)
	statsShadowStyle := lipgloss.NewStyle().
		Background(colorBg).
		Faint(true)
	statsCard := renderCompositedBox(statsContent, statsCardStyle, statsShadowStyle, 2, 1)
	statsWidth := lipgloss.Width(statsCard)

	titleWidth := width - logoWidth - titleGap
	statsOnTopRow := false
	if width >= logoWidth+titleGap+minTitleWidth+headerGap+statsWidth {
		statsOnTopRow = true
		titleWidth = width - statsWidth - headerGap - logoWidth - titleGap
	}
	if titleWidth < minTitleWidth {
		titleWidth = minTitleWidth
	}

	titleBlock := lipgloss.NewStyle().
		PaddingTop(3).
		Width(titleWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, titleLines...))
	leftBlock := lipgloss.JoinHorizontal(lipgloss.Top, logoBlock, strings.Repeat(" ", titleGap), titleBlock)

	var sections []string
	if statsOnTopRow {
		spacerWidth := width - lipgloss.Width(leftBlock) - statsWidth
		if spacerWidth < headerGap {
			spacerWidth = headerGap
		}
		topRow := lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, strings.Repeat(" ", spacerWidth), statsCard)
		sections = append(sections, topRow)
	} else {
		sections = append(sections, leftBlock)
		sections = append(sections, lipgloss.PlaceHorizontal(width, lipgloss.Right, statsCard))
	}

	policyText := "SAFE = non-destructive  •  NODEL = no delete ops  •  ALL = everything"
	policyInset := logoWidth + titleGap
	if width >= 120 && width-policyInset >= lipgloss.Width(policyText) {
		sections = append(sections, lipgloss.NewStyle().
			PaddingLeft(policyInset).
			Render(detailLabelStyle.Render(policyText)))
	}

	return lipgloss.NewStyle().
		Width(width).
		Background(colorPanelAlt).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForegroundBlend(colorHeroAlt, colorHero, colorAccent).
		PaddingTop(1).
		PaddingBottom(1).
		Render(lipgloss.JoinVertical(lipgloss.Left, sections...))
}

// renderMiniHeader renders a compact single-line header for mini mode
func (m Model) renderMiniHeader(width int) string {
	waiting := m.getWaitingSessions()
	totalSessions := len(m.sessions)
	waitingCount := len(waiting)

	// Build compact status line: "PG │ 5 sessions │ 2 WAITING │ ⣾"
	parts := []string{
		statusAccent.Bold(true).Render("PG"),
		detailValueStyle.Render(fmt.Sprintf("%d sess", totalSessions)),
	}

	if waitingCount > 0 {
		parts = append(parts, statusWaiting.Render(fmt.Sprintf("%d WAIT", waitingCount)))
	} else {
		parts = append(parts, statusApproved.Render("OK"))
	}

	parts = append(parts, m.spinner.View())

	// Add action status if present
	if m.actionStatus != "" {
		status := m.actionStatus
		if width > 0 {
			maxStatusLen := max(8, min(20, width/4))
			if len(status) > maxStatusLen {
				status = status[:maxStatusLen-3] + "..."
			}
		} else if len(status) > 20 {
			status = status[:17] + "..."
		}
		var styledStatus string
		switch {
		case strings.HasPrefix(status, "AUTO"):
			styledStatus = statusAuto.Render(status)
		case strings.HasPrefix(status, "SKIP"):
			styledStatus = statusWaiting.Render(status)
		default:
			styledStatus = detailLabelStyle.Render(status)
		}
		parts = append(parts, styledStatus)
	}

	header := strings.Join(parts, dividerStyle.Render(" │ "))
	return " " + header
}

func (m Model) renderMainContent(width, height int) string {
	// Check if we need detail panel (only for waiting sessions)
	showDetail := m.cursor < len(m.sessions) && m.sessions[m.cursor].PromptType != detector.PromptUnknown

	// Detail panel is compact and collapses on short screens.
	detailHeight := 0
	if showDetail {
		switch {
		case height >= 16:
			detailHeight = 6
		case height >= 12:
			detailHeight = 4
		default:
			detailHeight = 0
		}
	}

	// Table gets all remaining height - sessions are priority!
	tableHeight := height - detailHeight
	if tableHeight < 4 {
		tableHeight = 4
	}

	// Full-width session table at top
	tablePanel := m.renderSessionTable(width, tableHeight)

	// Detail panel at bottom (only if waiting session selected)
	var detailPanel string
	if showDetail && detailHeight > 0 {
		detailPanel = m.renderDetailPanel(width, detailHeight)
	}

	return lipgloss.JoinVertical(lipgloss.Left, tablePanel, detailPanel)
}

func (m Model) renderSessionTable(width, height int) string {
	isMini := m.shouldUseMiniMode(width)

	// Get view mode indicator for title (shorter in mini mode)
	viewIndicator := ""
	if isMini {
		viewIndicator = " [M]"
	} else if m.viewMode == ViewExpanded {
		viewIndicator = " [Expanded]"
	}
	titleRule := ""
	if width >= 80 {
		ruleWidth := width / 6
		if ruleWidth > 18 {
			ruleWidth = 18
		}
		if ruleWidth < 8 {
			ruleWidth = 8
		}
		titleRule = " " + headerRuleStyle.Render(strings.Repeat("═", ruleWidth))
	}
	title := panelTitleStyle.Render("SESSIONS"+viewIndicator) + titleRule

	// Get column definitions based on mode
	var allColumns []ColumnDef
	var separator string
	var sepWidth int
	innerWidth := width - 4
	if innerWidth < 0 {
		innerWidth = 0
	}
	if isMini {
		allColumns = m.getMiniColumns(innerWidth)
		separator = MiniSeparator
		sepWidth = MiniSeparatorWidth
	} else {
		allColumns = GetVisibleColumns(innerWidth) // -4 for panel padding
		separator = ColumnSeparator
		sepWidth = SeparatorWidth
	}

	// Filter out user-hidden columns
	var columns []ColumnDef
	for _, col := range allColumns {
		if !m.hiddenColumns[col.ID] {
			columns = append(columns, col)
		}
	}

	// Build header
	sepStyle := lipgloss.NewStyle().Foreground(colorHeroAlt).Faint(true)
	headerTextStyle := tableHeaderStyle

	var headerCells []string
	var divParts []string
	for _, col := range columns {
		headerCells = append(headerCells, headerTextStyle.Width(col.Width).Render(col.Header))
		divParts = append(divParts, strings.Repeat("─", col.Width))
	}
	headerLine := " " + strings.Join(headerCells, sepStyle.Render(separator))
	divSep := "─"
	if !isMini {
		divSep = "─┼─"
	}
	_ = sepWidth // suppress unused warning
	divider := headerRuleStyle.Render(" " + strings.Join(divParts, divSep))

	var lines []string
	lines = append(lines, headerLine)
	lines = append(lines, divider)

	waitingNum := 1
	for i, session := range m.sessions {
		isSelected := i == m.cursor
		rowLines := m.renderTableRow(session, isSelected, waitingNum, innerWidth, i, columns, isMini)
		lines = append(lines, rowLines...)

		if session.PromptType != detector.PromptUnknown {
			waitingNum++
		}
	}

	if len(m.sessions) == 0 {
		lines = append(lines, statusIdle.Render("  No sessions found"))
	}

	maxLines := height - 3
	if maxLines > 0 {
		if len(lines) > maxLines {
			lines = lines[:maxLines]
		} else if len(lines) < maxLines {
			rowIndex := len(m.sessions)
			for len(lines) < maxLines {
				lines = append(lines, m.renderEmptyRow(columns, isMini, rowIndex))
				rowIndex++
			}
		}
	}
	content := strings.Join(lines, "\n")

	panel := panelStyle.Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, content),
	)

	return panel
}

func (m Model) renderTableRow(session detector.WaitingSession, selected bool, waitingNum int, width int, rowIndex int, columns []ColumnDef, isMini bool) []string {
	isWaiting := session.PromptType != detector.PromptUnknown
	sepStyle := lipgloss.NewStyle().Foreground(colorBorder)

	// Choose separator based on mode
	separator := ColumnSeparator
	if isMini {
		separator = MiniSeparator
	}

	// Determine row background
	var rowBg color.Color
	if selected {
		rowBg = colorSelected
	} else if rowIndex%2 == 1 {
		rowBg = colorPanelAlt
	} else {
		rowBg = colorPanel
	}

	// Helper to apply background to a style
	withBg := func(s lipgloss.Style) lipgloss.Style {
		return s.Background(rowBg)
	}

	// Prepare all column values
	values := m.getColumnValues(session, isWaiting, waitingNum, columns, isMini)

	// Build cells for visible columns only
	var cells []string
	for _, col := range columns {
		cv := values[col.ID]
		val := cv.Text
		style := cv.Style
		// In compact mode, truncate request in the row
		if col.ID == ColRequest && m.viewMode == ViewCompact {
			if len(val) > col.Width && col.Width > 3 {
				val = val[:col.Width-3] + "..."
			}
		}
		// In expanded mode, show "-" for request in the main row (we show it on line 2)
		if col.ID == ColRequest && m.viewMode == ViewExpanded && isWaiting {
			val = "(see below)"
			style = detailLabelStyle
		}
		cells = append(cells, withBg(style).Width(col.Width).Render(val))
	}

	// Separator with background
	sep := withBg(sepStyle).Render(separator)
	row := strings.Join(cells, sep)

	// Add selection indicator or space prefix
	if selected {
		indicator := withBg(selectedIndicatorStyle).Render("▶")
		row = indicator + row
	} else {
		row = withBg(lipgloss.NewStyle()).Render(" ") + row
	}

	result := []string{row}

	// In expanded view, add a second line for the request
	if m.viewMode == ViewExpanded && isWaiting {
		req := session.Request
		if req == "" {
			req = string(session.PromptType) + " request"
		}
		req = strings.ReplaceAll(req, "\n", " ")
		req = strings.TrimSpace(req)

		// Calculate max width for request line (full row width minus some padding)
		maxReqWidth := width - 6
		if len(req) > maxReqWidth && maxReqWidth > 3 {
			req = req[:maxReqWidth-3] + "..."
		}

		// Render request line with same background
		reqLine := withBg(lipgloss.NewStyle()).Render("  ") +
			withBg(statusWaiting).Render("└─ "+req)

		// Pad to full width
		if lipgloss.Width(reqLine) < width {
			padding := width - lipgloss.Width(reqLine)
			reqLine = reqLine + withBg(lipgloss.NewStyle()).Render(strings.Repeat(" ", padding))
		}

		result = append(result, reqLine)
	}

	return result
}

func (m Model) renderEmptyRow(columns []ColumnDef, isMini bool, rowIndex int) string {
	sepStyle := lipgloss.NewStyle().Foreground(colorBorder)
	separator := ColumnSeparator
	if isMini {
		separator = MiniSeparator
	}

	rowBg := colorBg
	if rowIndex%2 == 1 {
		rowBg = colorPanelAlt
	} else {
		rowBg = colorPanel
	}
	withBg := func(s lipgloss.Style) lipgloss.Style {
		return s.Background(rowBg)
	}

	var cells []string
	for _, col := range columns {
		cells = append(cells, withBg(lipgloss.NewStyle()).Width(col.Width).Render(""))
	}

	sep := withBg(sepStyle).Render(separator)
	row := strings.Join(cells, sep)
	row = withBg(lipgloss.NewStyle()).Render(" ") + row

	return row
}

// ColumnValue holds the text and style for a column cell
type ColumnValue struct {
	Text  string
	Style lipgloss.Style
}

// getColumnValues returns the display value and style for each column
func (m Model) getColumnValues(session detector.WaitingSession, isWaiting bool, waitingNum int, columns []ColumnDef, isMini bool) map[Column]ColumnValue {
	values := make(map[Column]ColumnValue)

	// NUM
	if isWaiting {
		if isMini {
			values[ColNum] = ColumnValue{fmt.Sprintf("%d", waitingNum), statusWaiting.Bold(true)}
		} else {
			values[ColNum] = ColumnValue{fmt.Sprintf("%d.", waitingNum), statusWaiting.Bold(true)}
		}
	} else {
		values[ColNum] = ColumnValue{"", statusIdle}
	}

	// NAME
	name := session.Session.Name
	nameWidth := GetColumnWidth(columns, ColName)
	if nameWidth > 0 && len(name) > nameWidth {
		name = name[:nameWidth-3] + "..."
	}
	values[ColName] = ColumnValue{name, sessionNameStyle}

	// STATUS - use shorter labels in mini mode
	autoMode := m.autoApprove[session.Session.Name]
	isBurst := m.burstMode[session.Session.Name]
	var statusText string
	var statusStyle lipgloss.Style
	if autoMode != AutoOff {
		if isBurst {
			if isMini {
				mini := autoMode.String()[0:1] // "S", "N", or "A"
				statusText = mini + "B"
			} else {
				statusText = autoMode.String() + "+B"
			}
		} else {
			if isMini {
				statusText = autoMode.String()[0:1] // "S", "N", or "A"
			} else {
				statusText = autoMode.String()
			}
		}
		statusStyle = statusAuto
	} else if isWaiting {
		if isMini {
			statusText = "WAIT"
		} else {
			statusText = "WAITING"
		}
		statusStyle = statusWaiting
	} else if session.Info.IsWorking {
		if isMini {
			statusText = "work"
		} else {
			statusText = "working"
		}
		statusStyle = statusWorking
	} else {
		statusText = "idle"
		statusStyle = statusIdle
	}
	values[ColStatus] = ColumnValue{statusText, statusStyle}

	// TIME - prefer session info from status line, fall back to local timer
	var timeText string
	var timeStyle lipgloss.Style
	if session.Info.SessionTime != "" {
		timeText = session.Info.SessionTime
		timeStyle = detailValueStyle
	} else if startTime, exists := m.taskStartTime[session.Session.Name]; exists {
		duration := time.Since(startTime)
		timeText = formatDuration(duration)
		if duration > 30*time.Minute {
			timeStyle = statusWaiting
		} else {
			timeStyle = statusWorking
		}
	} else {
		timeText = "-"
		timeStyle = statusIdle
	}
	values[ColTime] = ColumnValue{timeText, timeStyle}

	// MODEL -> now shows Agent type (Claude/Codex)
	var agentText string
	var agentStyle lipgloss.Style
	switch session.Agent {
	case detector.AgentClaude:
		if isMini {
			agentText = "CC"
		} else {
			agentText = "Claude"
		}
		agentStyle = lipgloss.NewStyle().Foreground(colorClaude).Bold(true)
	case detector.AgentCodex:
		if isMini {
			agentText = "CX"
		} else {
			agentText = "Codex"
		}
		agentStyle = lipgloss.NewStyle().Foreground(colorCodex).Bold(true)
	default:
		agentText = "-"
		agentStyle = detailLabelStyle
	}
	values[ColModel] = ColumnValue{agentText, agentStyle}

	// CTX
	ctxWidth := GetColumnWidth(columns, ColCtx)
	values[ColCtx] = ColumnValue{renderContextBar(session.Info.ContextSize, ctxWidth), lipgloss.NewStyle()}

	// GIT
	git := strings.TrimSpace(session.Info.GitBranch)
	if git == "" || git == "no" {
		git = "-"
	} else {
		if changes := strings.TrimSpace(session.Info.GitChanges); changes != "" {
			git = git + " " + changes
		}
		gitWidth := GetColumnWidth(columns, ColGit)
		if gitWidth > 0 && len(git) > gitWidth {
			git = git[:gitWidth-3] + "..."
		}
	}
	values[ColGit] = ColumnValue{git, detailLabelStyle}

	// DIR
	dir := session.CWD
	if dir == "" || dir == "unknown" {
		dir = "-"
	} else {
		if strings.HasPrefix(dir, "/Users/") {
			parts := strings.SplitN(dir, "/", 4)
			if len(parts) >= 4 {
				dir = "~/" + parts[3]
			}
		}
		dirWidth := GetColumnWidth(columns, ColDir)
		if dirWidth > 0 && len(dir) > dirWidth {
			dir = smartTruncate(dir, dirWidth)
		}
	}
	values[ColDir] = ColumnValue{dir, detailLabelStyle}

	// REQUEST - show prompt request if waiting, working status if active, last input if idle
	var req string
	var reqStyle lipgloss.Style
	if isWaiting {
		req = session.Request
		if req == "" {
			req = string(session.PromptType) + " request"
		}
		req = strings.ReplaceAll(req, "\n", " ")
		req = strings.TrimSpace(req)
		reqStyle = statusWaiting
	} else if session.Info.IsWorking && session.Info.WorkingStatus != "" {
		req = session.Info.WorkingStatus
		reqStyle = statusWorking
	} else {
		req = "-"
		reqStyle = statusIdle
	}
	values[ColRequest] = ColumnValue{req, reqStyle}

	return values
}

// renderContextBar creates a visual progress bar for context size
func renderContextBar(ctxSize string, width int) string {
	if width <= 0 {
		width = 6 // default width
	}
	if ctxSize == "" {
		return lipgloss.NewStyle().Foreground(colorTextDim).Width(width).Render("-")
	}

	ctxSize = strings.TrimSpace(ctxSize)
	if strings.HasSuffix(ctxSize, "%") {
		var remainingPct float64
		_, _ = fmt.Sscanf(strings.TrimSuffix(ctxSize, "%"), "%f", &remainingPct)
		pct := remainingPct / 100.0
		if pct < 0 {
			pct = 0
		}
		if pct > 1.0 {
			pct = 1.0
		}

		filled := int(pct * float64(width))
		if filled > width {
			filled = width
		}

		bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

		var barStyle lipgloss.Style
		if pct < 0.2 {
			barStyle = lipgloss.NewStyle().Foreground(colorError)
		} else if pct < 0.4 {
			barStyle = lipgloss.NewStyle().Foreground(colorWarning)
		} else {
			barStyle = lipgloss.NewStyle().Foreground(colorSuccess)
		}

		return barStyle.Width(width).Render(bar)
	}

	// Claude reports an absolute context figure (for example "38.9k"), not a percent.
	// We do not know the active model's true max context from terminal output alone,
	// so showing the exact value is less misleading than a fake progress bar.
	return lipgloss.NewStyle().
		Foreground(colorInfo).
		Width(width).
		Align(lipgloss.Right).
		Render(ctxSize)
}

// smartTruncate truncates from the middle, preserving start and end
func smartTruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 10 {
		return s[:maxLen-3] + "..."
	}
	// Keep first part and last part
	keepStart := (maxLen - 3) / 2
	keepEnd := maxLen - 3 - keepStart
	return s[:keepStart] + "..." + s[len(s)-keepEnd:]
}

func (m Model) renderDetailPanel(width, height int) string {
	if m.cursor >= len(m.sessions) {
		return ""
	}

	session := m.sessions[m.cursor]
	if session.PromptType == detector.PromptUnknown {
		return ""
	}

	// Title with prompt type and preview hint
	promptType := strings.ToUpper(string(session.PromptType))
	title := statusWaiting.Bold(true).Render(fmt.Sprintf(" %s REQUEST ", promptType)) +
		detailLabelStyle.Render(" [p] expand")

	// Show the parsed request - this is the good content!
	var previewLines []string
	request := session.Request
	if request == "" {
		request = "(no request details extracted)"
	}

	// Wrap long requests to fit width
	maxLineWidth := width - 8
	reqLines := strings.Split(request, "\n")
	for _, line := range reqLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Word-wrap long lines
		for len(line) > maxLineWidth {
			previewLines = append(previewLines, "  "+statusWaiting.Render(line[:maxLineWidth]))
			line = line[maxLineWidth:]
		}
		if line != "" {
			previewLines = append(previewLines, "  "+statusWaiting.Render(line))
		}
	}

	// Limit to available height
	maxLines := height - 3
	if maxLines < 2 {
		maxLines = 2
	}
	if len(previewLines) > maxLines {
		previewLines = previewLines[:maxLines-1]
		previewLines = append(previewLines, detailLabelStyle.Render("  ...press [p] for full preview"))
	}

	content := strings.Join(previewLines, "\n")

	panel := panelStyle.Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, content),
	)

	return panel
}

type helpMode int

const (
	helpModeMinimal helpMode = iota
	helpModeCompact
	helpModeMedium
	helpModeFull
)

func helpModeForSize(width, height int, isMini bool) helpMode {
	if isMini || height < 10 {
		return helpModeMinimal
	}

	switch {
	case width >= 100:
		if height < 12 {
			return helpModeCompact
		}
		return helpModeFull
	case width >= 90:
		if height < 12 {
			return helpModeCompact
		}
		return helpModeMedium
	case width >= 70:
		return helpModeCompact
	default:
		return helpModeMinimal
	}
}

func (m Model) renderHelpBar(width int, mode helpMode, isMini bool) string {
	if width <= 0 {
		return ""
	}

	waiting := m.getWaitingSessions()

	// Responsive help bar - structured rows for readability.
	item := func(key, label string) string {
		if label == "" {
			return helpKeyStyle.Render(key)
		}
		return helpKeyStyle.Render(key) + " " + label
	}

	var lines [][]string
	switch mode {
	case helpModeFull:
		line1 := []string{}
		if len(waiting) > 0 {
			line1 = append(line1, item("[1-9]", "approve"), item("[!-)]", "always"))
		}
		line1 = append(line1,
			item("[a]", "approve"),
			item("[s]", "always"),
			item("[d]", "deny"),
			item("[i]", "input"),
			item("[p]", "preview"),
			item("[v]", "view"),
			item("[l]", "log"),
		)
		line2 := []string{
			item("[t]", "off/safe"),
			item("[T]", "cycle safe/nodel/all"),
			item("[x]", "nodel"),
			item("[b]", "burst until idle"),
			item("[R]", "rename"),
			item("[K]", "kill"),
			item("[g]", "git"),
			item("[M]", "agent"),
			item("[q]", "quit"),
		}
		line3 := []string{
			detailLabelStyle.Render("SAFE = approve except destructive ops"),
			detailLabelStyle.Render("NODEL = approve except delete ops"),
			detailLabelStyle.Render("ALL = approve every prompt"),
		}
		lines = [][]string{line1, line2, line3}
	case helpModeMedium:
		line1 := []string{
			item("[a]", "approve"),
			item("[s]", "always"),
			item("[d]", "deny"),
			item("[i]", "input"),
			item("[p]", "preview"),
			item("[v]", "view"),
			item("[l]", "log"),
		}
		line2 := []string{
			item("[t]", "off/safe"),
			item("[T]", "cycle modes"),
			item("[x]", "nodel"),
			item("[b]", "burst"),
			item("[R]", "rename"),
			item("[K]", "kill"),
			item("[g]", "git"),
			item("[M]", "agent"),
			item("[q]", "quit"),
		}
		lines = [][]string{line1, line2}
	case helpModeCompact:
		lines = append(lines, []string{
			helpKeyStyle.Render("a") + "/" + helpKeyStyle.Render("d") + " y/n",
			helpKeyStyle.Render("t") + " auto",
			helpKeyStyle.Render("p") + " preview",
			helpKeyStyle.Render("v") + " view",
			helpKeyStyle.Render("q") + " quit",
		})
	default:
		if isMini {
			lines = append(lines, []string{
				helpKeyStyle.Render("a") + "/" + helpKeyStyle.Render("d"),
				helpKeyStyle.Render("t"),
				helpKeyStyle.Render("v"),
				helpKeyStyle.Render("q"),
			})
		} else {
			lines = append(lines, []string{
				helpKeyStyle.Render("a") + "/" + helpKeyStyle.Render("d"),
				helpKeyStyle.Render("t"),
				helpKeyStyle.Render("p"),
				helpKeyStyle.Render("q"),
			})
		}
	}

	pad := 2
	if width < 50 {
		pad = 1
	}
	if width < 40 {
		pad = 0
	}
	wrapWidth := width - pad*2
	if wrapWidth < 1 {
		wrapWidth = 1
	}

	var rendered []string
	for _, items := range lines {
		if len(items) == 0 {
			continue
		}
		line := strings.Join(items, "  ")
		rendered = append(rendered, ansi.Wrap(line, wrapWidth, " "))
	}
	if len(rendered) == 0 {
		return ""
	}
	return helpStyle.Padding(0, pad).Render(strings.Join(rendered, "\n"))
}

// processControlFiles reads control files written by external tools (e.g. Claude Code skills)
// to set auto-approve modes. Each file is named after a session, contents = mode string.
func (m *Model) processControlFiles() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	controlDir := filepath.Join(home, ".config", "permission-guardian", "control")

	entries, err := os.ReadDir(controlDir)
	if err != nil {
		return // dir doesn't exist yet — normal
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		sessionName := entry.Name()
		filePath := filepath.Join(controlDir, sessionName)

		data, err := os.ReadFile(filePath)
		_ = os.Remove(filePath) // always clean up
		if err != nil {
			continue
		}

		modeStr := strings.TrimSpace(string(data))
		var mode AutoMode
		switch modeStr {
		case "safe":
			mode = AutoSafe
		case "all":
			mode = AutoAll
		case "nodelete":
			mode = AutoNoDelete
		case "off":
			mode = AutoOff
			delete(m.burstMode, sessionName)
		default:
			continue // unknown mode — skip silently
		}

		m.autoApprove[sessionName] = mode
		if mode == AutoOff {
			delete(m.autoApprove, sessionName)
		}
		if mode == AutoOff {
			m.actionStatus = autoModeStatus(AutoOff, sessionName) + " (via control file)"
		} else {
			m.actionStatus = autoModeStatus(mode, sessionName) + " (via control file)"
		}
		m.actionTime = time.Now()
	}
}
