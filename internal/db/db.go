// Package db provides SQLite-based audit logging for permission-guardian
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
}

// Decision represents a logged permission decision
type Decision struct {
	ID         int64
	Timestamp  time.Time
	Session    string
	Decision   string // approved, denied, auto_approved, auto_skipped
	Mode       string // manual, SAFE, ALL, BURST
	PromptType string // bash, file, mcp, etc.
	Request    string
	ProjectDir string
	GitBranch  string
	TaskRunID  string // Groups decisions in same work session
}

// TaskRun represents a completed task run
type TaskRun struct {
	ID        int64
	StartTime time.Time
	EndTime   time.Time
	Session   string
	Duration  time.Duration
	Approvals int
}

// dbPath returns the path to the SQLite database
func dbPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "permission-guardian", "audit.db")
}

// Open opens or creates the audit database
func Open() (*DB, error) {
	path := dbPath()
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// migrate creates tables if they don't exist
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS decisions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		session TEXT NOT NULL,
		decision TEXT NOT NULL,
		mode TEXT,
		prompt_type TEXT,
		request TEXT,
		project_dir TEXT,
		git_branch TEXT,
		task_run_id TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_decisions_timestamp ON decisions(timestamp);
	CREATE INDEX IF NOT EXISTS idx_decisions_session ON decisions(session);
	CREATE INDEX IF NOT EXISTS idx_decisions_task_run ON decisions(task_run_id);

	CREATE TABLE IF NOT EXISTS task_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		start_time DATETIME NOT NULL,
		end_time DATETIME NOT NULL,
		session TEXT NOT NULL,
		duration_seconds INTEGER NOT NULL,
		approvals INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_task_runs_session ON task_runs(session);
	CREATE INDEX IF NOT EXISTS idx_task_runs_start ON task_runs(start_time);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// LogDecision logs a permission decision
func (db *DB) LogDecision(d Decision) error {
	_, err := db.conn.Exec(`
		INSERT INTO decisions (timestamp, session, decision, mode, prompt_type, request, project_dir, git_branch, task_run_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, d.Timestamp, d.Session, d.Decision, d.Mode, d.PromptType, d.Request, d.ProjectDir, d.GitBranch, d.TaskRunID)
	return err
}

// LogTaskRun logs a completed task run
func (db *DB) LogTaskRun(t TaskRun) error {
	_, err := db.conn.Exec(`
		INSERT INTO task_runs (start_time, end_time, session, duration_seconds, approvals)
		VALUES (?, ?, ?, ?, ?)
	`, t.StartTime, t.EndTime, t.Session, int64(t.Duration.Seconds()), t.Approvals)
	return err
}

// GetRecentDecisions returns the most recent decisions
func (db *DB) GetRecentDecisions(limit int) ([]Decision, error) {
	rows, err := db.conn.Query(`
		SELECT id, timestamp, session, decision, mode, prompt_type, request, project_dir, git_branch, task_run_id
		FROM decisions
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var d Decision
		var ts string
		var mode, promptType, request, projectDir, gitBranch, taskRunID sql.NullString
		if err := rows.Scan(&d.ID, &ts, &d.Session, &d.Decision, &mode, &promptType, &request, &projectDir, &gitBranch, &taskRunID); err != nil {
			return nil, err
		}
		d.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		d.Mode = mode.String
		d.PromptType = promptType.String
		d.Request = request.String
		d.ProjectDir = projectDir.String
		d.GitBranch = gitBranch.String
		d.TaskRunID = taskRunID.String
		decisions = append(decisions, d)
	}
	return decisions, nil
}

// SearchDecisions searches decisions by session name or request content
func (db *DB) SearchDecisions(query string, limit int) ([]Decision, error) {
	rows, err := db.conn.Query(`
		SELECT id, timestamp, session, decision, mode, prompt_type, request, project_dir, git_branch, task_run_id
		FROM decisions
		WHERE session LIKE ? OR request LIKE ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, "%"+query+"%", "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var d Decision
		var ts string
		var mode, promptType, request, projectDir, gitBranch, taskRunID sql.NullString
		if err := rows.Scan(&d.ID, &ts, &d.Session, &d.Decision, &mode, &promptType, &request, &projectDir, &gitBranch, &taskRunID); err != nil {
			return nil, err
		}
		d.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		d.Mode = mode.String
		d.PromptType = promptType.String
		d.Request = request.String
		d.ProjectDir = projectDir.String
		d.GitBranch = gitBranch.String
		d.TaskRunID = taskRunID.String
		decisions = append(decisions, d)
	}
	return decisions, nil
}

// GetDecisionsBySession returns decisions for a specific session
func (db *DB) GetDecisionsBySession(session string, limit int) ([]Decision, error) {
	rows, err := db.conn.Query(`
		SELECT id, timestamp, session, decision, mode, prompt_type, request, project_dir, git_branch, task_run_id
		FROM decisions
		WHERE session = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, session, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var d Decision
		var ts string
		var mode, promptType, request, projectDir, gitBranch, taskRunID sql.NullString
		if err := rows.Scan(&d.ID, &ts, &d.Session, &d.Decision, &mode, &promptType, &request, &projectDir, &gitBranch, &taskRunID); err != nil {
			return nil, err
		}
		d.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		d.Mode = mode.String
		d.PromptType = promptType.String
		d.Request = request.String
		d.ProjectDir = projectDir.String
		d.GitBranch = gitBranch.String
		d.TaskRunID = taskRunID.String
		decisions = append(decisions, d)
	}
	return decisions, nil
}

// GetStats returns aggregate statistics
func (db *DB) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total decisions
	var total int
	if err := db.conn.QueryRow("SELECT COUNT(*) FROM decisions").Scan(&total); err != nil {
		return nil, err
	}
	stats["total_decisions"] = total

	// Decisions by type
	rows, err := db.conn.Query(`
		SELECT decision, COUNT(*) as count
		FROM decisions
		GROUP BY decision
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byType := make(map[string]int)
	for rows.Next() {
		var decision string
		var count int
		if err := rows.Scan(&decision, &count); err != nil {
			return nil, err
		}
		byType[decision] = count
	}
	stats["by_decision"] = byType

	// Decisions by mode
	rows2, err := db.conn.Query(`
		SELECT COALESCE(mode, 'manual'), COUNT(*) as count
		FROM decisions
		GROUP BY mode
	`)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	byMode := make(map[string]int)
	for rows2.Next() {
		var mode string
		var count int
		if err := rows2.Scan(&mode, &count); err != nil {
			return nil, err
		}
		byMode[mode] = count
	}
	stats["by_mode"] = byMode

	// Task runs today
	var todayRuns int
	if err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM task_runs
		WHERE date(start_time) = date('now')
	`).Scan(&todayRuns); err != nil {
		return nil, err
	}
	stats["task_runs_today"] = todayRuns

	// Total task time today (seconds)
	var todayTime int64
	if err := db.conn.QueryRow(`
		SELECT COALESCE(SUM(duration_seconds), 0) FROM task_runs
		WHERE date(start_time) = date('now')
	`).Scan(&todayTime); err != nil {
		return nil, err
	}
	stats["task_time_today_seconds"] = todayTime

	return stats, nil
}
