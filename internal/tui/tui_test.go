package tui

import (
	"strings"
	"testing"
)

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
