package detector

import (
	"testing"
)

func TestHasPermissionPrompt(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "yes option detected",
			content:  "Some content\n\u276F 1. Yes\nMore content",
			expected: true,
		},
		{
			name:     "do you want to proceed",
			content:  "Running command\nDo you want to proceed?\n> Yes",
			expected: true,
		},
		{
			name:     "do you want to allow",
			content:  "Do you want to allow this action?",
			expected: true,
		},
		{
			name:     "do you want to create",
			content:  "Do you want to create this file?",
			expected: true,
		},
		{
			name:     "yes and dont ask again",
			content:  "Yes, and don't ask again for this session",
			expected: true,
		},
		{
			name:     "no tell claude option",
			content:  "No, and tell Claude what went wrong",
			expected: true,
		},
		{
			name:     "no prompt",
			content:  "Just some regular output\nwith multiple lines",
			expected: false,
		},
		{
			name:     "empty content",
			content:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPermissionPrompt(tt.content)
			if result != tt.expected {
				t.Errorf("HasPermissionPrompt() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDetectPromptType(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected PromptType
	}{
		{
			name:     "bash command",
			content:  "Bash command: ls -la\nDo you want to proceed?",
			expected: PromptBash,
		},
		{
			name:     "bash paren style",
			content:  "Bash(ls -la)\nDo you want to proceed?",
			expected: PromptBash,
		},
		{
			name:     "fetch url",
			content:  "Claude wants to fetch https://example.com",
			expected: PromptFetch,
		},
		{
			name:     "fetch paren style",
			content:  "Fetch(https://example.com)",
			expected: PromptFetch,
		},
		{
			name:     "edit file",
			content:  "Edit(main.go)\nChanging line 10",
			expected: PromptEdit,
		},
		{
			name:     "make this edit to",
			content:  "Claude wants to make this edit to main.go",
			expected: PromptEdit,
		},
		{
			name:     "write file",
			content:  "Write(new_file.go)",
			expected: PromptWrite,
		},
		{
			name:     "create file",
			content:  "Do you want to create new_file.go?",
			expected: PromptWrite,
		},
		{
			name:     "mcp tool",
			content:  "MCP tool: some_tool",
			expected: PromptMCP,
		},
		{
			name:     "task agent",
			content:  "Task(subagent exploration)",
			expected: PromptTask,
		},
		{
			name:     "unknown",
			content:  "Some other prompt type",
			expected: PromptUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectPromptType(tt.content)
			if result != tt.expected {
				t.Errorf("DetectPromptType() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestExtractCWD(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple path",
			content:  "cwd: /Users/test/project",
			expected: "/Users/test/project",
		},
		{
			name:     "path with nbsp separator",
			content:  "cwd:\u00A0/Users/test/project\u00A0Session: 5m",
			expected: "/Users/test/project",
		},
		{
			name:     "no cwd",
			content:  "some content without cwd",
			expected: "unknown",
		},
		{
			name:     "complex path",
			content:  "Model: Opus cwd: /home/user/my-project other",
			expected: "/home/user/my-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractCWD(tt.content)
			if result != tt.expected {
				t.Errorf("ExtractCWD() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestExtractSessionInfo(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		expectModel     string
		expectContext   string
		expectBranch    string
		expectChanges   string
		expectWorking   bool
		expectWorkingSt string
	}{
		{
			name:          "full status line with context",
			content:       "Model:\u00A0Opus Ctx:\u00A038.9k (+128,-26) Session:\u00A010m",
			expectModel:   "Opus",
			expectContext: "38.9k",
			expectChanges: "+128,-26",
		},
		{
			name:            "working status",
			content:         "\u273D Manifesting...\n(ctrl+c to interrupt)",
			expectWorking:   true,
			expectWorkingSt: "Manifesting...",
		},
		{
			name:            "compacting status",
			content:         "\u00B7 Compacting conversation...\nPlease wait",
			expectWorking:   true,
			expectWorkingSt: "Compacting conversation...",
		},
		{
			name:          "sonnet model",
			content:       "Model: Sonnet Ctx: 12k",
			expectModel:   "Sonnet",
			expectContext: "12k",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ExtractSessionInfo(tt.content)

			if tt.expectModel != "" && info.Model != tt.expectModel {
				t.Errorf("Model = %q, expected %q", info.Model, tt.expectModel)
			}
			if tt.expectContext != "" && info.ContextSize != tt.expectContext {
				t.Errorf("ContextSize = %q, expected %q", info.ContextSize, tt.expectContext)
			}
			if tt.expectBranch != "" && info.GitBranch != tt.expectBranch {
				t.Errorf("GitBranch = %q, expected %q", info.GitBranch, tt.expectBranch)
			}
			if tt.expectChanges != "" && info.GitChanges != tt.expectChanges {
				t.Errorf("GitChanges = %q, expected %q", info.GitChanges, tt.expectChanges)
			}
			if info.IsWorking != tt.expectWorking {
				t.Errorf("IsWorking = %v, expected %v", info.IsWorking, tt.expectWorking)
			}
			if tt.expectWorkingSt != "" && info.WorkingStatus != tt.expectWorkingSt {
				t.Errorf("WorkingStatus = %q, expected %q", info.WorkingStatus, tt.expectWorkingSt)
			}
		})
	}
}

func TestExtractRequest(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		promptType   PromptType
		wantNonEmpty bool
	}{
		{
			name: "bash request",
			content: `Some context
Bash command: ls -la /tmp
Description: List files in tmp
Do you want to proceed?`,
			promptType:   PromptBash,
			wantNonEmpty: true,
		},
		{
			name: "write request",
			content: `Creating new file
Do you want to create /path/to/new_file.go?
Yes / No`,
			promptType:   PromptWrite,
			wantNonEmpty: true,
		},
		{
			name: "edit request",
			content: `Claude wants to make this edit to main.go
- old line
+ new line`,
			promptType:   PromptEdit,
			wantNonEmpty: true,
		},
		{
			name: "fallback request",
			content: `Some context before
More context
Even more
Do you want to proceed?`,
			promptType:   PromptUnknown,
			wantNonEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractRequest(tt.content, tt.promptType)
			if tt.wantNonEmpty && result == "" {
				t.Errorf("ExtractRequest() returned empty, expected non-empty")
			}
		})
	}
}

func TestMustAtoi(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"123", 123},
		{"0", 0},
		{"42", 42},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mustAtoi(tt.input)
			if result != tt.expected {
				t.Errorf("mustAtoi(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}
