package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alexandrmotologa/tracereplay/pkg/models"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Storage handles persistent database operations for TraceReplay.
type Storage struct {
	db *sql.DB
}

// New creates a SQLite database connection with schema initialization and WAL mode.
func New(dbPath string) (*Storage, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	s := &Storage{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Storage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS trajectories (
		id TEXT PRIMARY KEY,
		parent_id TEXT,
		forked_at_step INTEGER,
		name TEXT NOT NULL,
		agent_id TEXT NOT NULL,
		model TEXT NOT NULL,
		status TEXT NOT NULL,
		failure_reason TEXT,
		total_steps INTEGER NOT NULL DEFAULT 0,
		total_tokens INTEGER NOT NULL DEFAULT 0,
		total_cost_usd REAL NOT NULL DEFAULT 0.0,
		started_at DATETIME NOT NULL,
		completed_at DATETIME,
		metadata TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		FOREIGN KEY (parent_id) REFERENCES trajectories(id) ON DELETE SET NULL
	);

	CREATE INDEX IF NOT EXISTS idx_trajectories_status ON trajectories(status);
	CREATE INDEX IF NOT EXISTS idx_trajectories_agent ON trajectories(agent_id);
	CREATE INDEX IF NOT EXISTS idx_trajectories_parent ON trajectories(parent_id);

	CREATE TABLE IF NOT EXISTS steps (
		id TEXT PRIMARY KEY,
		trajectory_id TEXT NOT NULL,
		step_index INTEGER NOT NULL,
		step_type TEXT NOT NULL,
		prompt TEXT NOT NULL,
		response TEXT NOT NULL,
		state_hash TEXT NOT NULL,
		duration_ms INTEGER NOT NULL,
		input_tokens INTEGER NOT NULL DEFAULT 0,
		output_tokens INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		error_message TEXT,
		is_replay_mock BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (trajectory_id) REFERENCES trajectories(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_steps_trajectory ON steps(trajectory_id, step_index);

	CREATE TABLE IF NOT EXISTS tool_calls (
		id TEXT PRIMARY KEY,
		step_id TEXT NOT NULL,
		tool_name TEXT NOT NULL,
		arguments TEXT NOT NULL,
		output TEXT NOT NULL,
		is_error BOOLEAN NOT NULL DEFAULT 0,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		is_virtual BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (step_id) REFERENCES steps(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_tool_calls_step ON tool_calls(step_id);

	CREATE TABLE IF NOT EXISTS memory_snapshots (
		id TEXT PRIMARY KEY,
		step_id TEXT NOT NULL,
		variables TEXT NOT NULL,
		scratchpad TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (step_id) REFERENCES steps(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_memory_snapshots_step ON memory_snapshots(step_id);
	`
	_, err := s.db.Exec(schema)
	return err
}

// CreateTrajectory inserts a new trajectory.
func (s *Storage) CreateTrajectory(t *models.Trajectory) error {
	if t.ID == "" {
		t.ID = "trj_" + uuid.New().String()
	}
	now := time.Now().UTC()
	if t.StartedAt.IsZero() {
		t.StartedAt = now
	}
	t.CreatedAt = now
	t.UpdatedAt = now

	metaJSON, _ := json.Marshal(t.Metadata)

	query := `
	INSERT INTO trajectories (
		id, parent_id, forked_at_step, name, agent_id, model, status,
		failure_reason, total_steps, total_tokens, total_cost_usd,
		started_at, completed_at, metadata, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		t.ID, t.ParentID, t.ForkedAtStep, t.Name, t.AgentID, t.Model, string(t.Status),
		t.FailureReason, t.TotalSteps, t.TotalTokens, t.TotalCostUSD,
		t.StartedAt, t.CompletedAt, string(metaJSON), t.CreatedAt, t.UpdatedAt,
	)
	return err
}

// UpdateTrajectory updates an existing trajectory summary.
func (s *Storage) UpdateTrajectory(t *models.Trajectory) error {
	t.UpdatedAt = time.Now().UTC()
	metaJSON, _ := json.Marshal(t.Metadata)

	query := `
	UPDATE trajectories SET
		status = ?, failure_reason = ?, total_steps = ?, total_tokens = ?,
		total_cost_usd = ?, completed_at = ?, metadata = ?, updated_at = ?
	WHERE id = ?`

	_, err := s.db.Exec(query,
		string(t.Status), t.FailureReason, t.TotalSteps, t.TotalTokens,
		t.TotalCostUSD, t.CompletedAt, string(metaJSON), t.UpdatedAt, t.ID,
	)
	return err
}

// GetTrajectory returns a trajectory with all its steps, tool calls, and memory snapshots.
func (s *Storage) GetTrajectory(id string) (*models.Trajectory, error) {
	row := s.db.QueryRow(`
		SELECT id, parent_id, forked_at_step, name, agent_id, model, status,
		       failure_reason, total_steps, total_tokens, total_cost_usd,
		       started_at, completed_at, metadata, created_at, updated_at
		FROM trajectories WHERE id = ?`, id)

	var t models.Trajectory
	var statusStr string
	var metaStr sql.NullString

	err := row.Scan(
		&t.ID, &t.ParentID, &t.ForkedAtStep, &t.Name, &t.AgentID, &t.Model, &statusStr,
		&t.FailureReason, &t.TotalSteps, &t.TotalTokens, &t.TotalCostUSD,
		&t.StartedAt, &t.CompletedAt, &metaStr, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("trajectory not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	t.Status = models.TrajectoryStatus(statusStr)

	if metaStr.Valid && metaStr.String != "" {
		_ = json.Unmarshal([]byte(metaStr.String), &t.Metadata)
	}

	steps, err := s.GetStepsByTrajectory(id)
	if err != nil {
		return nil, err
	}
	t.Steps = steps

	return &t, nil
}

// ListTrajectories retrieves trajectories filtered by status, agent or limit.
func (s *Storage) ListTrajectories(params models.FilterParams) ([]models.Trajectory, error) {
	query := `
		SELECT id, parent_id, forked_at_step, name, agent_id, model, status,
		       failure_reason, total_steps, total_tokens, total_cost_usd,
		       started_at, completed_at, metadata, created_at, updated_at
		FROM trajectories
		WHERE 1=1`
	var args []any

	if params.Status != "" {
		query += " AND status = ?"
		args = append(args, params.Status)
	}
	if params.AgentID != "" {
		query += " AND agent_id = ?"
		args = append(args, params.AgentID)
	}

	query += " ORDER BY created_at DESC"

	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	if params.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", params.Offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Trajectory
	for rows.Next() {
		var t models.Trajectory
		var statusStr string
		var metaStr sql.NullString

		err := rows.Scan(
			&t.ID, &t.ParentID, &t.ForkedAtStep, &t.Name, &t.AgentID, &t.Model, &statusStr,
			&t.FailureReason, &t.TotalSteps, &t.TotalTokens, &t.TotalCostUSD,
			&t.StartedAt, &t.CompletedAt, &metaStr, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		t.Status = models.TrajectoryStatus(statusStr)
		if metaStr.Valid && metaStr.String != "" {
			_ = json.Unmarshal([]byte(metaStr.String), &t.Metadata)
		}
		result = append(result, t)
	}

	return result, rows.Err()
}

// DeleteTrajectory deletes a trajectory and associated records via cascade.
func (s *Storage) DeleteTrajectory(id string) error {
	_, err := s.db.Exec("DELETE FROM trajectories WHERE id = ?", id)
	return err
}

// CreateStep inserts an individual step.
func (s *Storage) CreateStep(step *models.Step) error {
	if step.ID == "" {
		step.ID = "stp_" + uuid.New().String()
	}
	if step.CreatedAt.IsZero() {
		step.CreatedAt = time.Now().UTC()
	}

	query := `
	INSERT INTO steps (
		id, trajectory_id, step_index, step_type, prompt, response,
		state_hash, duration_ms, input_tokens, output_tokens, status,
		error_message, is_replay_mock, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		step.ID, step.TrajectoryID, step.StepIndex, string(step.StepType),
		step.Prompt, step.Response, step.StateHash, step.DurationMS,
		step.InputTokens, step.OutputTokens, string(step.Status),
		step.ErrorMessage, step.IsReplayMock, step.CreatedAt,
	)
	if err != nil {
		return err
	}

	for i := range step.ToolCalls {
		step.ToolCalls[i].StepID = step.ID
		if err := s.CreateToolCall(&step.ToolCalls[i]); err != nil {
			return err
		}
	}

	if step.MemorySnapshot != nil {
		step.MemorySnapshot.StepID = step.ID
		if err := s.CreateMemorySnapshot(step.MemorySnapshot); err != nil {
			return err
		}
	}

	return nil
}

// GetStepsByTrajectory fetches all steps for a trajectory ordered by step_index.
func (s *Storage) GetStepsByTrajectory(trajectoryID string) ([]models.Step, error) {
	query := `
	SELECT id, trajectory_id, step_index, step_type, prompt, response,
	       state_hash, duration_ms, input_tokens, output_tokens, status,
	       error_message, is_replay_mock, created_at
	FROM steps
	WHERE trajectory_id = ?
	ORDER BY step_index ASC`

	rows, err := s.db.Query(query, trajectoryID)
	if err != nil {
		return nil, err
	}

	var steps []models.Step
	for rows.Next() {
		var step models.Step
		var typeStr, statusStr string

		err := rows.Scan(
			&step.ID, &step.TrajectoryID, &step.StepIndex, &typeStr, &step.Prompt,
			&step.Response, &step.StateHash, &step.DurationMS, &step.InputTokens,
			&step.OutputTokens, &statusStr, &step.ErrorMessage, &step.IsReplayMock,
			&step.CreatedAt,
		)
		if err != nil {
			rows.Close()
			return nil, err
		}
		step.StepType = models.StepType(typeStr)
		step.Status = models.StepStatus(statusStr)
		steps = append(steps, step)
	}
	rows.Close()

	// Hydrate tool calls and memory snapshots after closing the primary rows cursor
	for i := range steps {
		toolCalls, err := s.getToolCallsByStep(steps[i].ID)
		if err != nil {
			return nil, err
		}
		steps[i].ToolCalls = toolCalls

		mem, err := s.getMemoryByStep(steps[i].ID)
		if err == nil && mem != nil {
			steps[i].MemorySnapshot = mem
		}
	}

	return steps, nil
}

// CreateToolCall persists an individual tool invocation.
func (s *Storage) CreateToolCall(tc *models.ToolCall) error {
	if tc.ID == "" {
		tc.ID = "tc_" + uuid.New().String()
	}
	if tc.CreatedAt.IsZero() {
		tc.CreatedAt = time.Now().UTC()
	}

	query := `
	INSERT INTO tool_calls (
		id, step_id, tool_name, arguments, output, is_error,
		duration_ms, is_virtual, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		tc.ID, tc.StepID, tc.ToolName, string(tc.Arguments), tc.Output,
		tc.IsError, tc.DurationMS, tc.IsVirtual, tc.CreatedAt,
	)
	return err
}

func (s *Storage) getToolCallsByStep(stepID string) ([]models.ToolCall, error) {
	query := `
	SELECT id, step_id, tool_name, arguments, output, is_error, duration_ms, is_virtual, created_at
	FROM tool_calls
	WHERE step_id = ?
	ORDER BY created_at ASC`

	rows, err := s.db.Query(query, stepID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []models.ToolCall
	for rows.Next() {
		var tc models.ToolCall
		var argsStr string
		err := rows.Scan(
			&tc.ID, &tc.StepID, &tc.ToolName, &argsStr, &tc.Output,
			&tc.IsError, &tc.DurationMS, &tc.IsVirtual, &tc.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tc.Arguments = json.RawMessage(argsStr)
		calls = append(calls, tc)
	}
	return calls, rows.Err()
}

// CreateMemorySnapshot persists memory state for a step.
func (s *Storage) CreateMemorySnapshot(ms *models.MemorySnapshot) error {
	if ms.ID == "" {
		ms.ID = "mem_" + uuid.New().String()
	}
	if ms.CreatedAt.IsZero() {
		ms.CreatedAt = time.Now().UTC()
	}

	query := `
	INSERT INTO memory_snapshots (
		id, step_id, variables, scratchpad, created_at
	) VALUES (?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		ms.ID, ms.StepID, string(ms.Variables), ms.Scratchpad, ms.CreatedAt,
	)
	return err
}

func (s *Storage) getMemoryByStep(stepID string) (*models.MemorySnapshot, error) {
	query := `
	SELECT id, step_id, variables, scratchpad, created_at
	FROM memory_snapshots
	WHERE step_id = ?`

	var ms models.MemorySnapshot
	var varsStr string
	err := s.db.QueryRow(query, stepID).Scan(
		&ms.ID, &ms.StepID, &varsStr, &ms.Scratchpad, &ms.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ms.Variables = json.RawMessage(varsStr)
	return &ms, nil
}
