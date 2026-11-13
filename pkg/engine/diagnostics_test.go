package engine

import (
	"testing"
	"time"

	"github.com/alexandrmotologa/tracereplay/pkg/models"
)

func TestRootCauseDiagnostic(t *testing.T) {
	failMsg := "Migration schema error: target version 'invalid_v99' does not exist in registry"
	traj := &models.Trajectory{
		ID:        "trj_fail_test",
		StartedAt: time.Now().UTC(),
		Steps: []models.Step{
			{
				StepIndex: 0,
				StepType:  models.StepTypePlanning,
				Prompt:    "Initialize plan",
				Status:    models.StepStatusSuccess,
			},
			{
				StepIndex: 1,
				StepType:  models.StepTypePlanning,
				Prompt:    "Synthesize patch parameters for migration configuration",
				Status:    models.StepStatusSuccess,
			},
			{
				StepIndex:    2,
				StepType:     models.StepTypeToolCall,
				Prompt:       "Run migration",
				Status:       models.StepStatusFailed,
				ErrorMessage: &failMsg,
				ToolCalls: []models.ToolCall{
					{
						ToolName: "run_migration",
						IsError:  true,
					},
				},
			},
		},
	}

	diag := AnalyzeTrajectoryRootCause(traj)
	if diag == nil {
		t.Fatalf("expected diagnostic for failed trajectory")
	}

	if diag.FailureStepIndex != 2 {
		t.Errorf("expected failure step 2, got %d", diag.FailureStepIndex)
	}
	if diag.RootCauseStepIndex != 1 {
		t.Errorf("expected root cause step 1, got %d", diag.RootCauseStepIndex)
	}
	if diag.IssueCategory != "SCHEMA_ARGUMENT_MISMATCH" {
		t.Errorf("expected SCHEMA_ARGUMENT_MISMATCH, got %s", diag.IssueCategory)
	}
	if diag.ConfidenceScore < 0.9 {
		t.Errorf("expected high confidence score, got %f", diag.ConfidenceScore)
	}
}
