package storage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alexandrmotologa/tracereplay/pkg/models"
)

func TestStorageLifecycle(t *testing.T) {
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to init in-memory storage: %v", err)
	}
	defer store.Close()

	// 1. Create Trajectory
	traj := &models.Trajectory{
		ID:        "trj_test_01",
		Name:      "test-agent-run",
		AgentID:   "refactor-agent",
		Model:     "claude-3-7-sonnet",
		Status:    models.StatusRunning,
		StartedAt: time.Now().UTC(),
		Metadata: map[string]any{
			"env": "staging",
		},
	}
	if err := store.CreateTrajectory(traj); err != nil {
		t.Fatalf("CreateTrajectory failed: %v", err)
	}

	// 2. Add Step with ToolCall and MemorySnapshot
	step := &models.Step{
		ID:           "stp_test_01",
		TrajectoryID: traj.ID,
		StepIndex:    0,
		StepType:     models.StepTypeToolCall,
		Prompt:       "Inspect repository file structure",
		Response:     "Invoking list_files tool",
		StateHash:    "dummy_hash_01",
		DurationMS:   142,
		InputTokens:  240,
		OutputTokens: 60,
		Status:       models.StepStatusSuccess,
		ToolCalls: []models.ToolCall{
			{
				ToolName:   "list_files",
				Arguments:  json.RawMessage(`{"path":"./src"}`),
				Output:     `["main.go","util.go"]`,
				DurationMS: 25,
			},
		},
		MemorySnapshot: &models.MemorySnapshot{
			Variables:  json.RawMessage(`{"current_file":"main.go"}`),
			Scratchpad: "Investigating entrypoint",
		},
	}
	if err := store.CreateStep(step); err != nil {
		t.Fatalf("CreateStep failed: %v", err)
	}

	// 3. Retrieve Trajectory with Steps
	loaded, err := store.GetTrajectory(traj.ID)
	if err != nil {
		t.Fatalf("GetTrajectory failed: %v", err)
	}

	if loaded.Name != traj.Name {
		t.Errorf("expected name %s, got %s", traj.Name, loaded.Name)
	}
	if len(loaded.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(loaded.Steps))
	}
	if len(loaded.Steps[0].ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(loaded.Steps[0].ToolCalls))
	}
	if loaded.Steps[0].MemorySnapshot == nil {
		t.Fatal("expected memory snapshot to be populated")
	}

	// 4. Update Trajectory status
	loaded.Status = models.StatusCompleted
	now := time.Now().UTC()
	loaded.CompletedAt = &now
	loaded.TotalSteps = 1
	loaded.TotalTokens = 300
	if err := store.UpdateTrajectory(loaded); err != nil {
		t.Fatalf("UpdateTrajectory failed: %v", err)
	}

	// 5. List Trajectories
	list, err := store.ListTrajectories(models.FilterParams{Status: "completed"})
	if err != nil {
		t.Fatalf("ListTrajectories failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 completed trajectory, got %d", len(list))
	}
}
