package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/buddyh/permission-guardian/internal/detector"
	"github.com/buddyh/permission-guardian/internal/tmux"
)

func TestMatchAnyGlob(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		patterns []string
		expected bool
	}{
		{
			name:     "exact match",
			s:        "my-session",
			patterns: []string{"my-session"},
			expected: true,
		},
		{
			name:     "wildcard suffix",
			s:        "my-session-123",
			patterns: []string{"my-session*"},
			expected: true,
		},
		{
			name:     "wildcard prefix",
			s:        "prefix-session",
			patterns: []string{"*-session"},
			expected: true,
		},
		{
			name:     "no match",
			s:        "other-session",
			patterns: []string{"my-session*"},
			expected: false,
		},
		{
			name:     "multiple patterns first match",
			s:        "test-session",
			patterns: []string{"test*", "other*"},
			expected: true,
		},
		{
			name:     "multiple patterns second match",
			s:        "other-session",
			patterns: []string{"test*", "other*"},
			expected: true,
		},
		{
			name:     "empty patterns",
			s:        "anything",
			patterns: []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchAnyGlob(tt.s, tt.patterns)
			if result != tt.expected {
				t.Errorf("matchAnyGlob(%q, %v) = %v, expected %v", tt.s, tt.patterns, result, tt.expected)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		s        string
		expected bool
	}{
		{
			name:     "found exact",
			slice:    []string{"bash", "edit", "write"},
			s:        "bash",
			expected: true,
		},
		{
			name:     "found case insensitive",
			slice:    []string{"BASH", "EDIT", "WRITE"},
			s:        "bash",
			expected: true,
		},
		{
			name:     "not found",
			slice:    []string{"bash", "edit", "write"},
			s:        "fetch",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			s:        "bash",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsString(tt.slice, tt.s)
			if result != tt.expected {
				t.Errorf("containsString(%v, %q) = %v, expected %v", tt.slice, tt.s, result, tt.expected)
			}
		})
	}
}

func TestNewMatcher(t *testing.T) {
	config := &Config{
		Version: 1,
		Rules: []Rule{
			{
				Name:    "test-rule",
				Enabled: true,
				Match: Match{
					Commands: []string{`^ls\s`},
				},
				Action: "approve",
			},
			{
				Name:    "disabled-rule",
				Enabled: false,
				Match: Match{
					Commands: []string{`^rm\s`},
				},
				Action: "deny",
			},
		},
	}

	matcher, err := NewMatcher(config)
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	// Should only have 1 compiled rule (the enabled one)
	if len(matcher.compiledRules) != 1 {
		t.Errorf("Expected 1 compiled rule, got %d", len(matcher.compiledRules))
	}
}

func TestNewMatcherInvalidRegex(t *testing.T) {
	config := &Config{
		Version: 1,
		Rules: []Rule{
			{
				Name:    "bad-regex",
				Enabled: true,
				Match: Match{
					Commands: []string{`[invalid`}, // Invalid regex
				},
				Action: "approve",
			},
		},
	}

	_, err := NewMatcher(config)
	if err == nil {
		t.Error("Expected error for invalid regex, got nil")
	}
}

func TestMatcherMatch(t *testing.T) {
	config := &Config{
		Version: 1,
		Rules: []Rule{
			{
				Name:    "approve-ls",
				Enabled: true,
				Match: Match{
					PromptTypes: []string{"bash"},
					Commands:    []string{`^ls\s`},
				},
				Action: "approve",
			},
			{
				Name:    "approve-git-status",
				Enabled: true,
				Match: Match{
					PromptTypes: []string{"bash"},
					Commands:    []string{`^git\s+status`},
				},
				Action: "approve",
			},
			{
				Name:    "deny-rm",
				Enabled: true,
				Match: Match{
					PromptTypes: []string{"bash"},
					Commands:    []string{`^rm\s`},
				},
				Action: "deny",
			},
		},
	}

	matcher, err := NewMatcher(config)
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		name       string
		session    detector.WaitingSession
		wantMatch  bool
		wantAction string
	}{
		{
			name: "matches ls command",
			session: detector.WaitingSession{
				Session:    tmux.Session{Name: "test"},
				PromptType: detector.PromptBash,
				Request:    "ls -la /tmp",
			},
			wantMatch:  true,
			wantAction: "approve",
		},
		{
			name: "matches git status",
			session: detector.WaitingSession{
				Session:    tmux.Session{Name: "test"},
				PromptType: detector.PromptBash,
				Request:    "git status",
			},
			wantMatch:  true,
			wantAction: "approve",
		},
		{
			name: "matches rm command denied",
			session: detector.WaitingSession{
				Session:    tmux.Session{Name: "test"},
				PromptType: detector.PromptBash,
				Request:    "rm -rf /tmp/test",
			},
			wantMatch:  true,
			wantAction: "deny",
		},
		{
			name: "no match for edit",
			session: detector.WaitingSession{
				Session:    tmux.Session{Name: "test"},
				PromptType: detector.PromptEdit,
				Request:    "ls -la /tmp",
			},
			wantMatch: false,
		},
		{
			name: "no match for unknown command",
			session: detector.WaitingSession{
				Session:    tmux.Session{Name: "test"},
				PromptType: detector.PromptBash,
				Request:    "echo hello",
			},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(tt.session)
			if result.Matched != tt.wantMatch {
				t.Errorf("Matched = %v, want %v", result.Matched, tt.wantMatch)
			}
			if tt.wantMatch && result.Action != tt.wantAction {
				t.Errorf("Action = %q, want %q", result.Action, tt.wantAction)
			}
		})
	}
}

func TestMatcherSessionPattern(t *testing.T) {
	config := &Config{
		Version: 1,
		Rules: []Rule{
			{
				Name:    "approve-trusted",
				Enabled: true,
				Match: Match{
					Sessions: []string{"trusted-*"},
				},
				Action: "approve",
			},
		},
	}

	matcher, err := NewMatcher(config)
	if err != nil {
		t.Fatalf("NewMatcher() error = %v", err)
	}

	tests := []struct {
		name      string
		session   string
		wantMatch bool
	}{
		{"matches trusted session", "trusted-dev", true},
		{"matches trusted session 2", "trusted-prod", true},
		{"no match untrusted", "untrusted-session", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws := detector.WaitingSession{
				Session: tmux.Session{Name: tt.session},
			}
			result := matcher.Match(ws)
			if result.Matched != tt.wantMatch {
				t.Errorf("Matched = %v, want %v", result.Matched, tt.wantMatch)
			}
		})
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	// Test loading from non-existent path
	config, err := LoadConfig("/non/existent/path.yaml")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config == nil {
		t.Fatal("Expected non-nil config")
	}
	if len(config.Rules) != 0 {
		t.Errorf("Expected empty rules, got %d", len(config.Rules))
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test-rules.yaml")

	original := &Config{
		Version: 1,
		Rules: []Rule{
			{
				Name:    "test-rule",
				Enabled: true,
				Match: Match{
					PromptTypes: []string{"bash"},
					Commands:    []string{`^ls\s`},
				},
				Action: "approve",
			},
		},
	}

	// Save
	if err := SaveConfig(original, path); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.Version != original.Version {
		t.Errorf("Version = %d, want %d", loaded.Version, original.Version)
	}
	if len(loaded.Rules) != len(original.Rules) {
		t.Errorf("Rules count = %d, want %d", len(loaded.Rules), len(original.Rules))
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	config := CreateDefaultConfig()

	if config == nil {
		t.Fatal("Expected non-nil config")
	}
	if config.Version != 1 {
		t.Errorf("Version = %d, want 1", config.Version)
	}
	if len(config.Rules) == 0 {
		t.Error("Expected some default rules")
	}

	// All default rules should be disabled
	for _, rule := range config.Rules {
		if rule.Enabled {
			t.Errorf("Default rule %q should be disabled", rule.Name)
		}
	}
}
