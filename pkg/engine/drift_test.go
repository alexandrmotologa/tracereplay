package engine

import (
	"testing"

	"github.com/alexandrmotologa/tracereplay/pkg/models"
)

func TestStabilityReport(t *testing.T) {
	traj := &models.Trajectory{
		ID: "trj_stability_test",
		Steps: []models.Step{
			{
				StepIndex:    0,
				StepType:     models.StepTypePlanning,
				IsReplayMock: true,
			},
			{
				StepIndex:    1,
				StepType:     models.StepTypeLLMGeneration,
				OutputTokens: 450,
			},
		},
	}

	report := ComputeStabilityReport(traj)
	if report == nil {
		t.Fatalf("expected stability report")
	}

	if report.TotalSteps != 2 {
		t.Errorf("expected 2 total steps, got %d", report.TotalSteps)
	}
	if report.StepStability[0].StabilityScore != 1.0 {
		t.Errorf("step 0 should be 1.0 (replay mock), got %f", report.StepStability[0].StabilityScore)
	}
	if report.StepStability[1].StabilityScore >= 1.0 {
		t.Errorf("step 1 should have risk factor penalty, got %f", report.StepStability[1].StabilityScore)
	}
}
