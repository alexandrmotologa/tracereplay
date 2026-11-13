package exporter

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alexandrmotologa/tracereplay/pkg/models"
	"github.com/google/uuid"
)

// OpenInferenceTrace represents the standard OpenTelemetry/OpenInference GenAI trace format.
type OpenInferenceTrace struct {
	Version      string               `json:"version"`
	TraceID      string               `json:"trace_id"`
	AgentID      string               `json:"agent_id"`
	Spans        []OpenInferenceSpan  `json:"spans"`
}

// OpenInferenceSpan represents an individual OpenInference execution span.
type OpenInferenceSpan struct {
	SpanID       string         `json:"span_id"`
	ParentSpanID *string        `json:"parent_span_id,omitempty"`
	Name         string         `json:"name"`
	Kind         string         `json:"kind"` // LLM, TOOL, CHAIN
	StartTime    string         `json:"start_time"`
	EndTime      string         `json:"end_time"`
	Status       string         `json:"status"` // OK, ERROR
	Attributes   map[string]any `json:"attributes"`
}

// ExportOpenInferenceJSON exports a trajectory in the OpenInference trace standard.
func ExportOpenInferenceJSON(t *models.Trajectory) ([]byte, error) {
	traceID := t.ID
	spans := make([]OpenInferenceSpan, 0, len(t.Steps))

	for _, step := range t.Steps {
		spanID := step.ID
		kind := "CHAIN"
		switch step.StepType {
		case models.StepTypeLLMGeneration:
			kind = "LLM"
		case models.StepTypeToolCall:
			kind = "TOOL"
		case models.StepTypePlanning, models.StepTypeReasoning:
			kind = "CHAIN"
		}

		status := "OK"
		if step.Status == models.StepStatusFailed {
			status = "ERROR"
		}

		startTime := step.CreatedAt
		endTime := startTime.Add(time.Duration(step.DurationMS) * time.Millisecond)

		attrs := map[string]any{
			"openinference.span.kind":          kind,
			"input.value":                      step.Prompt,
			"output.value":                     step.Response,
			"llm.model_name":                   t.Model,
			"llm.token_count.prompt_tokens":    step.InputTokens,
			"llm.token_count.completion_tokens": step.OutputTokens,
			"llm.token_count.total_tokens":     step.InputTokens + step.OutputTokens,
			"step.index":                       step.StepIndex,
			"step.state_hash":                  step.StateHash,
			"step.is_replay_mock":              step.IsReplayMock,
		}

		if len(step.ToolCalls) > 0 {
			attrs["tool.name"] = step.ToolCalls[0].ToolName
			attrs["tool.parameters"] = string(step.ToolCalls[0].Arguments)
			attrs["tool.output"] = step.ToolCalls[0].Output
		}

		spans = append(spans, OpenInferenceSpan{
			SpanID:     spanID,
			Name:       fmt.Sprintf("%s_%s", t.AgentID, step.StepType),
			Kind:       kind,
			StartTime:  startTime.Format(time.RFC3339Nano),
			EndTime:    endTime.Format(time.RFC3339Nano),
			Status:     status,
			Attributes: attrs,
		})
	}

	trace := OpenInferenceTrace{
		Version: "openinference-v1",
		TraceID: traceID,
		AgentID: t.AgentID,
		Spans:   spans,
	}

	return json.MarshalIndent(trace, "", "  ")
}

// ImportOpenInferenceJSON parses an OpenInference JSON payload into a TraceReplay trajectory.
func ImportOpenInferenceJSON(data []byte) (*models.Trajectory, error) {
	var trace OpenInferenceTrace
	if err := json.Unmarshal(data, &trace); err != nil {
		return nil, fmt.Errorf("failed to parse OpenInference JSON: %w", err)
	}

	traj := &models.Trajectory{
		ID:        "trj_imported_" + uuid.New().String()[:8],
		Name:      fmt.Sprintf("imported-otel-%s", trace.TraceID),
		AgentID:   trace.AgentID,
		Model:     "imported-model",
		Status:    models.StatusCompleted,
		StartedAt: time.Now().UTC(),
		Metadata: map[string]any{
			"source":          "openinference_import",
			"origin_trace_id": trace.TraceID,
		},
	}

	for idx, span := range trace.Spans {
		stepType := models.StepTypePlanning
		if span.Kind == "LLM" {
			stepType = models.StepTypeLLMGeneration
		} else if span.Kind == "TOOL" {
			stepType = models.StepTypeToolCall
		}

		stepStatus := models.StepStatusSuccess
		if span.Status == "ERROR" {
			stepStatus = models.StepStatusFailed
			traj.Status = models.StatusFailed
		}

		prompt, _ := span.Attributes["input.value"].(string)
		resp, _ := span.Attributes["output.value"].(string)

		step := models.Step{
			ID:           span.SpanID,
			TrajectoryID: traj.ID,
			StepIndex:    idx,
			StepType:     stepType,
			Prompt:       prompt,
			Response:     resp,
			StateHash:    fmt.Sprintf("imported_hash_%d", idx),
			DurationMS:   50,
			Status:       stepStatus,
			CreatedAt:    time.Now().UTC(),
		}

		if toolName, ok := span.Attributes["tool.name"].(string); ok && toolName != "" {
			toolParams, _ := span.Attributes["tool.parameters"].(string)
			toolOutput, _ := span.Attributes["tool.output"].(string)
			step.ToolCalls = []models.ToolCall{
				{
					ToolName:  toolName,
					Arguments: json.RawMessage(toolParams),
					Output:    toolOutput,
				},
			}
		}

		traj.Steps = append(traj.Steps, step)
		traj.TotalSteps++
	}

	return traj, nil
}
