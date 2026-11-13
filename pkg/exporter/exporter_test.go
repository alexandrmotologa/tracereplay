package exporter

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/alexandrmotologa/tracereplay/pkg/models"
)

func TestExporterFormats(t *testing.T) {
	traj := &models.Trajectory{
		ID:        "trj_export_test",
		AgentID:   "migration-worker",
		Model:     "gpt-4o",
		Status:    models.StatusCompleted,
		StartedAt: time.Now().UTC(),
		Steps: []models.Step{
			{
				StepIndex:   0,
				StepType:    models.StepTypePlanning,
				Prompt:      "Plan refactor",
				Response:    "Plan ready",
				StateHash:   "hash_001",
				Status:      models.StepStatusSuccess,
				DurationMS:  50,
				InputTokens: 100,
			},
			{
				StepIndex:    1,
				StepType:     models.StepTypeToolCall,
				Prompt:       "Run tool",
				Response:     "Tool finished",
				StateHash:    "hash_002",
				Status:       models.StepStatusSuccess,
				DurationMS:   150,
				OutputTokens: 80,
				ToolCalls: []models.ToolCall{
					{
						ToolName:  "fetch_schema",
						Arguments: json.RawMessage(`{"db":"main"}`),
						Output:    `{"tables": 12}`,
					},
				},
			},
		},
	}

	// 1. JSON
	jsonBytes, err := ExportJSON(traj)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}
	var suite RegressionSuite
	if err := json.Unmarshal(jsonBytes, &suite); err != nil {
		t.Fatalf("failed to unmarshal exported JSON: %v", err)
	}
	if suite.TrajectoryID != traj.ID || suite.TotalSteps != 2 {
		t.Errorf("JSON export mismatch: %+v", suite)
	}

	// 2. Go Test
	goCode, err := ExportGoTest(traj)
	if err != nil {
		t.Fatalf("ExportGoTest failed: %v", err)
	}
	if !strings.Contains(goCode, "TestTrajectory_trj_export_test") {
		t.Errorf("Go test does not contain expected function name")
	}
	if !strings.Contains(goCode, "hash_001") {
		t.Errorf("Go test missing step 0 hash")
	}

	// 3. Python pytest
	pyCode, err := ExportPythonTest(traj)
	if err != nil {
		t.Fatalf("ExportPythonTest failed: %v", err)
	}
	if !strings.Contains(pyCode, "test_trajectory_regression_steps_count") {
		t.Errorf("Python test missing test function")
	}
	if !strings.Contains(pyCode, "hash_002") {
		t.Errorf("Python test missing step 1 hash")
	}

	// 4. TypeScript (Vitest/Jest)
	tsCode, err := ExportTypeScriptTest(traj)
	if err != nil {
		t.Fatalf("ExportTypeScriptTest failed: %v", err)
	}
	if !strings.Contains(tsCode, "describe(\"TraceReplay Regression Suite") {
		t.Errorf("TypeScript test missing describe block")
	}
	if !strings.Contains(tsCode, "hash_001") {
		t.Errorf("TypeScript test missing step 0 stateHash")
	}
}
