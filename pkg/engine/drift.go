package engine

import (
	"github.com/alexandrmotologa/tracereplay/pkg/models"
)

// TrajectoryStabilityReport details the reproducibility and flakiness risk of a trajectory.
type TrajectoryStabilityReport struct {
	TrajectoryID            string           `json:"trajectory_id"`
	OverallDeterminismScore float64          `json:"overall_determinism_score"` // 0.0 to 100.0%
	TotalSteps              int              `json:"total_steps"`
	ZeroDriftSteps          int              `json:"zero_drift_steps"`
	DriftRiskSteps          int              `json:"drift_risk_steps"`
	StepStability           []StepStability  `json:"step_stability"`
}

// StepStability provides determinism rating for an individual step.
type StepStability struct {
	StepIndex        int     `json:"step_index"`
	StepType         string  `json:"step_type"`
	StabilityScore   float64 `json:"stability_score"` // 0.0 to 1.0
	IsReplayMock     bool    `json:"is_replay_mock"`
	DriftRiskFactors []string `json:"drift_risk_factors,omitempty"`
}

// ComputeStabilityReport evaluates the determinism profile of an agent trajectory.
func ComputeStabilityReport(traj *models.Trajectory) *TrajectoryStabilityReport {
	if traj == nil || len(traj.Steps) == 0 {
		return nil
	}

	report := &TrajectoryStabilityReport{
		TrajectoryID:  traj.ID,
		TotalSteps:    len(traj.Steps),
		StepStability: make([]StepStability, 0, len(traj.Steps)),
	}

	var cumulativeScore float64

	for _, step := range traj.Steps {
		score := 1.0
		var risks []string

		if step.IsReplayMock {
			// Virtual replay steps are guaranteed 100% deterministic
			score = 1.0
		} else {
			// Heuristics based on step characteristics
			if step.StepType == models.StepTypeReasoning || step.StepType == models.StepTypeLLMGeneration {
				if step.OutputTokens > 300 {
					score -= 0.15
					risks = append(risks, "High token length variance risk")
				}
			}
			if step.Status == models.StepStatusFailed {
				score -= 0.30
				risks = append(risks, "Execution failure observed")
			}
			if len(step.ToolCalls) > 0 {
				for _, tc := range step.ToolCalls {
					if tc.IsError {
						score -= 0.20
						risks = append(risks, "Tool call failure")
					}
				}
			}
		}

		if score < 0.2 {
			score = 0.2
		}

		if score >= 0.95 {
			report.ZeroDriftSteps++
		} else {
			report.DriftRiskSteps++
		}

		cumulativeScore += score

		report.StepStability = append(report.StepStability, StepStability{
			StepIndex:        step.StepIndex,
			StepType:         string(step.StepType),
			StabilityScore:   score,
			IsReplayMock:     step.IsReplayMock,
			DriftRiskFactors: risks,
		})
	}

	report.OverallDeterminismScore = (cumulativeScore / float64(len(traj.Steps))) * 100.0
	return report
}
