package exporter

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alexandrmotologa/tracereplay/pkg/models"
)

func TestOpenInferenceExportAndImport(t *testing.T) {
	traj := &models.Trajectory{
		ID:        "trj_otel_test_01",
		AgentID:   "otel-researcher",
		Model:     "gpt-4o",
		Status:    models.StatusCompleted,
		StartedAt: time.Now().UTC(),
		Steps: []models.Step{
			{
				ID:          "step_0",
				StepIndex:   0,
				StepType:    models.StepTypeLLMGeneration,
				Prompt:      "Explain quantum annealing",
				Response:    "Quantum annealing is an optimization heuristic...",
				StateHash:   "hash_q_0",
				DurationMS:  120,
				InputTokens: 25,
				OutputTokens: 60,
				Status:      models.StepStatusSuccess,
			},
			{
				ID:          "step_1",
				StepIndex:   1,
				StepType:    models.StepTypeToolCall,
				Prompt:      "Search papers on D-Wave hardware",
				Response:    "Calling search_arxiv",
				StateHash:   "hash_q_1",
				DurationMS:  240,
				InputTokens: 30,
				OutputTokens: 20,
				Status:      models.StepStatusSuccess,
				ToolCalls: []models.ToolCall{
					{
						ToolName:  "search_arxiv",
						Arguments: json.RawMessage(`{"query":"D-Wave Advantage"}`),
						Output:    `[{"id":"arxiv.2104.12345","title":"Performance of D-Wave"}]`,
					},
				},
			},
		},
	}

	// 1. Export OpenInference JSON
	otelJSON, err := ExportOpenInferenceJSON(traj)
	if err != nil {
		t.Fatalf("ExportOpenInferenceJSON failed: %v", err)
	}

	var parsed OpenInferenceTrace
	if err := json.Unmarshal(otelJSON, &parsed); err != nil {
		t.Fatalf("failed to parse generated OpenInference JSON: %v", err)
	}
	if parsed.TraceID != traj.ID || len(parsed.Spans) != 2 {
		t.Fatalf("mismatch in exported trace: %+v", parsed)
	}
	if parsed.Spans[0].Kind != "LLM" || parsed.Spans[1].Kind != "TOOL" {
		t.Errorf("expected span kinds LLM and TOOL, got %s and %s", parsed.Spans[0].Kind, parsed.Spans[1].Kind)
	}

	// 2. Import OpenInference JSON back into TraceReplay
	imported, err := ImportOpenInferenceJSON(otelJSON)
	if err != nil {
		t.Fatalf("ImportOpenInferenceJSON failed: %v", err)
	}
	if imported.TotalSteps != 2 {
		t.Errorf("expected 2 imported steps, got %d", imported.TotalSteps)
	}
	if imported.Steps[1].ToolCalls[0].ToolName != "search_arxiv" {
		t.Errorf("imported tool call mismatch: %s", imported.Steps[1].ToolCalls[0].ToolName)
	}
}
