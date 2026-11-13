package models

import (
	"encoding/json"
	"time"
)

// TrajectoryStatus represents the execution state of an agent trajectory.
type TrajectoryStatus string

const (
	StatusRunning   TrajectoryStatus = "running"
	StatusCompleted TrajectoryStatus = "completed"
	StatusFailed    TrajectoryStatus = "failed"
	StatusAborted   TrajectoryStatus = "aborted"
)

// StepType defines the category of an agent step.
type StepType string

const (
	StepTypePlanning       StepType = "planning"
	StepTypeReasoning      StepType = "reasoning"
	StepTypeToolCall       StepType = "tool_call"
	StepTypeToolResult     StepType = "tool_result"
	StepTypeLLMGeneration  StepType = "llm_generation"
	StepTypeMemoryMutation StepType = "memory_mutation"
	StepTypeUserInput      StepType = "user_input"
)

// StepStatus defines the outcome of a single step.
type StepStatus string

const (
	StepStatusSuccess StepStatus = "success"
	StepStatusFailed  StepStatus = "failed"
	StepStatusSkipped StepStatus = "skipped"
)

// Trajectory represents an entire recorded agent session.
type Trajectory struct {
	ID            string            `json:"id"`
	ParentID      *string           `json:"parent_id,omitempty"`
	ForkedAtStep  *int              `json:"forked_at_step,omitempty"`
	Name          string            `json:"name"`
	AgentID       string            `json:"agent_id"`
	Model         string            `json:"model"`
	Status        TrajectoryStatus  `json:"status"`
	FailureReason *string           `json:"failure_reason,omitempty"`
	TotalSteps    int               `json:"total_steps"`
	TotalTokens   int               `json:"total_tokens"`
	TotalCostUSD  float64           `json:"total_cost_usd"`
	StartedAt     time.Time         `json:"started_at"`
	CompletedAt   *time.Time        `json:"completed_at,omitempty"`
	Metadata      map[string]any    `json:"metadata,omitempty"`
	Steps         []Step            `json:"steps,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// Step represents a discrete event or turn inside an agent trajectory.
type Step struct {
	ID             string          `json:"id"`
	TrajectoryID   string          `json:"trajectory_id"`
	StepIndex      int             `json:"step_index"`
	StepType       StepType        `json:"step_type"`
	Prompt         string          `json:"prompt"`
	Response       string          `json:"response"`
	StateHash      string          `json:"state_hash"`
	DurationMS     int64           `json:"duration_ms"`
	InputTokens    int             `json:"input_tokens"`
	OutputTokens   int             `json:"output_tokens"`
	Status         StepStatus      `json:"status"`
	ErrorMessage   *string         `json:"error_message,omitempty"`
	ToolCalls      []ToolCall      `json:"tool_calls,omitempty"`
	MemorySnapshot *MemorySnapshot `json:"memory_snapshot,omitempty"`
	IsReplayMock   bool            `json:"is_replay_mock"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ToolCall represents an external tool invocation and its recorded output.
type ToolCall struct {
	ID          string          `json:"id"`
	StepID      string          `json:"step_id"`
	ToolName    string          `json:"tool_name"`
	Arguments   json.RawMessage `json:"arguments"`
	Output      string          `json:"output"`
	IsError     bool            `json:"is_error"`
	DurationMS  int64           `json:"duration_ms"`
	IsVirtual   bool            `json:"is_virtual"`
	CreatedAt   time.Time       `json:"created_at"`
}

// MemorySnapshot captures the agent state variables and scratchpad at a specific step.
type MemorySnapshot struct {
	ID         string          `json:"id"`
	StepID     string          `json:"step_id"`
	Variables  json.RawMessage `json:"variables"`
	Scratchpad string          `json:"scratchpad"`
	CreatedAt  time.Time       `json:"created_at"`
}

// ForkRequest payload for rewinding and creating a child trajectory.
type ForkRequest struct {
	ForkAtStep int            `json:"fork_at_step"`
	Name       string         `json:"name"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// ReplayMutation defines edits made to a step during time-travel resumption.
type ReplayMutation struct {
	Prompt            *string         `json:"prompt,omitempty"`
	ToolCallOverride  *ToolOverride   `json:"tool_call_override,omitempty"`
	MemoryOverrides   json.RawMessage `json:"memory_overrides,omitempty"`
	ScratchpadOverride *string        `json:"scratchpad_override,omitempty"`
}

// ToolOverride defines mutated arguments for a tool call.
type ToolOverride struct {
	ToolName  string          `json:"tool_name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ReplayRequest payload for rewinding, mutating, and resuming trajectory execution.
type ReplayRequest struct {
	FromStep  int             `json:"from_step"`
	Name      string          `json:"name,omitempty"`
	Mutation  *ReplayMutation `json:"mutation,omitempty"`
}

// TrajectoryDiff captures the difference between a baseline and comparison trajectory.
type TrajectoryDiff struct {
	BaseTrajectoryID    string     `json:"base_trajectory_id"`
	CompareTrajectoryID string     `json:"compare_trajectory_id"`
	StepDiffs           []StepDiff `json:"step_diffs"`
	TokenDelta          int        `json:"token_delta"`
	CostDeltaUSD        float64    `json:"cost_delta_usd"`
	BaseStatus          TrajectoryStatus `json:"base_status"`
	CompareStatus       TrajectoryStatus `json:"compare_status"`
}

// StepDiff contains step-level comparison data.
type StepDiff struct {
	StepIndex        int     `json:"step_index"`
	PromptEqual      bool    `json:"prompt_equal"`
	PromptBase       string  `json:"prompt_base"`
	PromptCompare    string  `json:"prompt_compare"`
	ResponseEqual    bool    `json:"response_equal"`
	ResponseBase     string  `json:"response_base"`
	ResponseCompare  string  `json:"response_compare"`
	ToolCallsEqual   bool    `json:"tool_calls_equal"`
	BaseToolCalls    []ToolCall `json:"base_tool_calls,omitempty"`
	CompareToolCalls []ToolCall `json:"compare_tool_calls,omitempty"`
	MemoryEqual      bool    `json:"memory_equal"`
	DurationDeltaMS  int64   `json:"duration_delta_ms"`
}

// FilterParams allows filtering trajectories in queries.
type FilterParams struct {
	Status  string
	AgentID string
	Limit   int
	Offset  int
}
