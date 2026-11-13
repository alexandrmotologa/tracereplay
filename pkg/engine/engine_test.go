package engine

import (
	"encoding/json"
	"testing"

	"github.com/alexandrmotologa/tracereplay/pkg/models"
	"github.com/alexandrmotologa/tracereplay/pkg/storage"
)

type mockBroadcaster struct {
	events []string
}

func (m *mockBroadcaster) Broadcast(eventType string, payload any) {
	m.events = append(m.events, eventType)
}

func TestEngineRecordingAndForking(t *testing.T) {
	store, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}
	defer store.Close()

	broadcaster := &mockBroadcaster{}
	eng := New(store, broadcaster)

	// 1. Start Recording
	traj, err := eng.StartRecording("demo-agent-session", "agent-refactor", "gpt-4o", map[string]any{"scenario": "multi-step"})
	if err != nil {
		t.Fatalf("StartRecording failed: %v", err)
	}

	// 2. Record 3 steps
	for i := 0; i < 3; i++ {
		step := &models.Step{
			TrajectoryID: traj.ID,
			StepIndex:    i,
			StepType:     models.StepTypeToolCall,
			Prompt:       "Execute step action",
			Response:     "Step completed",
			DurationMS:   100,
			InputTokens:  150,
			OutputTokens: 50,
			Status:       models.StepStatusSuccess,
			ToolCalls: []models.ToolCall{
				{
					ToolName:   "db_query",
					Arguments:  json.RawMessage(`{"query":"SELECT * FROM users"}`),
					Output:     `[{"id":1}]`,
					DurationMS: 20,
				},
			},
			MemorySnapshot: &models.MemorySnapshot{
				Variables:  json.RawMessage(`{"counter":1}`),
				Scratchpad: "step scratchpad",
			},
		}
		if err := eng.RecordStep(step); err != nil {
			t.Fatalf("RecordStep %d failed: %v", i, err)
		}
	}

	// Step 3 (index 3) fails
	failStep := &models.Step{
		TrajectoryID: traj.ID,
		StepIndex:    3,
		StepType:     models.StepTypeToolCall,
		Prompt:       "Run external schema migration",
		Response:     "Execution error encountered",
		DurationMS:   200,
		InputTokens:  120,
		OutputTokens: 30,
		Status:       models.StepStatusFailed,
		ToolCalls: []models.ToolCall{
			{
				ToolName:  "run_migration",
				Arguments: json.RawMessage(`{"version":999}`),
				Output:    "Migration 999 does not exist",
				IsError:   true,
			},
		},
	}
	if err := eng.RecordStep(failStep); err != nil {
		t.Fatalf("RecordStep failed step failed: %v", err)
	}

	failReason := "Migration script not found"
	completedTraj, err := eng.CompleteTrajectory(traj.ID, models.StatusFailed, &failReason)
	if err != nil {
		t.Fatalf("CompleteTrajectory failed: %v", err)
	}

	if completedTraj.Status != models.StatusFailed {
		t.Errorf("expected status failed, got %s", completedTraj.Status)
	}
	if completedTraj.TotalSteps != 4 {
		t.Errorf("expected 4 steps, got %d", completedTraj.TotalSteps)
	}

	// 3. Time-Travel: Rewind to step 2 and mutate migration version to 14
	newPrompt := "Run migration with fixed version"
	replayReq := models.ReplayRequest{
		FromStep: 2,
		Name:     "repaired-run-v14",
		Mutation: &models.ReplayMutation{
			Prompt: &newPrompt,
			ToolCallOverride: &models.ToolOverride{
				ToolName:  "run_migration",
				Arguments: json.RawMessage(`{"version":14}`),
			},
		},
	}

	forkedTraj, mutatedStep, err := eng.ReplayWithMutation(traj.ID, replayReq)
	if err != nil {
		t.Fatalf("ReplayWithMutation failed: %v", err)
	}

	if *forkedTraj.ParentID != traj.ID {
		t.Errorf("expected parent ID %s, got %s", traj.ID, *forkedTraj.ParentID)
	}
	if *forkedTraj.ForkedAtStep != 2 {
		t.Errorf("expected forked at step 2, got %d", *forkedTraj.ForkedAtStep)
	}

	// Steps before step 2 should be replay mocks
	for i := 0; i < 2; i++ {
		if !forkedTraj.Steps[i].IsReplayMock {
			t.Errorf("step %d should be replay mock", i)
		}
		if len(forkedTraj.Steps[i].ToolCalls) > 0 && !forkedTraj.Steps[i].ToolCalls[0].IsVirtual {
			t.Errorf("step %d tool calls should be virtual", i)
		}
	}

	// Mutated step check
	if mutatedStep.Prompt != newPrompt {
		t.Errorf("expected prompt %s, got %s", newPrompt, mutatedStep.Prompt)
	}
	if len(mutatedStep.ToolCalls) != 1 || mutatedStep.ToolCalls[0].ToolName != "run_migration" {
		t.Fatalf("expected mutated tool call run_migration")
	}

	// 4. Test Diff
	loadedOrig, _ := store.GetTrajectory(traj.ID)
	diff := DiffTrajectories(loadedOrig, forkedTraj)
	if diff.BaseTrajectoryID != traj.ID || diff.CompareTrajectoryID != forkedTraj.ID {
		t.Errorf("diff IDs mismatch")
	}
	if len(diff.StepDiffs) == 0 {
		t.Fatalf("expected diff steps")
	}
}
