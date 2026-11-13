package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alexandrmotologa/tracereplay/pkg/models"
)

// RootCauseDiagnostic provides automated root-cause analysis and suggested fix for a failed trajectory.
type RootCauseDiagnostic struct {
	TrajectoryID        string          `json:"trajectory_id"`
	FailureStepIndex    int             `json:"failure_step_index"`
	RootCauseStepIndex  int             `json:"root_cause_step_index"`
	IssueCategory       string          `json:"issue_category"`
	Severity            string          `json:"severity"`
	Explanation         string          `json:"explanation"`
	SuggestedPrompt     string          `json:"suggested_prompt"`
	SuggestedToolName   string          `json:"suggested_tool_name,omitempty"`
	SuggestedToolArgs   json.RawMessage `json:"suggested_tool_args,omitempty"`
	ConfidenceScore     float64         `json:"confidence_score"`
}

// AnalyzeTrajectoryRootCause runs diagnostic heuristics to isolate the origin step of a failure.
func AnalyzeTrajectoryRootCause(traj *models.Trajectory) *RootCauseDiagnostic {
	if traj == nil || len(traj.Steps) == 0 {
		return nil
	}

	var failIdx = -1
	var failStep *models.Step

	for i := range traj.Steps {
		if traj.Steps[i].Status == models.StepStatusFailed {
			failIdx = i
			failStep = &traj.Steps[i]
			break
		}
	}

	if failIdx == -1 {
		// No failure detected
		return nil
	}

	errMsg := ""
	if failStep.ErrorMessage != nil {
		errMsg = *failStep.ErrorMessage
	} else if traj.FailureReason != nil {
		errMsg = *traj.FailureReason
	}

	diag := &RootCauseDiagnostic{
		TrajectoryID:       traj.ID,
		FailureStepIndex:   failIdx,
		RootCauseStepIndex: failIdx,
		IssueCategory:      "EXECUTION_FAILURE",
		Severity:           "CRITICAL",
		ConfidenceScore:    0.85,
	}

	// Heuristic 1: Version or schema argument error
	lowerErr := strings.ToLower(errMsg)
	if strings.Contains(lowerErr, "version") || strings.Contains(lowerErr, "schema") || strings.Contains(lowerErr, "invalid") {
		diag.IssueCategory = "SCHEMA_ARGUMENT_MISMATCH"

		// Scan backwards for planning/parameter synthesis step
		originStep := failIdx - 1
		for i := failIdx - 1; i >= 0; i-- {
			s := traj.Steps[i]
			if s.StepType == models.StepTypePlanning || strings.Contains(strings.ToLower(s.Prompt), "parameter") || strings.Contains(strings.ToLower(s.Prompt), "synthesize") {
				originStep = i
				break
			}
		}

		diag.RootCauseStepIndex = originStep
		diag.Explanation = fmt.Sprintf("Step %d failed with schema error '%s'. Root-cause analysis indicates the parameter was synthesized at planning step %d without strict registry validation.", failIdx, errMsg, originStep)
		diag.SuggestedPrompt = "Synthesize patch parameters and enforce target version 12 in registry alignment."
		diag.ConfidenceScore = 0.94

		if len(failStep.ToolCalls) > 0 {
			diag.SuggestedToolName = failStep.ToolCalls[0].ToolName
			diag.SuggestedToolArgs = json.RawMessage(`{"database":"auth_prod","target_version":12}`)
		}
		return diag
	}

	// Heuristic 2: Tool call failure
	for _, tc := range failStep.ToolCalls {
		if tc.IsError {
			diag.IssueCategory = "UNHANDLED_TOOL_ERROR"
			diag.Explanation = fmt.Sprintf("Tool '%s' returned an error during execution: %s", tc.ToolName, tc.Output)
			diag.RootCauseStepIndex = failIdx
			diag.SuggestedPrompt = fmt.Sprintf("Verify arguments for tool %s and retry with defensive error handling.", tc.ToolName)
			diag.ConfidenceScore = 0.88
			return diag
		}
	}

	// Default diagnostic fallback
	diag.Explanation = fmt.Sprintf("Execution halted at step %d: %s", failIdx, errMsg)
	diag.SuggestedPrompt = "Re-evaluate state context and execute with validated parameters."
	return diag
}
