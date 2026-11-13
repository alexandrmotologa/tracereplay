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
- `POST /api/v1/trajectories/import`
  - Imports an OpenInference/OpenTelemetry JSON trace into a replayable trajectory.
- `POST /api/v1/trajectories/:id/steps`
  - Records a new step inside a trajectory.
- `POST /api/v1/trajectories/:id/complete`
  - Closes a trajectory session with status `completed` or `failed`.

### Time-Travel, Branching & Hot-Switching
- `POST /api/v1/trajectories/:id/fork`
  - Payload: `{ "fork_at_step": 8, "name": "fork-step-8-patched-prompt" }`
  - Creates a child trajectory branching from step 8.
- `POST /api/v1/trajectories/:id/replay`
  - Payload:
    ```json
    {
      "from_step": 8,
      "name": "fork-step-8-repaired",
      "mutation": {
        "model_override": "ollama/qwen2.5-coder",
        "prompt": "Updated prompt fixing ambiguous instructions",
        "tool_call_override": {
          "tool_name": "db_migrate",
          "arguments": { "target_version": 12 }
        },
        "memory_overrides": {
          "retry_count": 0
        }
      }
    }
    ```
  - Replays execution from step 8 using recorded mock values for earlier steps and executing mutated step forward.

### Tool Side-Effect Virtualizer (Mock Gateway)
- `POST /api/v1/tools/execute` and `POST /v1/tools/execute`
  - Payload: `{ "trajectory_id": "trj_...", "step_index": 3, "tool_name": "fetch_db_schema", "arguments": {} }`
  - Intercepts external tool calls during replay: if the step occurred prior to the fork point, returns recorded mock output without invoking external APIs (zero side-effects).

### Diagnostics & Stability
- `GET /api/v1/trajectories/:id/diagnostics`
  - Returns AI/heuristic root-cause diagnosis, identifying the origin step of a failure and recommended fix.
- `GET /api/v1/trajectories/:id/stability`
  - Returns reproducibility and determinism matrix scoring each step from 0% to 100%.

### Diffing & Test Synthesis
- `GET /api/v1/diff?base=:id1&compare=:id2`
  - Returns step-by-step diff (prompts, tool parameters, outputs, memory, latency).
- `GET /api/v1/trajectories/:id/export?format=go|python|ts|otel|json`
  - Returns generated regression test code or OpenInference trace.
- `POST /api/v1/demo/run`
  - Triggers the built-in 15-step autonomous agent demo in the background.

## 2. Transparent Proxy Endpoints
- `POST /v1/chat/completions` (OpenAI format with SSE streaming support)
- `POST /v1/messages` (Anthropic Claude format)

## 3. WebSocket Endpoint
- `GET /api/v1/ws`
  - Subscribes to live trajectory events: `step_recorded`, `trajectory_started`, `trajectory_completed`, `trajectory_forked`.

## 4. CLI Commands
- `tracereplay serve [--port 8080] [--db ./trajectories.db]`
- `tracereplay record [--port 8080] [--db path] -- <command> [args...]`
- `tracereplay agent run [--fail=true|false] [--steps 15]`
- `tracereplay replay --id <trajectory-id> --step <step-number> [--prompt "..."] [--patch-prompt file]`
- `tracereplay export-fixture --id <trajectory-id> --format <go|python|ts|json> --output ./tests`
- `tracereplay test <suite.json>`
- `tracereplay diff <trajectory-id-1> <trajectory-id-2>`
