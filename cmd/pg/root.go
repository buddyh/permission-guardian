package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/buddyh/permission-guardian/internal/db"
	"github.com/buddyh/permission-guardian/internal/detector"
	"github.com/buddyh/permission-guardian/internal/rules"
	"github.com/buddyh/permission-guardian/internal/tmux"
	"github.com/buddyh/permission-guardian/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var version = "dev"

type rootFlags struct {
	asJSON bool
}

func execute(args []string) error {
	var flags rootFlags

	rootCmd := &cobra.Command{
		Use:     "pg",
		Short:   "Permission Guardian - Monitor and manage Claude Code/Codex permission prompts",
		Version: version,
		Long: `Permission Guardian watches your tmux sessions running Claude Code or Codex,
detects when they're waiting for permission approval, and lets you approve or deny
from a single dashboard.

Run 'pg watch' to start the live dashboard, or 'pg list' for a quick check.`,
	}

	rootCmd.PersistentFlags().BoolVar(&flags.asJSON, "json", false, "Output as JSON")

	rootCmd.AddCommand(newWatchCmd(&flags))
	rootCmd.AddCommand(newListCmd(&flags))
	rootCmd.AddCommand(newApproveCmd(&flags))
	rootCmd.AddCommand(newDenyCmd(&flags))
	rootCmd.AddCommand(newAutoCmd(&flags))
	rootCmd.AddCommand(newRulesCmd(&flags))
	rootCmd.AddCommand(newLogCmd(&flags))

	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}

func newWatchCmd(flags *rootFlags) *cobra.Command {
	var refreshSec int

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Start the live dashboard",
		Long:  "Launch an interactive TUI that monitors all Claude/Codex sessions and allows quick approval/denial.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !tmux.IsRunning() {
				return fmt.Errorf("tmux is not running")
			}

			refreshRate := time.Duration(refreshSec) * time.Second
			model := tui.New(refreshRate)

			p := tea.NewProgram(model, tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}

	cmd.Flags().IntVarP(&refreshSec, "refresh", "r", 2, "Refresh interval in seconds")

	return cmd
}

func newListCmd(flags *rootFlags) *cobra.Command {
	var idleMin int
	var summary bool
	var countOnly bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sessions waiting for approval",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !tmux.IsRunning() {
				if flags.asJSON {
					fmt.Println("[]")
					return nil
				}
				return fmt.Errorf("tmux is not running")
			}

			sessions, err := detector.GetWaitingSessions(idleMin*60, 50)
			if err != nil {
				return err
			}

			if countOnly {
				fmt.Println(len(sessions))
				if len(sessions) > 255 {
					os.Exit(255)
				}
				os.Exit(len(sessions))
			}

			if flags.asJSON {
				return outputJSON(sessions)
			}

			if summary {
				return outputSummary(sessions)
			}

			return outputHuman(sessions)
		},
	}

	cmd.Flags().IntVar(&idleMin, "idle-min", 0, "Minimum idle time in minutes")
	cmd.Flags().BoolVar(&summary, "summary", false, "Ultra-compact one-line-per-session output")
	cmd.Flags().BoolVar(&countOnly, "count", false, "Output count only (exit code = count)")

	return cmd
}

func newApproveCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve [session]",
		Short: "Approve a waiting session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionName := args[0]

			// Send "1" then Enter
			if err := tmux.SendKeys(sessionName, "1"); err != nil {
				return fmt.Errorf("failed to send keys: %w", err)
			}
			time.Sleep(100 * time.Millisecond)
			if err := tmux.SendEnter(sessionName); err != nil {
				return fmt.Errorf("failed to send enter: %w", err)
			}

			if flags.asJSON {
				fmt.Printf(`{"success": true, "session": "%s", "action": "approved"}%s`, sessionName, "\n")
			} else {
				fmt.Printf("Approved: %s\n", sessionName)
			}
			return nil
		},
	}

	return cmd
}

func newDenyCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deny [session]",
		Short: "Deny/cancel a waiting session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionName := args[0]

			// Send Escape to cancel
			if err := tmux.SendKeys(sessionName, "Escape"); err != nil {
				return fmt.Errorf("failed to send keys: %w", err)
			}

			if flags.asJSON {
				fmt.Printf(`{"success": true, "session": "%s", "action": "denied"}%s`, sessionName, "\n")
			} else {
				fmt.Printf("Denied: %s\n", sessionName)
			}
			return nil
		},
	}

	return cmd
}

type jsonSessionInfo struct {
	Model         string `json:"model,omitempty"`
	ContextSize   string `json:"context_size,omitempty"`
	GitBranch     string `json:"git_branch,omitempty"`
	GitChanges    string `json:"git_changes,omitempty"`
	SessionTime   string `json:"session_time,omitempty"`
	BlockTime     string `json:"block_time,omitempty"`
	LastUserInput string `json:"last_user_input,omitempty"`
	IsWorking     bool   `json:"is_working"`
	WorkingStatus string `json:"working_status,omitempty"`
}

type jsonSession struct {
	Session     string          `json:"session"`
	Agent       string          `json:"agent"`
	IdleMinutes int             `json:"idle_minutes"`
	CWD         string          `json:"cwd"`
	PromptType  string          `json:"prompt_type"`
	Request     string          `json:"request"`
	Info        jsonSessionInfo `json:"info"`
	RawContent  string          `json:"raw_content"`
}

func outputJSON(sessions []detector.WaitingSession) error {
	out := make([]jsonSession, 0) // Initialize to empty slice, not nil
	for _, s := range sessions {
		out = append(out, jsonSession{
			Session:     s.Session.Name,
			Agent:       string(s.Agent),
			IdleMinutes: s.Session.IdleSeconds / 60,
			CWD:         s.CWD,
			PromptType:  string(s.PromptType),
			Request:     s.Request,
			Info: jsonSessionInfo{
				Model:         s.Info.Model,
				ContextSize:   s.Info.ContextSize,
				GitBranch:     s.Info.GitBranch,
				GitChanges:    s.Info.GitChanges,
				SessionTime:   s.Info.SessionTime,
				BlockTime:     s.Info.BlockTime,
				LastUserInput: s.Info.LastUserInput,
				IsWorking:     s.Info.IsWorking,
				WorkingStatus: s.Info.WorkingStatus,
			},
			RawContent: s.RawContent,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputSummary(sessions []detector.WaitingSession) error {
	for _, s := range sessions {
		request := s.Request
		if len(request) > 60 {
			request = request[:57] + "..."
		}
		fmt.Printf("%s|%s|%dm|%s\n",
			s.Session.Name,
			s.PromptType,
			s.Session.IdleSeconds/60,
			request,
		)
	}
	return nil
}

func outputHuman(sessions []detector.WaitingSession) error {
	if len(sessions) == 0 {
		fmt.Println("No sessions waiting for approval.")
		return nil
	}

	fmt.Println("=== Sessions Waiting for Approval ===")
	fmt.Println()

	for _, s := range sessions {
		fmt.Printf("Session: %s (%s)\n", s.Session.Name, s.Agent)
		fmt.Printf("Idle: %dm | Dir: %s\n", s.Session.IdleSeconds/60, s.CWD)
		fmt.Println("Request:")

		request := s.Request
		if len(request) > 200 {
			request = request[:197] + "..."
		}
		fmt.Printf("  %s\n", request)
		fmt.Println()
		fmt.Println("---")
		fmt.Println()
	}

	return nil
}

func newAutoCmd(flags *rootFlags) *cobra.Command {
	var configPath string
	var interval int
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "auto",
		Short: "Run auto-approval daemon based on rules",
		Long: `Start a daemon that automatically approves or denies permission prompts
based on rules defined in ~/.config/permission-guardian/rules.yaml

Use 'pg rules init' to create a default config file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !tmux.IsRunning() {
				return fmt.Errorf("tmux is not running")
			}

			config, err := rules.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load rules: %w", err)
			}

			matcher, err := rules.NewMatcher(config)
			if err != nil {
				return fmt.Errorf("failed to compile rules: %w", err)
			}

			enabledCount := 0
			for _, r := range config.Rules {
				if r.Enabled {
					enabledCount++
				}
			}

			if enabledCount == 0 {
				fmt.Println("No enabled rules found. Use 'pg rules init' to create defaults,")
				fmt.Println("then 'pg rules enable <name>' to enable specific rules.")
				return nil
			}

			fmt.Printf("Auto-approval daemon started with %d enabled rules\n", enabledCount)
			if dryRun {
				fmt.Println("DRY RUN MODE - no actions will be taken")
			}
			fmt.Printf("Checking every %d seconds. Press Ctrl+C to stop.\n\n", interval)

			// Handle graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			ticker := time.NewTicker(time.Duration(interval) * time.Second)
			defer ticker.Stop()

			// Run immediately, then on interval
			runAutoApprove(matcher, flags.asJSON, dryRun)

			for {
				select {
				case <-ticker.C:
					runAutoApprove(matcher, flags.asJSON, dryRun)
				case <-sigChan:
					fmt.Println("\nStopping auto-approval daemon...")
					return nil
				}
			}
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to rules config file")
	cmd.Flags().IntVarP(&interval, "interval", "i", 2, "Check interval in seconds")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be approved without acting")

	return cmd
}

func runAutoApprove(matcher *rules.Matcher, asJSON bool, dryRun bool) {
	sessions, err := detector.GetWaitingSessions(0, 50)
	if err != nil {
		fmt.Printf("Error getting sessions: %v\n", err)
		return
	}

	for _, session := range sessions {
		result := matcher.Match(session)
		if !result.Matched {
			continue
		}

		timestamp := time.Now().Format("15:04:05")

		if dryRun {
			fmt.Printf("[%s] Would %s: %s (rule: %s)\n",
				timestamp, result.Action, session.Session.Name, result.Rule.Name)
			continue
		}

		switch result.Action {
		case "approve":
			if err := tmux.SendKeys(session.Session.Name, "1"); err != nil {
				fmt.Printf("[%s] Error approving %s: %v\n", timestamp, session.Session.Name, err)
				continue
			}
			time.Sleep(100 * time.Millisecond)
			if err := tmux.SendEnter(session.Session.Name); err != nil {
				fmt.Printf("[%s] Error approving %s: %v\n", timestamp, session.Session.Name, err)
				continue
			}
			if asJSON {
				fmt.Printf(`{"time":"%s","action":"approved","session":"%s","rule":"%s"}`+"\n",
					timestamp, session.Session.Name, result.Rule.Name)
			} else {
				fmt.Printf("[%s] Approved: %s (rule: %s)\n",
					timestamp, session.Session.Name, result.Rule.Name)
			}

		case "deny":
			if err := tmux.SendKeys(session.Session.Name, "Escape"); err != nil {
				fmt.Printf("[%s] Error denying %s: %v\n", timestamp, session.Session.Name, err)
				continue
			}
			if asJSON {
				fmt.Printf(`{"time":"%s","action":"denied","session":"%s","rule":"%s"}`+"\n",
					timestamp, session.Session.Name, result.Rule.Name)
			} else {
				fmt.Printf("[%s] Denied: %s (rule: %s)\n",
					timestamp, session.Session.Name, result.Rule.Name)
			}
		}
	}
}

func newRulesCmd(flags *rootFlags) *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Manage auto-approval rules",
		Long:  `View, create, and manage auto-approval rules.`,
	}

	// rules init
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize default rules config",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := configPath
			if path == "" {
				path = rules.DefaultConfigPath()
			}

			// Check if file already exists
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("config already exists at %s\nUse --config to specify a different path", path)
			}

			config := rules.CreateDefaultConfig()
			if err := rules.SaveConfig(config, path); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Created default rules config at: %s\n", path)
			fmt.Println("\nDefault rules (all disabled):")
			for _, r := range config.Rules {
				fmt.Printf("  - %s: %s\n", r.Name, r.Description)
			}
			fmt.Println("\nUse 'pg rules enable <name>' to enable a rule.")
			return nil
		},
	}

	// rules list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := rules.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load rules: %w", err)
			}

			if len(config.Rules) == 0 {
				fmt.Println("No rules found. Use 'pg rules init' to create defaults.")
				return nil
			}

			if flags.asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(config.Rules)
			}

			fmt.Println("Auto-approval rules:")
			fmt.Println()
			for _, r := range config.Rules {
				status := "disabled"
				if r.Enabled {
					status = "ENABLED"
				}
				fmt.Printf("  [%s] %s\n", status, r.Name)
				if r.Description != "" {
					fmt.Printf("      %s\n", r.Description)
				}
				fmt.Printf("      Action: %s\n", r.Action)
				if len(r.Match.PromptTypes) > 0 {
					fmt.Printf("      Prompt types: %v\n", r.Match.PromptTypes)
				}
				if len(r.Match.Commands) > 0 {
					fmt.Printf("      Command patterns: %v\n", r.Match.Commands)
				}
				if len(r.Match.Sessions) > 0 {
					fmt.Printf("      Session patterns: %v\n", r.Match.Sessions)
				}
				fmt.Println()
			}
			return nil
		},
	}

	// rules enable
	enableCmd := &cobra.Command{
		Use:   "enable <rule-name>",
		Short: "Enable a rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setRuleEnabled(configPath, args[0], true)
		},
	}

	// rules disable
	disableCmd := &cobra.Command{
		Use:   "disable <rule-name>",
		Short: "Disable a rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setRuleEnabled(configPath, args[0], false)
		},
	}

	// rules add
	addCmd := &cobra.Command{
		Use:   "add <name> --session <pattern> --type <type> --action <approve|deny>",
		Short: "Add a new rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("rule name required")
			}

			name := args[0]
			sessionPattern, _ := cmd.Flags().GetString("session")
			promptType, _ := cmd.Flags().GetString("type")
			commandPattern, _ := cmd.Flags().GetString("command")
			cwdPattern, _ := cmd.Flags().GetString("cwd")
			action, _ := cmd.Flags().GetString("action")
			description, _ := cmd.Flags().GetString("description")

			if action != "approve" && action != "deny" {
				return fmt.Errorf("action must be 'approve' or 'deny'")
			}

			config, err := rules.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load rules: %w", err)
			}

			// Check for duplicate name
			for _, r := range config.Rules {
				if r.Name == name {
					return fmt.Errorf("rule '%s' already exists", name)
				}
			}

			rule := rules.Rule{
				Name:        name,
				Description: description,
				Enabled:     true, // New rules are enabled by default
				Action:      action,
				Match:       rules.Match{},
			}

			if sessionPattern != "" {
				rule.Match.Sessions = []string{sessionPattern}
			}
			if promptType != "" {
				rule.Match.PromptTypes = []string{promptType}
			}
			if commandPattern != "" {
				rule.Match.Commands = []string{commandPattern}
			}
			if cwdPattern != "" {
				rule.Match.CWDs = []string{cwdPattern}
			}

			config.Rules = append(config.Rules, rule)

			path := configPath
			if path == "" {
				path = rules.DefaultConfigPath()
			}

			if err := rules.SaveConfig(config, path); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Added rule '%s' (enabled)\n", name)
			return nil
		},
	}
	addCmd.Flags().String("session", "", "Session name pattern (glob)")
	addCmd.Flags().String("type", "", "Prompt type (bash, edit, write, fetch, mcp)")
	addCmd.Flags().String("command", "", "Command/request pattern (regex)")
	addCmd.Flags().String("cwd", "", "Working directory pattern (glob)")
	addCmd.Flags().String("action", "approve", "Action to take (approve or deny)")
	addCmd.Flags().String("description", "", "Rule description")

	// rules delete
	deleteCmd := &cobra.Command{
		Use:   "delete <rule-name>",
		Short: "Delete a rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := rules.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load rules: %w", err)
			}

			found := false
			newRules := make([]rules.Rule, 0, len(config.Rules))
			for _, r := range config.Rules {
				if r.Name == args[0] {
					found = true
					continue
				}
				newRules = append(newRules, r)
			}

			if !found {
				return fmt.Errorf("rule '%s' not found", args[0])
			}

			config.Rules = newRules

			path := configPath
			if path == "" {
				path = rules.DefaultConfigPath()
			}

			if err := rules.SaveConfig(config, path); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Deleted rule '%s'\n", args[0])
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to rules config file")

	cmd.AddCommand(initCmd, listCmd, enableCmd, disableCmd, addCmd, deleteCmd)

	return cmd
}

func setRuleEnabled(configPath string, name string, enabled bool) error {
	config, err := rules.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	found := false
	for i := range config.Rules {
		if config.Rules[i].Name == name {
			config.Rules[i].Enabled = enabled
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("rule '%s' not found", name)
	}

	path := configPath
	if path == "" {
		path = rules.DefaultConfigPath()
	}

	if err := rules.SaveConfig(config, path); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	action := "Disabled"
	if enabled {
		action = "Enabled"
	}
	fmt.Printf("%s rule '%s'\n", action, name)
	return nil
}

func newLogCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "View and search the audit log",
		Long:  "View recent decisions, search the audit log, and see statistics.",
	}

	// Subcommand: log show (default, shows recent entries)
	var limit int
	var session string
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show recent log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			auditDB, err := db.Open()
			if err != nil {
				return fmt.Errorf("failed to open audit database: %w", err)
			}
			defer auditDB.Close()

			var decisions []db.Decision
			if session != "" {
				decisions, err = auditDB.GetDecisionsBySession(session, limit)
			} else {
				decisions, err = auditDB.GetRecentDecisions(limit)
			}
			if err != nil {
				return fmt.Errorf("failed to get decisions: %w", err)
			}

			if flags.asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(decisions)
			}

			if len(decisions) == 0 {
				fmt.Println("No log entries found.")
				return nil
			}

			fmt.Println("=== Recent Decisions ===")
			fmt.Println()
			for _, d := range decisions {
				req := d.Request
				if len(req) > 60 {
					req = req[:57] + "..."
				}
				fmt.Printf("%s | %-12s | %-15s | %-10s | %s\n",
					d.Timestamp.Format("2006-01-02 15:04"),
					d.Decision,
					d.Session,
					d.PromptType,
					req,
				)
			}
			return nil
		},
	}
	showCmd.Flags().IntVarP(&limit, "limit", "n", 20, "Number of entries to show")
	showCmd.Flags().StringVarP(&session, "session", "s", "", "Filter by session name")

	// Subcommand: log search
	searchCmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search log entries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			auditDB, err := db.Open()
			if err != nil {
				return fmt.Errorf("failed to open audit database: %w", err)
			}
			defer auditDB.Close()

			decisions, err := auditDB.SearchDecisions(query, limit)
			if err != nil {
				return fmt.Errorf("failed to search decisions: %w", err)
			}

			if flags.asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(decisions)
			}

			if len(decisions) == 0 {
				fmt.Printf("No entries matching '%s' found.\n", query)
				return nil
			}

			fmt.Printf("=== Search Results for '%s' ===\n\n", query)
			for _, d := range decisions {
				req := d.Request
				if len(req) > 60 {
					req = req[:57] + "..."
				}
				fmt.Printf("%s | %-12s | %-15s | %-10s | %s\n",
					d.Timestamp.Format("2006-01-02 15:04"),
					d.Decision,
					d.Session,
					d.PromptType,
					req,
				)
			}
			return nil
		},
	}
	searchCmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum results")

	// Subcommand: log stats
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show audit statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			auditDB, err := db.Open()
			if err != nil {
				return fmt.Errorf("failed to open audit database: %w", err)
			}
			defer auditDB.Close()

			stats, err := auditDB.GetStats()
			if err != nil {
				return fmt.Errorf("failed to get stats: %w", err)
			}

			if flags.asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(stats)
			}

			fmt.Println("=== Audit Statistics ===")
			fmt.Println()
			fmt.Printf("Total decisions: %d\n", stats["total_decisions"])
			fmt.Println()

			if byDecision, ok := stats["by_decision"].(map[string]int); ok {
				fmt.Println("By decision type:")
				for decision, count := range byDecision {
					fmt.Printf("  %-20s %d\n", decision, count)
				}
				fmt.Println()
			}

			if byMode, ok := stats["by_mode"].(map[string]int); ok {
				fmt.Println("By mode:")
				for mode, count := range byMode {
					fmt.Printf("  %-20s %d\n", mode, count)
				}
				fmt.Println()
			}

			fmt.Printf("Task runs today: %d\n", stats["task_runs_today"])
			if seconds, ok := stats["task_time_today_seconds"].(int64); ok {
				hours := seconds / 3600
				mins := (seconds % 3600) / 60
				fmt.Printf("Task time today: %dh %dm\n", hours, mins)
			}

			return nil
		},
	}

	cmd.AddCommand(showCmd, searchCmd, statsCmd)

	return cmd
}
