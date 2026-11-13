# TraceReplay API and CLI Reference

## 1. REST Endpoints

### Trajectories
- `GET /api/v1/trajectories`
  - Query parameters: `status`, `agent_id`, `limit`, `offset`
  - Returns list of trajectories with step counts, total tokens, and status.
- `GET /api/v1/trajectories/:id`
  - Returns full trajectory details including all steps, tool calls, and memory snapshots.
- `POST /api/v1/trajectories`
  - Creates a new trajectory session.
- `POST /api/v1/trajectories/:id/steps`
  - Records a new step inside a trajectory.
- `POST /api/v1/trajectories/:id/complete`
  - Closes a trajectory session with status `completed` or `failed`.

### Time-Travel and Forking
- `POST /api/v1/trajectories/:id/fork`
  - Payload: `{ "fork_at_step": 8, "name": "fork-step-8-patched-prompt" }`
  - Creates a child trajectory branching from step 8.
- `POST /api/v1/trajectories/:id/replay`
  - Payload:
    ```json
    {
      "from_step": 8,
      "mutations": {
        "prompt": "Updated prompt fixing ambiguous instructions",
        "tool_call_override": {
          "tool_name": "db_migrate",
          "arguments": { "target_version": 14 }
        },
        "memory_overrides": {
          "retry_count": 0
        }
      }
    }
    ```
  - Replays execution from step 8 using recorded mock values for earlier steps and executing mutated step forward.

### Diffing and Regression Tests
- `GET /api/v1/diff?base=:id1&compare=:id2`
  - Returns step-by-step diff (prompts, tool parameters, outputs, memory, latency).
- `GET /api/v1/trajectories/:id/export?format=go|python|json`
  - Returns generated regression test code.
- `POST /api/v1/demo/run`
  - Triggers the built-in 15-step autonomous agent demo in the background.

## 2. WebSocket Endpoint
- `GET /api/v1/ws`
  - Subscribes to live trajectory events: `step_started`, `step_recorded`, `trajectory_updated`, `replay_progress`.

## 3. CLI Commands
- `tracereplay serve [--port 8080] [--db ./tracereplay.db]`
- `tracereplay agent run [--fail-at 12] [--steps 15]`
- `tracereplay replay --id <trajectory-id> --step <step-number>`
- `tracereplay export --id <trajectory-id> --format <go|python|json> --output ./tests`
- `tracereplay test <suite.json>`
- `tracereplay diff <trajectory-id-1> <trajectory-id-2>`
