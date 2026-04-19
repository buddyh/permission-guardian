package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// testDB creates a temporary in-memory-like database for testing
func testDB(t *testing.T) *DB {
	t.Helper()

	// Create a temp file for SQLite (in-memory doesn't persist between connections)
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test-audit.db")

	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		_ = conn.Close()
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
		_ = os.Remove(path)
	})

	return db
}

func TestMigrate(t *testing.T) {
	db := testDB(t)

	// Verify tables exist
	var tableName string
	err := db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='decisions'").Scan(&tableName)
	if err != nil {
		t.Errorf("decisions table not found: %v", err)
	}

	err = db.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='task_runs'").Scan(&tableName)
	if err != nil {
		t.Errorf("task_runs table not found: %v", err)
	}
}

func TestLogDecision(t *testing.T) {
	db := testDB(t)

	decision := Decision{
		Timestamp:  time.Now(),
		Session:    "test-session",
		Decision:   "approved",
		Mode:       "SAFE",
		PromptType: "bash",
		Request:    "ls -la",
		ProjectDir: "/home/user/project",
		GitBranch:  "main",
		TaskRunID:  "task-123",
	}

	err := db.LogDecision(decision)
	if err != nil {
		t.Fatalf("LogDecision() error = %v", err)
	}

	// Verify it was inserted
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM decisions").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count decisions: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 decision, got %d", count)
	}
}

func TestLogTaskRun(t *testing.T) {
	db := testDB(t)

	taskRun := TaskRun{
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
		Session:   "test-session",
		Duration:  1 * time.Hour,
		Approvals: 10,
	}

	err := db.LogTaskRun(taskRun)
	if err != nil {
		t.Fatalf("LogTaskRun() error = %v", err)
	}

	// Verify it was inserted
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM task_runs").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count task_runs: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 task run, got %d", count)
	}
}

func TestGetRecentDecisions(t *testing.T) {
	db := testDB(t)

	// Insert multiple decisions
	for i := 0; i < 5; i++ {
		decision := Decision{
			Timestamp:  time.Now().Add(time.Duration(-i) * time.Minute),
			Session:    "test-session",
			Decision:   "approved",
			Mode:       "manual",
			PromptType: "bash",
			Request:    "command " + string(rune('A'+i)),
		}
		if err := db.LogDecision(decision); err != nil {
			t.Fatalf("LogDecision() error = %v", err)
		}
	}

	// Get recent decisions
	decisions, err := db.GetRecentDecisions(3)
	if err != nil {
		t.Fatalf("GetRecentDecisions() error = %v", err)
	}

	if len(decisions) != 3 {
		t.Errorf("Expected 3 decisions, got %d", len(decisions))
	}

	// Should be ordered by timestamp DESC (most recent first)
	for i := 0; i < len(decisions)-1; i++ {
		if decisions[i].Timestamp.Before(decisions[i+1].Timestamp) {
			t.Error("Decisions not ordered by timestamp DESC")
		}
	}
}

func TestSearchDecisions(t *testing.T) {
	db := testDB(t)

	// Insert decisions with different sessions and requests
	decisions := []Decision{
		{Timestamp: time.Now(), Session: "claude-main", Decision: "approved", Request: "git status"},
		{Timestamp: time.Now(), Session: "codex-test", Decision: "approved", Request: "npm test"},
		{Timestamp: time.Now(), Session: "claude-dev", Decision: "denied", Request: "rm -rf /"},
	}

	for _, d := range decisions {
		if err := db.LogDecision(d); err != nil {
			t.Fatalf("LogDecision() error = %v", err)
		}
	}

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{"search by session prefix", "claude", 2},
		{"search by request", "git", 1},
		{"search by exact session", "codex-test", 1},
		{"no results", "nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := db.SearchDecisions(tt.query, 10)
			if err != nil {
				t.Fatalf("SearchDecisions() error = %v", err)
			}
			if len(results) != tt.expected {
				t.Errorf("Expected %d results, got %d", tt.expected, len(results))
			}
		})
	}
}

func TestGetDecisionsBySession(t *testing.T) {
	db := testDB(t)

	// Insert decisions for different sessions
	for i := 0; i < 3; i++ {
		if err := db.LogDecision(Decision{
			Timestamp: time.Now(),
			Session:   "session-A",
			Decision:  "approved",
			Request:   "command A",
		}); err != nil {
			t.Fatalf("LogDecision() error = %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := db.LogDecision(Decision{
			Timestamp: time.Now(),
			Session:   "session-B",
			Decision:  "approved",
			Request:   "command B",
		}); err != nil {
			t.Fatalf("LogDecision() error = %v", err)
		}
	}

	// Get decisions for session-A
	decisions, err := db.GetDecisionsBySession("session-A", 10)
	if err != nil {
		t.Fatalf("GetDecisionsBySession() error = %v", err)
	}
	if len(decisions) != 3 {
		t.Errorf("Expected 3 decisions for session-A, got %d", len(decisions))
	}

	// Verify all belong to session-A
	for _, d := range decisions {
		if d.Session != "session-A" {
			t.Errorf("Expected session-A, got %s", d.Session)
		}
	}
}

func TestGetStats(t *testing.T) {
	db := testDB(t)

	// Insert some decisions
	decisions := []Decision{
		{Timestamp: time.Now(), Session: "s1", Decision: "approved", Mode: "SAFE"},
		{Timestamp: time.Now(), Session: "s1", Decision: "approved", Mode: "SAFE"},
		{Timestamp: time.Now(), Session: "s2", Decision: "denied", Mode: "manual"},
		{Timestamp: time.Now(), Session: "s2", Decision: "auto_approved", Mode: "ALL"},
	}

	for _, d := range decisions {
		if err := db.LogDecision(d); err != nil {
			t.Fatalf("LogDecision() error = %v", err)
		}
	}

	// Insert a task run
	if err := db.LogTaskRun(TaskRun{
		StartTime: time.Now().Add(-30 * time.Minute),
		EndTime:   time.Now(),
		Session:   "s1",
		Duration:  30 * time.Minute,
		Approvals: 5,
	}); err != nil {
		t.Fatalf("LogTaskRun() error = %v", err)
	}

	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	// Check total decisions
	if total, ok := stats["total_decisions"].(int); !ok || total != 4 {
		t.Errorf("total_decisions = %v, want 4", stats["total_decisions"])
	}

	// Check by_decision breakdown
	byDecision, ok := stats["by_decision"].(map[string]int)
	if !ok {
		t.Fatal("by_decision not found or wrong type")
	}
	if byDecision["approved"] != 2 {
		t.Errorf("approved count = %d, want 2", byDecision["approved"])
	}
	if byDecision["denied"] != 1 {
		t.Errorf("denied count = %d, want 1", byDecision["denied"])
	}

	// Check by_mode breakdown
	byMode, ok := stats["by_mode"].(map[string]int)
	if !ok {
		t.Fatal("by_mode not found or wrong type")
	}
	if byMode["SAFE"] != 2 {
		t.Errorf("SAFE mode count = %d, want 2", byMode["SAFE"])
	}

	// Check task runs today
	if todayRuns, ok := stats["task_runs_today"].(int); !ok || todayRuns != 1 {
		t.Errorf("task_runs_today = %v, want 1", stats["task_runs_today"])
	}
}

func TestClose(t *testing.T) {
	db := testDB(t)

	err := db.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Subsequent operations should fail
	err = db.LogDecision(Decision{
		Timestamp: time.Now(),
		Session:   "test",
		Decision:  "approved",
	})
	if err == nil {
		t.Error("Expected error after closing database")
	}
}
