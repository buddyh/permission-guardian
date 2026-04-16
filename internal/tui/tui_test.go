package tui

import (
	"strings"
	"testing"
)

func TestAutoModeDescription(t *testing.T) {
	tests := []struct {
		mode        AutoMode
		wantString  string
		wantDetails string
	}{
		{mode: AutoSafe, wantString: "SAFE", wantDetails: "approve except destructive ops"},
		{mode: AutoNoDelete, wantString: "NO-DELETE", wantDetails: "approve except delete ops"},
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
