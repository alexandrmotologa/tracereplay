package agentdemo

import (
	"encoding/json"
	"testing"

	"github.com/alexandrmotologa/tracereplay/pkg/engine"
	"github.com/alexandrmotologa/tracereplay/pkg/models"
	"github.com/alexandrmotologa/tracereplay/pkg/storage"
)

func TestAutonomousAgent15StepScenario(t *testing.T) {
	store, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}
	defer store.Close()

	eng := engine.New(store, nil)

	// 1. Run baseline demo that fails at step 12
	failedTraj, err := Run15StepDemo(eng, DemoConfig{
		ShouldFailAtStep12: true,
	})
	if err != nil {
		t.Fatalf("Run15StepDemo failed: %v", err)
	}

	if failedTraj.Status != models.StatusFailed {
		t.Errorf("expected failed status, got %s", failedTraj.Status)
	}
	if failedTraj.TotalSteps != 13 {
		t.Errorf("expected 13 steps recorded up to failure at 12, got %d", failedTraj.TotalSteps)
	}

	// 2. Perform Time-Travel Rewind to Step 8
	newPrompt := "Synthesize patch parameters and enforce target version 12 in registry."
	replayReq := models.ReplayRequest{
		FromStep: 8,
		Name:     "repaired-migration-from-step8",
		Mutation: &models.ReplayMutation{
			Prompt: &newPrompt,
			MemoryOverrides: json.RawMessage(`{"target_version":12,"schema_migration_ready":true}`),
		},
	}

	forkedTraj, mutatedStep, err := eng.ReplayWithMutation(failedTraj.ID, replayReq)
	if err != nil {
		t.Fatalf("ReplayWithMutation failed: %v", err)
	}

	if *forkedTraj.ParentID != failedTraj.ID {
		t.Errorf("expected parent ID %s, got %s", failedTraj.ID, *forkedTraj.ParentID)
	}
	if *forkedTraj.ForkedAtStep != 8 {
		t.Errorf("expected forked at step 8, got %d", *forkedTraj.ForkedAtStep)
	}
	if mutatedStep.Prompt != newPrompt {
		t.Errorf("expected prompt %s, got %s", newPrompt, mutatedStep.Prompt)
	}

	// 3. Continue steps 9 to 14 in repaired mode
	for i := 9; i < 15; i++ {
		step, isFail := buildStep(forkedTraj.ID, i, DemoConfig{ShouldFailAtStep12: false, FixedVersion: 12})
		if isFail {
			t.Fatalf("step %d unexpectedly failed in repaired mode", i)
		}
		if err := eng.RecordStep(step); err != nil {
			t.Fatalf("failed to record step %d: %v", i, err)
		}
	}

	completedFork, err := eng.CompleteTrajectory(forkedTraj.ID, models.StatusCompleted, nil)
	if err != nil {
		t.Fatalf("failed to complete forked trajectory: %v", err)
	}

	if completedFork.Status != models.StatusCompleted {
		t.Errorf("expected completed status, got %s", completedFork.Status)
	}
	if completedFork.TotalSteps != 15 {
		t.Errorf("expected 15 steps in repaired run, got %d", completedFork.TotalSteps)
	}

	// 4. Verify Diff between original failed run and time-traveled repaired run
	origFull, _ := store.GetTrajectory(failedTraj.ID)
	forkFull, _ := store.GetTrajectory(completedFork.ID)

	diff := engine.DiffTrajectories(origFull, forkFull)
	if diff.BaseStatus != models.StatusFailed || diff.CompareStatus != models.StatusCompleted {
		t.Errorf("diff status mismatch")
	}

	// Check step 12 in diff: failure vs success
	if len(diff.StepDiffs) < 13 {
		t.Fatalf("expected at least 13 diff steps, got %d", len(diff.StepDiffs))
	}
	step12Diff := diff.StepDiffs[12]
	if step12Diff.ToolCallsEqual {
		t.Errorf("step 12 tool calls should differ (failure vs success)")
	}
}
