package tui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/buddyh/permission-guardian/internal/detector"
	"github.com/buddyh/permission-guardian/internal/tmux"
)

func TestAutoModeDescription(t *testing.T) {
	tests := []struct {
		mode        AutoMode
		wantString  string
		wantDetails string
	}{
		{mode: AutoSafe, wantString: "SAFE", wantDetails: "approve except destructive ops"},
		{mode: AutoNoDelete, wantString: "NODEL", wantDetails: "approve except delete ops"},
		{mode: AutoAll, wantString: "ALL", wantDetails: "approve every prompt"},
	}

	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.wantString {
			t.Fatalf("String() = %q, want %q", got, tt.wantString)
		}
		if got := tt.mode.Description(); got != tt.wantDetails {
			t.Fatalf("Description() = %q, want %q", got, tt.wantDetails)
		}
	}
}

func TestRenderContextBar(t *testing.T) {
	t.Run("codex percentage renders bar", func(t *testing.T) {
		got := renderContextBar("65%", 6)
		if strings.Count(got, "█") != 3 {
			t.Fatalf("expected percentage context to render filled bar, got %q", got)
		}
	})

	t.Run("claude absolute context renders value", func(t *testing.T) {
		got := renderContextBar("38.9k", 6)
		if !strings.Contains(got, "38.9k") {
			t.Fatalf("expected absolute context to render raw value, got %q", got)
		}
	})
}

func TestRenderHeaderFitsWidth(t *testing.T) {
	m := Model{
		sessions: []detector.WaitingSession{
			{
				Session:    tmux.Session{Name: "feature-work"},
				PromptType: detector.PromptBash,
			},
		},
		actionStatus: "AUTO SAFE enabled for feature-work with an intentionally long status payload",
	}

	for _, width := range []int{90, 108, 132} {
		t.Run("width_"+strconv.Itoa(width), func(t *testing.T) {
			header := m.renderHeader(width)
			for _, line := range strings.Split(header, "\n") {
				if got := lipgloss.Width(line); got > width {
					t.Fatalf("header line width = %d, want <= %d; line=%q", got, width, line)
				}
			}
		})
	}
}

func TestRenderHeaderPolicyLineOnlyWhenItFits(t *testing.T) {
	m := Model{}

	narrow := m.renderHeader(108)
	if strings.Contains(narrow, "SAFE = non-destructive") {
		t.Fatalf("policy line should be hidden when it does not fit: %q", narrow)
	}

	wide := m.renderHeader(140)
	if !strings.Contains(wide, "SAFE = non-destructive") {
		t.Fatalf("policy line should render at wide widths: %q", wide)
	}
}
