// Package rules provides auto-approval rule matching for permission prompts
package rules

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/buddyhadry/permission-guardian/internal/detector"
	"gopkg.in/yaml.v3"
)

// Rule defines a single auto-approval rule
type Rule struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Enabled     bool     `yaml:"enabled"`
	Match       Match    `yaml:"match"`
	Action      string   `yaml:"action"` // "approve" or "deny"
	Priority    int      `yaml:"priority,omitempty"`
}

// Match defines the conditions for a rule to apply
type Match struct {
	Sessions    []string `yaml:"sessions,omitempty"`     // Glob patterns for session names
	PromptTypes []string `yaml:"prompt_types,omitempty"` // bash, edit, write, fetch, mcp, task
	Commands    []string `yaml:"commands,omitempty"`     // Regex patterns for command/request
	CWDs        []string `yaml:"cwds,omitempty"`         // Glob patterns for working directory
	Agents      []string `yaml:"agents,omitempty"`       // claude, codex
}

// Config holds all auto-approval rules
type Config struct {
	Version int    `yaml:"version"`
	Rules   []Rule `yaml:"rules"`
}

// DefaultConfigPath returns the default path for rules config
func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "permission-guardian", "rules.yaml")
}

// LoadConfig loads rules from a YAML file
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &Config{Version: 1, Rules: []Rule{}}, nil
		}
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig saves rules to a YAML file
func SaveConfig(config *Config, path string) error {
	if path == "" {
		path = DefaultConfigPath()
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// CreateDefaultConfig creates a default config with example rules
func CreateDefaultConfig() *Config {
	return &Config{
		Version: 1,
		Rules: []Rule{
			{
				Name:        "approve-read-only",
				Description: "Auto-approve read-only bash commands",
				Enabled:     false,
				Match: Match{
					PromptTypes: []string{"bash"},
					Commands:    []string{`^(ls|cat|head|tail|grep|find|pwd|echo|which|type|file|wc|diff)\s`},
				},
				Action:   "approve",
				Priority: 10,
			},
			{
				Name:        "approve-git-status",
				Description: "Auto-approve git status and log commands",
				Enabled:     false,
				Match: Match{
					PromptTypes: []string{"bash"},
					Commands:    []string{`^git\s+(status|log|diff|branch|show|remote)`},
				},
				Action:   "approve",
				Priority: 10,
			},
			{
				Name:        "approve-test-commands",
				Description: "Auto-approve test and build commands",
				Enabled:     false,
				Match: Match{
					PromptTypes: []string{"bash"},
					Commands:    []string{`^(go\s+test|npm\s+test|pytest|cargo\s+test|make\s+test)`},
				},
				Action:   "approve",
				Priority: 10,
			},
			{
				Name:        "approve-specific-session",
				Description: "Auto-approve all prompts from a specific session",
				Enabled:     false,
				Match: Match{
					Sessions: []string{"my-trusted-session*"},
				},
				Action:   "approve",
				Priority: 5,
			},
		},
	}
}

// MatchResult represents the result of rule matching
type MatchResult struct {
	Matched bool
	Rule    *Rule
	Action  string // "approve", "deny", or "" (no match)
}

// Matcher handles rule matching against sessions
type Matcher struct {
	config          *Config
	compiledRules   []compiledRule
}

type compiledRule struct {
	rule            *Rule
	sessionPatterns []string
	commandRegexes  []*regexp.Regexp
	cwdPatterns     []string
}

// NewMatcher creates a new rule matcher
func NewMatcher(config *Config) (*Matcher, error) {
	m := &Matcher{config: config}

	for i := range config.Rules {
		rule := &config.Rules[i]
		if !rule.Enabled {
			continue
		}

		cr := compiledRule{rule: rule}
		cr.sessionPatterns = rule.Match.Sessions
		cr.cwdPatterns = rule.Match.CWDs

		// Compile command regexes
		for _, pattern := range rule.Match.Commands {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, err
			}
			cr.commandRegexes = append(cr.commandRegexes, re)
		}

		m.compiledRules = append(m.compiledRules, cr)
	}

	return m, nil
}

// Match checks if a session matches any enabled rule
func (m *Matcher) Match(session detector.WaitingSession) MatchResult {
	for _, cr := range m.compiledRules {
		if m.matchRule(cr, session) {
			return MatchResult{
				Matched: true,
				Rule:    cr.rule,
				Action:  cr.rule.Action,
			}
		}
	}

	return MatchResult{Matched: false}
}

func (m *Matcher) matchRule(cr compiledRule, session detector.WaitingSession) bool {
	rule := cr.rule

	// Check session name patterns
	if len(cr.sessionPatterns) > 0 {
		if !matchAnyGlob(session.Session.Name, cr.sessionPatterns) {
			return false
		}
	}

	// Check prompt types
	if len(rule.Match.PromptTypes) > 0 {
		if !containsString(rule.Match.PromptTypes, string(session.PromptType)) {
			return false
		}
	}

	// Check command/request patterns
	if len(cr.commandRegexes) > 0 {
		matched := false
		for _, re := range cr.commandRegexes {
			if re.MatchString(session.Request) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check CWD patterns
	if len(cr.cwdPatterns) > 0 {
		if !matchAnyGlob(session.CWD, cr.cwdPatterns) {
			return false
		}
	}

	// Check agent types
	if len(rule.Match.Agents) > 0 {
		if !containsString(rule.Match.Agents, string(session.Agent)) {
			return false
		}
	}

	return true
}

func matchAnyGlob(s string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, s); matched {
			return true
		}
		// Also try as prefix match with wildcard
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(s, prefix) {
				return true
			}
		}
	}
	return false
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if strings.EqualFold(item, s) {
			return true
		}
	}
	return false
}
