package engine

import (
	"bytes"
	"encoding/json"

	"github.com/alexandrmotologa/tracereplay/pkg/models"
)

// DiffTrajectories computes the step-by-step diff between two trajectories.
func DiffTrajectories(base, comp *models.Trajectory) *models.TrajectoryDiff {
	diff := &models.TrajectoryDiff{
		BaseTrajectoryID:    base.ID,
		CompareTrajectoryID: comp.ID,
		BaseStatus:          base.Status,
		CompareStatus:       comp.Status,
		TokenDelta:          comp.TotalTokens - base.TotalTokens,
		CostDeltaUSD:        comp.TotalCostUSD - base.TotalCostUSD,
	}

	maxSteps := len(base.Steps)
	if len(comp.Steps) > maxSteps {
		maxSteps = len(comp.Steps)
	}

	diff.StepDiffs = make([]models.StepDiff, maxSteps)

	for i := 0; i < maxSteps; i++ {
		stepDiff := models.StepDiff{
			StepIndex: i,
		}

		var baseStep *models.Step
		var compStep *models.Step

		if i < len(base.Steps) {
			baseStep = &base.Steps[i]
			stepDiff.PromptBase = baseStep.Prompt
			stepDiff.ResponseBase = baseStep.Response
			stepDiff.BaseToolCalls = baseStep.ToolCalls
		}
		if i < len(comp.Steps) {
			compStep = &comp.Steps[i]
			stepDiff.PromptCompare = compStep.Prompt
			stepDiff.ResponseCompare = compStep.Response
			stepDiff.CompareToolCalls = compStep.ToolCalls
		}

		if baseStep != nil && compStep != nil {
			stepDiff.PromptEqual = baseStep.Prompt == compStep.Prompt
			stepDiff.ResponseEqual = baseStep.Response == compStep.Response
			stepDiff.DurationDeltaMS = compStep.DurationMS - baseStep.DurationMS

			// Compare Tool Calls
			stepDiff.ToolCallsEqual = compareToolCalls(baseStep.ToolCalls, compStep.ToolCalls)

			// Compare Memory Snapshots
			stepDiff.MemoryEqual = compareMemory(baseStep.MemorySnapshot, compStep.MemorySnapshot)
		} else {
			stepDiff.PromptEqual = false
			stepDiff.ResponseEqual = false
			stepDiff.ToolCallsEqual = false
			stepDiff.MemoryEqual = false
		}

		diff.StepDiffs[i] = stepDiff
	}

	return diff
}

func compareToolCalls(a, b []models.ToolCall) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ToolName != b[i].ToolName || a[i].IsError != b[i].IsError {
			return false
		}
		if !jsonEqual(a[i].Arguments, b[i].Arguments) {
			return false
		}
		if a[i].Output != b[i].Output {
			return false
		}
	}
	return true
}

func compareMemory(a, b *models.MemorySnapshot) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Scratchpad != b.Scratchpad {
		return false
	}
	return jsonEqual(a.Variables, b.Variables)
}

func jsonEqual(a, b json.RawMessage) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	var objA, objB any
	if err := json.Unmarshal(a, &objA); err != nil {
		return bytes.Equal(a, b)
	}
	if err := json.Unmarshal(b, &objB); err != nil {
		return bytes.Equal(a, b)
	}
	canA, _ := json.Marshal(objA)
	canB, _ := json.Marshal(objB)
	return bytes.Equal(canA, canB)
}
