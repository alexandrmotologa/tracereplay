package agentdemo

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alexandrmotologa/tracereplay/pkg/engine"
	"github.com/alexandrmotologa/tracereplay/pkg/models"
)

// DemoConfig sets parameters for running the 15-step agent scenario.
type DemoConfig struct {
	ShouldFailAtStep12 bool
	FixedVersion       int
	CustomPromptStep8  string
	DelayBetweenSteps  time.Duration
}

// Run15StepDemo executes the realistic autonomous agent trajectory.
func Run15StepDemo(eng *engine.Engine, cfg DemoConfig) (*models.Trajectory, error) {
	name := "autonomous-code-migration-agent"
	if !cfg.ShouldFailAtStep12 {
		name = "autonomous-code-migration-agent-repaired"
	}

	metadata := map[string]any{
		"agent_type": "autonomous_refactor_agent",
		"scenario":   "auth_microservice_migration",
		"env":        "production_simulation",
	}

	traj, err := eng.StartRecording(name, "auth-migration-bot", "claude-3-7-sonnet", metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to start trajectory: %w", err)
	}

	// 15 discrete production-grade steps
	for stepIdx := 0; stepIdx < 15; stepIdx++ {
		step, isFailure := buildStep(traj.ID, stepIdx, cfg)

		if err := eng.RecordStep(step); err != nil {
			return nil, fmt.Errorf("failed to record step %d: %w", stepIdx, err)
		}

		if isFailure {
			failReason := "Step 12 failed: Migration schema error with invalid version specification"
			return eng.CompleteTrajectory(traj.ID, models.StatusFailed, &failReason)
		}

		if cfg.DelayBetweenSteps > 0 {
			time.Sleep(cfg.DelayBetweenSteps)
		}
	}

	return eng.CompleteTrajectory(traj.ID, models.StatusCompleted, nil)
}

func buildStep(trajID string, idx int, cfg DemoConfig) (*models.Step, bool) {
	now := time.Now().UTC()
	step := &models.Step{
		TrajectoryID: trajID,
		StepIndex:    idx,
		CreatedAt:    now,
		Status:       models.StepStatusSuccess,
	}

	switch idx {
	case 0:
		step.StepType = models.StepTypePlanning
		step.Prompt = "User Request: Migrate auth service from legacy SHA1 to Argon2id and execute migration v12."
		step.Response = "Plan established: 1. Explore directory structure. 2. Fetch database schema. 3. Check legacy syntax. 4. Write migration script. 5. Apply migration. 6. Verify integration tests."
		step.DurationMS = 380
		step.InputTokens = 120
		step.OutputTokens = 85
		step.MemorySnapshot = &models.MemorySnapshot{
			Variables:  json.RawMessage(`{"stage":"initial_planning","target_version":12,"status":"in_progress"}`),
			Scratchpad: "Identified target version v12. Preparing initial file scan.",
		}

	case 1:
		step.StepType = models.StepTypeToolCall
		step.Prompt = "List source code files in ./services/auth to locate password hashing logic."
		step.Response = "Invoking read_repo_tree on directory ./services/auth"
		step.DurationMS = 210
		step.InputTokens = 145
		step.OutputTokens = 35
		step.ToolCalls = []models.ToolCall{
			{
				ToolName:   "read_repo_tree",
				Arguments:  json.RawMessage(`{"path":"./services/auth","depth":2}`),
				Output:     `["controller.go","hasher.go","schema.sql","config.yaml"]`,
				DurationMS: 45,
			},
		}

	case 2:
		step.StepType = models.StepTypeReasoning
		step.Prompt = "Analyze directory contents and inspect hasher.go dependencies."
		step.Response = "Found legacy hasher implementation in hasher.go utilizing deprecated crypto/sha1. We must introduce argon2 dependencies and update schema definitions."
		step.DurationMS = 420
		step.InputTokens = 230
		step.OutputTokens = 95
		step.MemorySnapshot = &models.MemorySnapshot{
			Variables:  json.RawMessage(`{"detected_files":["controller.go","hasher.go","schema.sql"],"legacy_crypto":"sha1"}`),
			Scratchpad: "Confirmed legacy crypto is sha1. Proceeding to inspect active database schema.",
		}

	case 3:
		step.StepType = models.StepTypeToolCall
		step.Prompt = "Fetch current database schema to inspect password_hash column constraint."
		step.Response = "Executing fetch_db_schema tool for table users."
		step.DurationMS = 280
		step.InputTokens = 190
		step.OutputTokens = 40
		step.ToolCalls = []models.ToolCall{
			{
				ToolName:   "fetch_db_schema",
				Arguments:  json.RawMessage(`{"table":"users","database":"auth_prod"}`),
				Output:     `{"table":"users","columns":[{"name":"id","type":"UUID"},{"name":"password_hash","type":"VARCHAR(40)"},{"name":"version","type":"INT"}]}`,
				DurationMS: 62,
			},
		}

	case 4:
		step.StepType = models.StepTypeReasoning
		step.Prompt = "Evaluate schema constraint: VARCHAR(40) is insufficient for 97-character Argon2id encoded hashes."
		step.Response = "Critical finding: password_hash column length is VARCHAR(40). An Argon2id hash requires at least VARCHAR(128). An ALTER TABLE migration is mandatory."
		step.DurationMS = 360
		step.InputTokens = 310
		step.OutputTokens = 75
		step.MemorySnapshot = &models.MemorySnapshot{
			Variables:  json.RawMessage(`{"alter_table_required":true,"required_column_size":"VARCHAR(128)"}`),
			Scratchpad: "Column length expansion required. Will draft migration SQL.",
		}

	case 5:
		step.StepType = models.StepTypeToolCall
		step.Prompt = "Run static analysis linter on ./services/auth/controller.go before edits."
		step.Response = "Invoking run_linter on target controller.go"
		step.DurationMS = 310
		step.InputTokens = 210
		step.OutputTokens = 30
		step.ToolCalls = []models.ToolCall{
			{
				ToolName:   "run_linter",
				Arguments:  json.RawMessage(`{"file":"./services/auth/controller.go","ruleset":"strict"}`),
				Output:     `{"status":"clean","warnings":0,"lint_errors":0}`,
				DurationMS: 85,
			},
		}

	case 6:
		step.StepType = models.StepTypeReasoning
		step.Prompt = "Prepare migration steps and confirm safe database transaction wrapping."
		step.Response = "All baseline files are lint-clean. Structuring migration script v12 with transactional DDL: BEGIN TRANSACTION; ALTER TABLE users ALTER COLUMN password_hash TYPE VARCHAR(128); COMMIT;"
		step.DurationMS = 290
		step.InputTokens = 180
		step.OutputTokens = 60

	case 7:
		step.StepType = models.StepTypeMemoryMutation
		step.Prompt = "Persist target schema version and verified AST modifications into agent working memory."
		step.Response = "Memory updated with verified target constraints."
		step.DurationMS = 110
		step.InputTokens = 90
		step.OutputTokens = 20
		step.MemorySnapshot = &models.MemorySnapshot{
			Variables:  json.RawMessage(`{"target_version":12,"schema_migration_ready":true,"retry_budget":3}`),
			Scratchpad: "State verified. Ready to configure patch parameters at step 8.",
		}

	case 8:
		step.StepType = models.StepTypePlanning
		step.Prompt = "Synthesize patch parameters for migration configuration."
		if cfg.CustomPromptStep8 != "" {
			step.Prompt = cfg.CustomPromptStep8
		}
		step.Response = "Validating target database environment variables and version alignment."
		step.DurationMS = 270
		step.InputTokens = 200
		step.OutputTokens = 65

	case 9:
		step.StepType = models.StepTypeToolCall
		step.Prompt = "Write migration file ./migrations/v12_auth_upgrade.sql."
		step.Response = "Writing migration script using write_patch_file"
		step.DurationMS = 240
		step.InputTokens = 250
		step.OutputTokens = 35
		step.ToolCalls = []models.ToolCall{
			{
				ToolName:   "write_patch_file",
				Arguments:  json.RawMessage(`{"path":"./migrations/v12_auth_upgrade.sql","content":"ALTER TABLE users ALTER COLUMN password_hash TYPE VARCHAR(128);"}`),
				Output:     `{"status":"written","bytes":72}`,
				DurationMS: 28,
			},
		}

	case 10:
		step.StepType = models.StepTypeReasoning
		step.Prompt = "Review generated file integrity and prepare for execution."
		step.Response = "File successfully written. Checksum verified. Proceeding to execution phase."
		step.DurationMS = 220
		step.InputTokens = 160
		step.OutputTokens = 45

	case 11:
		step.StepType = models.StepTypeMemoryMutation
		step.Prompt = "Update state variables prior to running database migration."
		step.Response = "Setting execution_lock=true and recording pre-migration checkpoint."
		step.DurationMS = 95
		step.InputTokens = 110
		step.OutputTokens = 25
		step.MemorySnapshot = &models.MemorySnapshot{
			Variables:  json.RawMessage(`{"execution_lock":true,"pre_migration_backup":"snapshot_20261113"}`),
			Scratchpad: "Checkpoint created. Running migration tool.",
		}

	case 12:
		step.StepType = models.StepTypeToolCall
		if cfg.ShouldFailAtStep12 {
			// Buggy execution scenario
			step.Prompt = "Execute database schema migration with target version."
			step.Response = "Triggering run_migration with hallucinated parameter target_version=invalid_v99"
			step.DurationMS = 450
			step.InputTokens = 320
			step.OutputTokens = 50
			step.Status = models.StepStatusFailed
			errMsg := "Migration schema error: target version 'invalid_v99' does not exist in registry"
			step.ErrorMessage = &errMsg
			step.ToolCalls = []models.ToolCall{
				{
					ToolName:   "run_migration",
					Arguments:  json.RawMessage(`{"database":"auth_prod","target_version":"invalid_v99"}`),
					Output:     `{"error":"Fatal: version invalid_v99 does not exist in ./migrations","code":500}`,
					IsError:    true,
					DurationMS: 180,
				},
			}
			return step, true
		} else {
			// Fixed execution scenario
			version := 12
			if cfg.FixedVersion > 0 {
				version = cfg.FixedVersion
			}
			step.Prompt = "Execute database schema migration with validated target version 12."
			step.Response = "Triggering run_migration with validated target_version=12"
			step.DurationMS = 310
			step.InputTokens = 320
			step.OutputTokens = 55
			step.Status = models.StepStatusSuccess
			step.ToolCalls = []models.ToolCall{
				{
					ToolName:   "run_migration",
					Arguments:  json.RawMessage(fmt.Sprintf(`{"database":"auth_prod","target_version":%d}`, version)),
					Output:     `{"status":"applied","applied_version":12,"migrated_rows":1420,"duration_ms":110}`,
					IsError:    false,
					DurationMS: 110,
				},
			}
		}

	case 13:
		step.StepType = models.StepTypeToolCall
		step.Prompt = "Run end-to-end integration tests to verify Argon2id hash generation and database persistence."
		step.Response = "Invoking run_integration_tests on suite auth_e2e"
		step.DurationMS = 850
		step.InputTokens = 290
		step.OutputTokens = 60
		step.ToolCalls = []models.ToolCall{
			{
				ToolName:   "run_integration_tests",
				Arguments:  json.RawMessage(`{"suite":"auth_e2e","concurrency":4}`),
				Output:     `{"tests_run":18,"passed":18,"failed":0,"coverage":"94.2%"}`,
				DurationMS: 620,
			},
		}

	case 14:
		step.StepType = models.StepTypePlanning
		step.Prompt = "Finalize migration summary, clean temporary artifacts, and output release report."
		step.Response = "Migration completed successfully. Auth service upgraded to Argon2id. Database column resized to VARCHAR(128). 18/18 integration tests passing."
		step.DurationMS = 210
		step.InputTokens = 210
		step.OutputTokens = 120
		step.MemorySnapshot = &models.MemorySnapshot{
			Variables:  json.RawMessage(`{"status":"success","migrated_version":12,"tests_passed":18}`),
			Scratchpad: "Trajectory execution verified and closed with 100% test coverage.",
		}
	}

	return step, false
}
