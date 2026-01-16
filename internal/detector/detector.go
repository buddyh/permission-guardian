// Package detector identifies permission prompts in tmux sessions
package detector

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/buddyhadry/permission-guardian/internal/tmux"
)

// PromptType represents the type of permission being requested
type PromptType string

const (
	PromptBash    PromptType = "bash"
	PromptFetch   PromptType = "fetch"
	PromptEdit    PromptType = "edit"
	PromptWrite   PromptType = "write"
	PromptMCP     PromptType = "mcp"
	PromptTask    PromptType = "task"
	PromptUnknown PromptType = "unknown"
)

// AgentType represents the AI agent running in the session
type AgentType string

const (
	AgentClaude  AgentType = "claude"
	AgentCodex   AgentType = "codex"
	AgentUnknown AgentType = "unknown"
)

// SessionInfo contains parsed metadata from Claude Code status line
type SessionInfo struct {
	Model         string // "Opus 4.5", "Sonnet", etc.
	ContextSize   string // "38.9k"
	GitBranch     string // "main"
	GitChanges    string // "+128,-26"
	SessionTime   string // "10m", "2hr 5m"
	BlockTime     string // "3hr 9m"
	LastUserInput string // Last thing user typed
	IsWorking     bool   // True if actively processing (shows "...ing" with ctrl+c)
	WorkingStatus string // e.g., "Manifesting...", "Thinking...", "Reading..."
}

// WaitingSession represents a session waiting for permission approval
type WaitingSession struct {
	Session     tmux.Session
	Agent       AgentType
	PromptType  PromptType
	Request     string
	RawContent  string
	CWD         string
	Info        SessionInfo
}

// Status represents the current status of a session
type Status string

const (
	StatusWaiting Status = "waiting"
	StatusWorking Status = "working"
	StatusIdle    Status = "idle"
)

// Patterns that indicate an active permission prompt
var promptPatterns = []*regexp.Regexp{
	regexp.MustCompile(`Do you want to proceed\?`),
	regexp.MustCompile(`Do you want to allow`),
	regexp.MustCompile(`Would you like to run`),
	regexp.MustCompile(`❯ 1\. Yes`),
	regexp.MustCompile(`Yes, and don't ask again`),
	regexp.MustCompile(`No, and tell Claude`),
	regexp.MustCompile(`Do you want to create`),
}

// Note: Claude Code status line uses non-breaking spaces (U+00A0) as separators
// We need patterns that stop at NBSP or regular space
var (
	// Match path after "cwd:" stopping at nbsp or regular space or Session:
	cwdPattern        = regexp.MustCompile(`cwd:[\s\x{00A0}]*(/[^\s\x{00A0}]+)`)
	modelPattern      = regexp.MustCompile(`Model:[\s\x{00A0}]*([^\s\x{00A0}]+(?:[\s\x{00A0}]+\d+\.?\d*)?)`)
	contextPattern    = regexp.MustCompile(`Ctx:[\s\x{00A0}]*([^\s\x{00A0}]+)`)
	gitBranchPattern  = regexp.MustCompile(`⎇[\s\x{00A0}]*([^\s\x{00A0}]+)`)
	gitChangesPattern = regexp.MustCompile(`\(([+-]\d+,[+-]?\d+)\)`)
	sessionPattern    = regexp.MustCompile(`Session:[\s\x{00A0}]*([^\s\x{00A0}]+)`)
	blockPattern      = regexp.MustCompile(`Block:[\s\x{00A0}]*([^\s\x{00A0}]+(?:[\s\x{00A0}]+\d+\w)?)`)
	userInputPattern  = regexp.MustCompile(`❯[\s\x{00A0}]*(.+)`)
	// Working status: "✽ Manifesting…" or "⏺ Reading..." with "(ctrl+c to interrupt)"
	workingPattern = regexp.MustCompile(`[✽⏺][\s\x{00A0}]*(\w+(?:ing|\.\.\.)[…\.]*)`)
	// Compacting status: "· Compacting conversation…"
	compactingPattern = regexp.MustCompile(`·[\s\x{00A0}]*(Compacting[^…\n]*[…\.]+)`)
	ctrlCPattern      = regexp.MustCompile(`\(ctrl\+c to interrupt`)
)

// DetectAgent checks if a pane is running Claude Code or Codex
func DetectAgent(panePID int) AgentType {
	// Check the process itself
	cmd := getProcessCommand(panePID)
	cmdLower := strings.ToLower(cmd)

	if strings.Contains(cmdLower, "claude") {
		return AgentClaude
	}
	if strings.Contains(cmdLower, "codex") {
		return AgentCodex
	}

	// Check child processes
	children := getChildCommands(panePID)
	for _, child := range children {
		childLower := strings.ToLower(child)
		if strings.Contains(childLower, "claude") {
			return AgentClaude
		}
		if strings.Contains(childLower, "codex") {
			return AgentCodex
		}
	}

	return AgentUnknown
}

func getProcessCommand(pid int) string {
	out, err := exec.Command("ps", "-o", "command=", "-p", fmt.Sprintf("%d", pid)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func getChildCommands(pid int) []string {
	out, err := exec.Command("pgrep", "-P", fmt.Sprintf("%d", pid)).Output()
	if err != nil {
		return nil
	}

	var cmds []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		cmd := getProcessCommand(mustAtoi(line))
		if cmd != "" {
			cmds = append(cmds, cmd)
		}
	}
	return cmds
}

func mustAtoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// HasPermissionPrompt checks if content contains a permission prompt
func HasPermissionPrompt(content string) bool {
	for _, pattern := range promptPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// DetectPromptType determines what type of permission is being requested
func DetectPromptType(content string) PromptType {
	if strings.Contains(content, "Bash command") || strings.Contains(content, "Bash(") {
		return PromptBash
	}
	if strings.Contains(content, "wants to fetch") || strings.Contains(content, "Fetch(") {
		return PromptFetch
	}
	if strings.Contains(content, "Edit(") || strings.Contains(content, "make this edit to") {
		return PromptEdit
	}
	if strings.Contains(content, "Write(") || strings.Contains(content, "Do you want to create") {
		return PromptWrite
	}
	if strings.Contains(content, "MCP") {
		return PromptMCP
	}
	if strings.Contains(content, "Task(") {
		return PromptTask
	}
	return PromptUnknown
}

// ExtractRequest extracts the request description from pane content
func ExtractRequest(content string, promptType PromptType) string {
	lines := strings.Split(content, "\n")

	switch promptType {
	case PromptBash:
		return extractBashRequest(lines)
	case PromptFetch:
		return extractFetchRequest(lines)
	case PromptWrite:
		return extractWriteRequest(lines)
	case PromptEdit:
		return extractEditRequest(lines)
	}

	return extractFallbackRequest(lines)
}

func extractBashRequest(lines []string) string {
	var result []string
	inBlock := false

	for _, line := range lines {
		if strings.Contains(line, "Bash command") {
			inBlock = true
		}
		if inBlock {
			if strings.Contains(line, "Do you want") {
				break
			}
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}

	if len(result) > 10 {
		result = result[:10]
	}
	return strings.Join(result, " ")
}

func extractFetchRequest(lines []string) string {
	for i, line := range lines {
		if strings.Contains(line, "wants to fetch") {
			start := max(0, i-1)
			end := min(len(lines), i+3)
			return strings.Join(lines[start:end], " ")
		}
	}
	return ""
}

func extractWriteRequest(lines []string) string {
	for _, line := range lines {
		if strings.Contains(line, "Do you want to create") {
			return strings.TrimSpace(line)
		}
	}
	for i, line := range lines {
		if strings.Contains(line, "Write(") {
			start := max(0, i-1)
			end := min(len(lines), i+3)
			return strings.Join(lines[start:end], " ")
		}
	}
	return ""
}

func extractEditRequest(lines []string) string {
	// Check for "make this edit to" pattern first
	for _, line := range lines {
		if strings.Contains(line, "make this edit to") {
			return strings.TrimSpace(line)
		}
	}
	// Then check for Edit( style
	for i, line := range lines {
		if strings.Contains(line, "Edit(") {
			start := max(0, i-1)
			end := min(len(lines), i+3)
			return strings.Join(lines[start:end], " ")
		}
	}
	return ""
}

func extractFallbackRequest(lines []string) string {
	for i, line := range lines {
		if strings.Contains(line, "Do you want") {
			start := max(0, i-5)
			return strings.Join(lines[start:i], " ")
		}
	}
	return ""
}

// ExtractCWD extracts the working directory from pane content
func ExtractCWD(content string) string {
	match := cwdPattern.FindStringSubmatch(content)
	if len(match) > 1 {
		return match[1]
	}
	return "unknown"
}

// ExtractSessionInfo parses Claude Code status line and other metadata
func ExtractSessionInfo(content string) SessionInfo {
	info := SessionInfo{}

	// Model (e.g., "Opus 4.5", "Sonnet")
	if match := modelPattern.FindStringSubmatch(content); len(match) > 1 {
		info.Model = match[1]
	}

	// Context size (e.g., "38.9k")
	if match := contextPattern.FindStringSubmatch(content); len(match) > 1 {
		info.ContextSize = match[1]
	}

	// Git branch (e.g., "main")
	if match := gitBranchPattern.FindStringSubmatch(content); len(match) > 1 {
		info.GitBranch = match[1]
	}

	// Git changes (e.g., "+128,-26")
	if match := gitChangesPattern.FindStringSubmatch(content); len(match) > 1 {
		info.GitChanges = match[1]
	}

	// Session time (e.g., "10m", "2hr")
	if match := sessionPattern.FindStringSubmatch(content); len(match) > 1 {
		info.SessionTime = match[1]
	}

	// Block time (e.g., "3hr 9m")
	if match := blockPattern.FindStringSubmatch(content); len(match) > 1 {
		info.BlockTime = match[1]
	}

	// Last user input - find the last line starting with ❯
	lines := strings.Split(content, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "❯") {
			if match := userInputPattern.FindStringSubmatch(line); len(match) > 1 {
				input := strings.TrimSpace(match[1])
				// Skip if it's just arrow indicators or empty
				if input != "" && !strings.HasPrefix(input, "↵") {
					info.LastUserInput = input
					break
				}
			}
		}
	}

	// Working status detection - look for "✽ Something..." with "(ctrl+c to interrupt)"
	if ctrlCPattern.MatchString(content) {
		info.IsWorking = true
		if match := workingPattern.FindStringSubmatch(content); len(match) > 1 {
			info.WorkingStatus = match[1]
		}
	}

	// Compacting status detection - "· Compacting conversation…"
	if match := compactingPattern.FindStringSubmatch(content); len(match) > 1 {
		info.IsWorking = true
		info.WorkingStatus = match[1]
	}

	return info
}

// GetWaitingSessions returns all sessions waiting for permission approval
func GetWaitingSessions(minIdleSeconds int, captureLines int) ([]WaitingSession, error) {
	sessions, err := tmux.ListSessions()
	if err != nil {
		return nil, err
	}

	var waiting []WaitingSession

	for _, session := range sessions {
		// Check idle time
		if session.IdleSeconds < minIdleSeconds {
			continue
		}

		// Get pane PID
		panePID, err := tmux.GetPanePID(session.Name)
		if err != nil {
			continue
		}

		// Check if it's Claude or Codex
		agent := DetectAgent(panePID)
		if agent == AgentUnknown {
			continue
		}

		// Capture pane content
		content, err := tmux.CapturePane(session.Name, captureLines)
		if err != nil {
			continue
		}

		// Check for permission prompt
		if !HasPermissionPrompt(content) {
			continue
		}

		promptType := DetectPromptType(content)
		request := ExtractRequest(content, promptType)
		cwd := ExtractCWD(content)
		info := ExtractSessionInfo(content)

		// Get last 20 lines for raw content
		lines := strings.Split(content, "\n")
		rawStart := max(0, len(lines)-20)
		rawContent := strings.Join(lines[rawStart:], "\n")

		waiting = append(waiting, WaitingSession{
			Session:    session,
			Agent:      agent,
			PromptType: promptType,
			Request:    strings.TrimSpace(request),
			RawContent: rawContent,
			CWD:        cwd,
			Info:       info,
		})
	}

	return waiting, nil
}

// GetAllAgentSessions returns all Claude/Codex sessions with their status
func GetAllAgentSessions(captureLines int) ([]WaitingSession, error) {
	sessions, err := tmux.ListSessions()
	if err != nil {
		return nil, err
	}

	var agentSessions []WaitingSession

	for _, session := range sessions {
		// Get pane PID
		panePID, err := tmux.GetPanePID(session.Name)
		if err != nil {
			continue
		}

		// Check if it's Claude or Codex
		agent := DetectAgent(panePID)
		if agent == AgentUnknown {
			continue
		}

		// Capture pane content
		content, err := tmux.CapturePane(session.Name, captureLines)
		if err != nil {
			continue
		}

		promptType := PromptUnknown
		request := ""

		if HasPermissionPrompt(content) {
			promptType = DetectPromptType(content)
			request = ExtractRequest(content, promptType)
		}

		cwd := ExtractCWD(content)
		info := ExtractSessionInfo(content)

		// Get last 20 lines for raw content
		lines := strings.Split(content, "\n")
		rawStart := max(0, len(lines)-20)
		rawContent := strings.Join(lines[rawStart:], "\n")

		agentSessions = append(agentSessions, WaitingSession{
			Session:    session,
			Agent:      agent,
			PromptType: promptType,
			Request:    strings.TrimSpace(request),
			RawContent: rawContent,
			CWD:        cwd,
			Info:       info,
		})
	}

	return agentSessions, nil
}
