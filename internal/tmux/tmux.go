// Package tmux provides functions for interacting with tmux sessions
package tmux

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Session represents a tmux session with its metadata
type Session struct {
	Name        string
	Activity    time.Time
	IdleSeconds int
	PanePID     int
	PaneContent string
}

// runCmd executes a command and returns stdout
func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// ListSessions returns all tmux sessions with their activity timestamps
func ListSessions() ([]Session, error) {
	output, err := runCmd("tmux", "list-sessions", "-F", "#{session_name}:#{session_activity}")
	if err != nil {
		return nil, err
	}

	if output == "" {
		return nil, nil
	}

	now := time.Now()
	var sessions []Session

	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		name := parts[0]
		activityTS, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}

		activity := time.Unix(activityTS, 0)
		idleSeconds := int(now.Sub(activity).Seconds())

		sessions = append(sessions, Session{
			Name:        name,
			Activity:    activity,
			IdleSeconds: idleSeconds,
		})
	}

	return sessions, nil
}

// GetPanePID returns the PID of the main pane in a session
func GetPanePID(sessionName string) (int, error) {
	output, err := runCmd("tmux", "list-panes", "-t", sessionName, "-F", "#{pane_pid}")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return 0, fmt.Errorf("no panes found")
	}

	return strconv.Atoi(lines[0])
}

// CapturePane captures the last N lines from a tmux pane (plain text)
func CapturePane(sessionName string, lines int) (string, error) {
	// Use -J to join wrapped lines and get full content
	return runCmd("tmux", "capture-pane", "-t", sessionName, "-p", "-J", "-S", fmt.Sprintf("-%d", lines))
}

// CapturePaneStyled captures the last N lines with ANSI escape codes preserved
func CapturePaneStyled(sessionName string, lines int) (string, error) {
	// Use -e to include escape sequences (ANSI colors)
	return runCmd("tmux", "capture-pane", "-t", sessionName, "-p", "-e", "-J", "-S", fmt.Sprintf("-%d", lines))
}

// GetPaneCWD returns the current working directory of a session's pane
// This comes from tmux's own tracking (via OSC 7), not from pane content
func GetPaneCWD(sessionName string) (string, error) {
	return runCmd("tmux", "list-panes", "-t", sessionName, "-F", "#{pane_current_path}")
}

// SendKeys sends keystrokes to a tmux session
func SendKeys(sessionName string, keys string) error {
	_, err := runCmd("tmux", "send-keys", "-t", sessionName, keys)
	return err
}

// SendEnter sends Enter key to a session (for confirming prompts)
func SendEnter(sessionName string) error {
	_, err := runCmd("tmux", "send-keys", "-t", sessionName, "Enter")
	return err
}

// IsRunning checks if tmux server is running
func IsRunning() bool {
	_, err := runCmd("tmux", "list-sessions")
	return err == nil
}
