// Package tui provides the terminal user interface for permission-guardian
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/buddyh/permission-guardian/internal/db"
	"github.com/buddyh/permission-guardian/internal/detector"
	"github.com/buddyh/permission-guardian/internal/tmux"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ═══════════════════════════════════════════════════════════════════════════
// Color Palette - Nord + Dracula inspired
// ═══════════════════════════════════════════════════════════════════════════
var (
	// Base colors
	colorBg       = lipgloss.Color("#1e2139")
	colorSurface  = lipgloss.Color("#282c47")
	colorSelected = lipgloss.Color("#3d4466") // More visible selection highlight
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
)

// ═══════════════════════════════════════════════════════════════════════════
// Styles
// ═══════════════════════════════════════════════════════════════════════════
var (
	// Base styles
	baseStyle = lipgloss.NewStyle().
			Background(colorBg)

	// Logo styles
	logoStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	logoSubStyle = lipgloss.NewStyle().
			Foreground(colorTextSec)

	// Header styles
	headerStyle = lipgloss.NewStyle().
			Background(colorSurface).
			Foreground(colorTextPri).
			Padding(0, 2)

	// Panel styles
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
			Foreground(colorTextSec).
			Bold(true).
			Padding(0, 1)

	// Session list styles
	sessionNameStyle = lipgloss.NewStyle().
				Foreground(colorInfo).
				Bold(true)

	sessionSelectedStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(colorTextPri).
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

	// Preview styles
	previewStyle = lipgloss.NewStyle().
			Foreground(colorTextPri).
			Background(colorSurface).
			Padding(0, 1)

	// Help bar styles
	helpStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Padding(0, 2)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorInfo)

	// Divider
	dividerStyle = lipgloss.NewStyle().
			Foreground(colorBorder)
)

// ASCII Logo
const logo = `
 ██████╗  ██████╗
 ██╔══██╗██╔════╝
 ██████╔╝██║  ███╗
 ██╔═══╝ ██║   ██║
 ██║     ╚██████╔╝
 ╚═╝      ╚═════╝ `

const logoText = `Permission
Guardian
───────────
AI Agent
Monitor`

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
	ToggleBurst    key.Binding
	ToggleView     key.Binding
	ViewLog        key.Binding
	Preview        key.Binding
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
		key.WithHelp("t", "auto on/off"),
	),
	ToggleAutoMode: key.NewBinding(
		key.WithKeys("T", "m"),
		key.WithHelp("T", "safe/all"),
	),
	ToggleBurst: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "burst mode"),
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
	AutoOff  AutoMode = iota // No auto-approve
	AutoSafe                 // Approve all except destructive commands
	AutoAll                  // Approve everything (yolo)
)

func (m AutoMode) String() string {
	switch m {
	case AutoSafe:
		return "SAFE"
	case AutoAll:
		return "ALL"
	default:
		return ""
	}
}

func (m AutoMode) Next() AutoMode {
	switch m {
	case AutoOff:
		return AutoSafe
	case AutoSafe:
		return AutoAll
	case AutoAll:
		return AutoOff
	default:
		return AutoOff
	}
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
	// View mode (compact/expanded)
	viewMode ViewMode
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
	defer f.Close()

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
	defer f.Close()

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
		m.spinner.Tick,
		fetchSessions,
		tickCmd(m.refreshRate),
	)
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchSessions() tea.Msg {
	sessions, err := detector.GetAllAgentSessions(50)
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If viewing log, any key exits log view
		if m.showLog {
			m.showLog = false
			return m, nil
		}

		// If viewing preview, any key exits preview view
		if m.showPreview {
			m.showPreview = false
			return m, nil
		}

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
						m.auditDB.LogDecision(db.Decision{
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
						m.auditDB.LogDecision(db.Decision{
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
						m.auditDB.LogDecision(db.Decision{
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
					m.actionStatus = fmt.Sprintf("AUTO SAFE: %s (skip destructive)", name)
				} else {
					// Turn off
					m.autoApprove[name] = AutoOff
					delete(m.burstMode, name)
					m.actionStatus = fmt.Sprintf("AUTO OFF: %s", name)
				}
				m.actionTime = time.Now()
			}

		case key.Matches(msg, keys.ToggleAutoMode):
			// Switch between SAFE and ALL (only when auto is already on)
			if m.cursor < len(m.sessions) {
				name := m.sessions[m.cursor].Session.Name
				currentMode := m.autoApprove[name]

				if currentMode == AutoSafe {
					m.autoApprove[name] = AutoAll
					m.actionStatus = fmt.Sprintf("AUTO ALL: %s (approve everything!)", name)
				} else if currentMode == AutoAll {
					m.autoApprove[name] = AutoSafe
					m.actionStatus = fmt.Sprintf("AUTO SAFE: %s (skip destructive)", name)
				} else {
					// Auto is off - inform user
					m.actionStatus = fmt.Sprintf("Auto is OFF - press [t] first: %s", name)
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
					m.actionStatus = fmt.Sprintf("BURST SAFE: %s (until idle)", name)
				} else if m.burstMode[name] {
					// Turn off burst
					delete(m.burstMode, name)
					m.actionStatus = fmt.Sprintf("BURST OFF: %s (still %s)", name, mode.String())
				} else {
					// Turn on burst
					m.burstMode[name] = true
					m.actionStatus = fmt.Sprintf("BURST %s: %s (until idle)", mode.String(), name)
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

		case key.Matches(msg, keys.ToggleView):
			// Cycle view mode
			m.viewMode = m.viewMode.Next()
			m.actionStatus = fmt.Sprintf("View: %s", m.viewMode.String())
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
						writeTaskLog(name, duration, approvals)
						// Log to SQLite
						if m.auditDB != nil {
							m.auditDB.LogTaskRun(db.TaskRun{
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

			if mode == AutoAll {
				// Approve everything
				shouldApprove = true
			} else if mode == AutoSafe {
				// Approve unless it's a destructive command
				if isDestructiveCommand(session.Request) {
					skipReason = "destructive"
				} else {
					shouldApprove = true
				}
			}

			if shouldApprove {
				writeLog(session.Session.Name, string(session.PromptType), session.Request)
				// Log to SQLite audit database
				if m.auditDB != nil {
					modeStr := mode.String()
					if m.burstMode[session.Session.Name] {
						modeStr += "+BURST"
					}
					m.auditDB.LogDecision(db.Decision{
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
					m.actionStatus = fmt.Sprintf("BURST: %s", session.Session.Name)
				} else {
					m.actionStatus = fmt.Sprintf("AUTO: %s", session.Session.Name)
				}
				m.actionTime = time.Now()
			} else if skipReason != "" {
				// Log skipped decisions too
				if m.auditDB != nil {
					m.auditDB.LogDecision(db.Decision{
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
				m.actionStatus = fmt.Sprintf("SKIP (%s): %s", skipReason, session.Session.Name)
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

// ═══════════════════════════════════════════════════════════════════════════
// View
// ═══════════════════════════════════════════════════════════════════════════
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press q to quit.\n", m.err)
	}

	width := m.width
	if width < 80 {
		width = 80
	}
	height := m.height
	if height < 15 {
		height = 15
	}

	// If showing log, render log view instead
	if m.showLog {
		return m.renderLogView(width, height)
	}

	var sections []string

	// Header with logo (8 lines)
	sections = append(sections, m.renderHeader(width))

	// Help bar (2 lines)
	helpBar := m.renderHelpBar(width)

	// Calculate available height for main content
	// Header ~8 lines, help ~2 lines
	availableHeight := height - 10
	if availableHeight < 10 {
		availableHeight = 10
	}

	if m.showPreview && m.cursor < len(m.sessions) {
		// Split view: sessions on top (~40%), preview on bottom (~60%)
		sessionsHeight := availableHeight * 2 / 5
		if sessionsHeight < 6 {
			sessionsHeight = 6
		}
		previewHeight := availableHeight - sessionsHeight

		// Render compact sessions table (no detail panel)
		sections = append(sections, m.renderSessionTable(width, sessionsHeight))

		// Render preview panel below
		sections = append(sections, m.renderSplitPreview(width, previewHeight))
	} else {
		// Normal view: full sessions with optional detail panel
		sections = append(sections, m.renderMainContent(width, availableHeight))
	}

	sections = append(sections, helpBar)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
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
	safeCount := 0
	allCount := 0
	for _, mode := range m.autoApprove {
		switch mode {
		case AutoSafe:
			safeCount++
		case AutoAll:
			allCount++
		}
	}

	lines = append(lines, "")
	lines = append(lines, dividerStyle.Render(strings.Repeat("─", width-4)))
	lines = append(lines, fmt.Sprintf("  Auto-approve: %d SAFE, %d ALL", safeCount, allCount))
	lines = append(lines, fmt.Sprintf("  Log file: %s", logFilePath()))
	lines = append(lines, "")
	lines = append(lines, detailLabelStyle.Render("  SAFE mode skips destructive commands (rm -rf, git push --force, etc.)"))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderPreviewView(width, height int) string {
	if m.cursor >= len(m.sessions) {
		m.showPreview = false
		return ""
	}

	session := m.sessions[m.cursor]

	// Title bar
	title := statusAccent.Render(fmt.Sprintf(" PREVIEW: %s (press any key to close) ", session.Session.Name))

	var lines []string
	lines = append(lines, "")
	lines = append(lines, title)
	lines = append(lines, "")

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

	lines = append(lines, "")
	lines = append(lines, dividerStyle.Render(strings.Repeat("─", width-4)))

	// Show request prominently if waiting
	if session.PromptType != detector.PromptUnknown {
		promptType := strings.ToUpper(string(session.PromptType))
		lines = append(lines, "")
		lines = append(lines, statusWaiting.Bold(true).Render(fmt.Sprintf("  %s REQUEST:", promptType)))
		lines = append(lines, "")

		// Show the extracted request (this is the good parsed content)
		reqLines := strings.Split(session.Request, "\n")
		for _, line := range reqLines {
			if len(line) > width-6 {
				line = line[:width-9] + "..."
			}
			lines = append(lines, "  "+statusWaiting.Render(line))
		}

		lines = append(lines, "")
		lines = append(lines, dividerStyle.Render(strings.Repeat("─", width-4)))
	}

	// Raw content - show as much as fits (with terminal styling preserved)
	lines = append(lines, "")
	lines = append(lines, detailLabelStyle.Render("  TERMINAL CONTENT:"))
	lines = append(lines, "")

	// Use StyledContent if available (preserves ANSI colors), fall back to RawContent
	contentToShow := session.StyledContent
	if contentToShow == "" {
		contentToShow = session.RawContent
	}

	rawLines := strings.Split(contentToShow, "\n")
	maxRawLines := height - len(lines) - 2
	if maxRawLines < 5 {
		maxRawLines = 5
	}
	if len(rawLines) > maxRawLines {
		rawLines = rawLines[len(rawLines)-maxRawLines:]
	}
	for _, line := range rawLines {
		// Don't truncate styled content - ANSI codes make length calculation wrong
		// Just add padding and render directly to preserve colors
		lines = append(lines, "  "+line)
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
	if maxLines < 3 {
		maxLines = 3
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

func (m Model) renderHeader(width int) string {
	// Logo section
	logoLines := strings.Split(logo, "\n")
	textLines := strings.Split(logoText, "\n")

	var headerLines []string
	maxLogoLines := max(len(logoLines), len(textLines))

	for i := 0; i < maxLogoLines; i++ {
		logoLine := ""
		textLine := ""
		if i < len(logoLines) {
			logoLine = logoLines[i]
		}
		if i < len(textLines) {
			textLine = textLines[i]
		}
		line := logoStyle.Render(fmt.Sprintf("%-20s", logoLine)) + "  " + logoSubStyle.Render(textLine)
		headerLines = append(headerLines, line)
	}

	logoBlock := lipgloss.JoinVertical(lipgloss.Left, headerLines...)

	// Stats section
	waiting := m.getWaitingSessions()
	totalSessions := len(m.sessions)
	waitingCount := len(waiting)

	var statusText string
	if waitingCount > 0 {
		statusText = statusWaiting.Render(fmt.Sprintf("  %d WAITING", waitingCount))
	} else {
		statusText = statusApproved.Render("  ALL CLEAR")
	}

	stats := fmt.Sprintf(
		"%s sessions %s %s",
		detailValueStyle.Render(fmt.Sprintf("%d", totalSessions)),
		statusText,
		m.spinner.View(),
	)

	// Action status
	actionLine := ""
	if m.actionStatus != "" {
		switch {
		case strings.HasPrefix(m.actionStatus, "approved+remembered"):
			actionLine = statusAccent.Render("  " + m.actionStatus + " (won't ask again)")
		case strings.HasPrefix(m.actionStatus, "approved"), strings.HasPrefix(m.actionStatus, "AUTO:"):
			actionLine = statusApproved.Render("  " + m.actionStatus)
		case strings.HasPrefix(m.actionStatus, "denied"):
			actionLine = statusDenied.Render("  " + m.actionStatus)
		case strings.HasPrefix(m.actionStatus, "AUTO SAFE"), strings.HasPrefix(m.actionStatus, "AUTO ALL"):
			actionLine = statusAuto.Render("  " + m.actionStatus)
		case strings.HasPrefix(m.actionStatus, "AUTO OFF"):
			actionLine = statusIdle.Render("  " + m.actionStatus)
		case strings.HasPrefix(m.actionStatus, "SKIP"):
			actionLine = statusWaiting.Render("  " + m.actionStatus)
		default:
			actionLine = detailValueStyle.Render("  " + m.actionStatus)
		}
	}

	statsBlock := lipgloss.JoinVertical(lipgloss.Left,
		"",
		"",
		stats,
		actionLine,
	)

	// Combine logo and stats
	header := lipgloss.JoinHorizontal(lipgloss.Top, logoBlock, "    ", statsBlock)

	return header + "\n"
}

func (m Model) renderMainContent(width, height int) string {
	// Check if we need detail panel (only for waiting sessions)
	showDetail := m.cursor < len(m.sessions) && m.sessions[m.cursor].PromptType != detector.PromptUnknown

	// Detail panel is compact - only 6 lines when shown
	detailHeight := 0
	if showDetail {
		detailHeight = 6
	}

	// Table gets all remaining height - sessions are priority!
	tableHeight := height - detailHeight
	if tableHeight < 5 {
		tableHeight = 5
	}

	// Full-width session table at top
	tablePanel := m.renderSessionTable(width, tableHeight)

	// Detail panel at bottom (only if waiting session selected)
	var detailPanel string
	if showDetail {
		detailPanel = m.renderDetailPanel(width, detailHeight)
	}

	return lipgloss.JoinVertical(lipgloss.Left, tablePanel, detailPanel)
}

func (m Model) renderSessionTable(width, height int) string {
	// Get view mode indicator for title
	viewIndicator := ""
	if m.viewMode == ViewExpanded {
		viewIndicator = " [Expanded]"
	}
	title := panelTitleStyle.Render("SESSIONS" + viewIndicator)

	// Get visible columns based on terminal width
	columns := GetVisibleColumns(width - 4) // -4 for panel padding

	// Build header
	sepStyle := lipgloss.NewStyle().Foreground(colorBorder)
	headerTextStyle := lipgloss.NewStyle().Foreground(colorTextDim).Bold(true)

	var headerCells []string
	var divParts []string
	for _, col := range columns {
		headerCells = append(headerCells, headerTextStyle.Width(col.Width).Render(col.Header))
		divParts = append(divParts, strings.Repeat("─", col.Width))
	}
	headerLine := " " + strings.Join(headerCells, sepStyle.Render(ColumnSeparator))
	divider := dividerStyle.Render(" " + strings.Join(divParts, "─┼─"))

	var lines []string
	lines = append(lines, headerLine)
	lines = append(lines, divider)

	waitingNum := 1
	for i, session := range m.sessions {
		isSelected := i == m.cursor
		rowLines := m.renderTableRow(session, isSelected, waitingNum, width-4, i, columns)
		lines = append(lines, rowLines...)

		if session.PromptType != detector.PromptUnknown {
			waitingNum++
		}
	}

	if len(m.sessions) == 0 {
		lines = append(lines, statusIdle.Render("  No sessions found"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	// NO padding - sessions get exactly the space they need
	// Only truncate if we have more lines than can fit
	contentLines := strings.Split(content, "\n")
	maxLines := height - 3
	if maxLines > 0 && len(contentLines) > maxLines {
		contentLines = contentLines[:maxLines]
	}
	content = strings.Join(contentLines, "\n")

	panel := panelStyle.Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, content),
	)

	return panel
}

func (m Model) renderTableRow(session detector.WaitingSession, selected bool, waitingNum int, width int, rowIndex int, columns []ColumnDef) []string {
	isWaiting := session.PromptType != detector.PromptUnknown
	sepStyle := lipgloss.NewStyle().Foreground(colorBorder)

	// Determine row background
	var rowBg lipgloss.Color
	if selected {
		rowBg = colorSelected
	} else if rowIndex%2 == 1 {
		rowBg = lipgloss.Color("#252840")
	} else {
		rowBg = colorBg
	}

	// Helper to apply background to a style
	withBg := func(s lipgloss.Style) lipgloss.Style {
		return s.Background(rowBg)
	}

	// Prepare all column values
	values := m.getColumnValues(session, isWaiting, waitingNum, columns)

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
	sep := withBg(sepStyle).Render(ColumnSeparator)
	row := strings.Join(cells, sep)

	// Add selection indicator or space prefix
	if selected {
		indicator := withBg(lipgloss.NewStyle()).Foreground(colorAccent).Bold(true).Render("▶")
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

// ColumnValue holds the text and style for a column cell
type ColumnValue struct {
	Text  string
	Style lipgloss.Style
}

// getColumnValues returns the display value and style for each column
func (m Model) getColumnValues(session detector.WaitingSession, isWaiting bool, waitingNum int, columns []ColumnDef) map[Column]ColumnValue {
	values := make(map[Column]ColumnValue)

	// NUM
	if isWaiting {
		values[ColNum] = ColumnValue{fmt.Sprintf("%d.", waitingNum), statusWaiting.Bold(true)}
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

	// STATUS
	autoMode := m.autoApprove[session.Session.Name]
	isBurst := m.burstMode[session.Session.Name]
	var statusText string
	var statusStyle lipgloss.Style
	if autoMode != AutoOff {
		if isBurst {
			statusText = autoMode.String() + "+B"
		} else {
			statusText = autoMode.String()
		}
		statusStyle = statusAuto
	} else if isWaiting {
		statusText = "WAITING"
		statusStyle = statusWaiting
	} else if session.Info.IsWorking {
		statusText = "working"
		statusStyle = statusWorking
	} else {
		statusText = "idle"
		statusStyle = statusIdle
	}
	values[ColStatus] = ColumnValue{statusText, statusStyle}

	// TIME - task duration
	var timeText string
	var timeStyle lipgloss.Style
	if startTime, exists := m.taskStartTime[session.Session.Name]; exists {
		duration := time.Since(startTime)
		timeText = formatDuration(duration)
		if duration > 30*time.Minute {
			timeStyle = statusWaiting // Yellow for long-running
		} else {
			timeStyle = statusWorking // Purple for active
		}
	} else {
		timeText = "-"
		timeStyle = statusIdle
	}
	values[ColTime] = ColumnValue{timeText, timeStyle}

	// MODEL
	model := strings.TrimSpace(session.Info.Model)
	if model == "" {
		model = "-"
	}
	modelWidth := GetColumnWidth(columns, ColModel)
	if modelWidth > 0 && len(model) > modelWidth {
		model = model[:modelWidth]
	}
	values[ColModel] = ColumnValue{model, detailLabelStyle}

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

	// REQUEST
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

	// Parse context size (e.g., "133.1k" -> 133100)
	var value float64
	ctxSize = strings.TrimSpace(ctxSize)
	if strings.HasSuffix(ctxSize, "k") {
		fmt.Sscanf(ctxSize, "%fk", &value)
		value *= 1000
	} else {
		fmt.Sscanf(ctxSize, "%f", &value)
	}

	// Assume max context is 200k, calculate percentage
	maxCtx := 200000.0
	pct := value / maxCtx
	if pct > 1.0 {
		pct = 1.0
	}

	// Create bar
	filled := int(pct * float64(width))
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	// Color based on usage
	var barStyle lipgloss.Style
	if pct > 0.8 {
		barStyle = lipgloss.NewStyle().Foreground(colorError)
	} else if pct > 0.6 {
		barStyle = lipgloss.NewStyle().Foreground(colorWarning)
	} else {
		barStyle = lipgloss.NewStyle().Foreground(colorSuccess)
	}

	return barStyle.Width(width).Render(bar)
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

func (m Model) renderSessionDetails(session detector.WaitingSession, width, height int) string {
	var lines []string

	// Session info
	lines = append(lines, m.detailLine("Session", session.Session.Name, width))
	lines = append(lines, m.detailLine("Agent", string(session.Agent), width))

	// Status
	var status string
	if session.PromptType != detector.PromptUnknown {
		status = statusWaiting.Render("WAITING FOR APPROVAL")
	} else if session.Info.IsWorking {
		workStatus := session.Info.WorkingStatus
		if workStatus == "" {
			workStatus = "working"
		}
		status = statusWorking.Render(workStatus)
	} else {
		status = statusIdle.Render("idle")
	}
	lines = append(lines, m.detailLine("Status", status, width))

	lines = append(lines, "")

	// Model info
	if session.Info.Model != "" {
		lines = append(lines, m.detailLine("Model", session.Info.Model, width))
	}
	if session.Info.ContextSize != "" {
		lines = append(lines, m.detailLine("Context", session.Info.ContextSize, width))
	}

	lines = append(lines, "")

	// Git info
	if session.Info.GitBranch != "" {
		gitInfo := session.Info.GitBranch
		if session.Info.GitChanges != "" {
			gitInfo += " " + session.Info.GitChanges
		}
		lines = append(lines, m.detailLine("Git", gitInfo, width))
	}

	// CWD
	if session.CWD != "" && session.CWD != "unknown" {
		cwd := session.CWD
		if strings.HasPrefix(cwd, "/Users/") {
			parts := strings.SplitN(cwd, "/", 4)
			if len(parts) >= 4 {
				cwd = "~/" + parts[3]
			}
		}
		if len(cwd) > width-12 {
			cwd = "..." + cwd[len(cwd)-(width-15):]
		}
		lines = append(lines, m.detailLine("Dir", cwd, width))
	}

	// Time info
	if session.Info.SessionTime != "" {
		lines = append(lines, m.detailLine("Session", session.Info.SessionTime, width))
	}
	if session.Info.BlockTime != "" {
		lines = append(lines, m.detailLine("Block", session.Info.BlockTime, width))
	}

	// Request preview (if waiting)
	if session.PromptType != detector.PromptUnknown {
		lines = append(lines, "")
		lines = append(lines, dividerStyle.Render(strings.Repeat("─", width)))
		lines = append(lines, "")

		promptType := strings.ToUpper(string(session.PromptType))
		lines = append(lines, statusWaiting.Render(fmt.Sprintf("  %s REQUEST", promptType)))
		lines = append(lines, "")

		// Raw content preview (use styled if available)
		contentToShow := session.StyledContent
		if contentToShow == "" {
			contentToShow = session.RawContent
		}
		previewLines := strings.Split(contentToShow, "\n")
		maxPreview := height - len(lines) - 2
		if maxPreview > 10 {
			maxPreview = 10
		}
		if len(previewLines) > maxPreview {
			previewLines = previewLines[len(previewLines)-maxPreview:]
		}

		for _, line := range previewLines {
			// Render directly to preserve ANSI colors
			lines = append(lines, "  "+line)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) detailLine(label, value string, width int) string {
	labelStr := detailLabelStyle.Render(fmt.Sprintf("  %-10s", label))
	valueStr := detailValueStyle.Render(value)
	return labelStr + valueStr
}

func (m Model) renderHelpBar(width int) string {
	waiting := m.getWaitingSessions()

	var items []string

	if len(waiting) > 0 {
		items = append(items, helpKeyStyle.Render("[1-9]")+" approve  "+helpKeyStyle.Render("[!-)]")+" always")
	}
	items = append(items, helpKeyStyle.Render("[a]")+"pprove")
	items = append(items, helpKeyStyle.Render("[s]")+" always")
	items = append(items, helpKeyStyle.Render("[d]")+"eny")
	items = append(items, helpKeyStyle.Render("[v]")+"iew")
	items = append(items, helpKeyStyle.Render("[p]")+"review")
	items = append(items, helpKeyStyle.Render("[t]")+" auto on/off")
	items = append(items, helpKeyStyle.Render("[T]")+" safe/all")
	items = append(items, helpKeyStyle.Render("[b]")+"urst")
	items = append(items, helpKeyStyle.Render("[l]")+"og")
	items = append(items, helpKeyStyle.Render("[q]")+"uit")

	help := strings.Join(items, "  ")

	return helpStyle.Render(help)
}
