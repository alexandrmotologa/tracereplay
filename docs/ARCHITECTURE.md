# TraceReplay Architecture Specification

## 1. System Overview

TraceReplay acts as a deterministic black box recorder and virtual execution sandbox for autonomous agent workflows.

```
       +-------------------------------------------------------+
       |                  Autonomous Agent                     |
       |     (LangChain / CrewAI / Custom Go or Python Loop)   |
       +-------------------------------------------------------+
                                   |
                           HTTP / SDK Calls
                                   v
+---------------------------------------------------------------------+
|                             TraceReplay                             |
|                                                                     |
|  +--------------------+  +--------------------+  +---------------+  |
|  | Interceptor Proxy  |  | Time-Travel Engine |  | Test Exporter |  |
|  |   (/v1/chat/...)   |  |  (Fork & Virtual)  |  | (Go/Py/JSON)  |  |
|  +--------------------+  +--------------------+  +---------------+  |
|                                                                     |
|  +---------------------------------------------------------------+  |
|  |           Deterministic Storage Layer (SQLite pure Go)         |  |
|  |    Trajectories | Steps | ToolCalls | Memory | StateHashes    |  |
|  +---------------------------------------------------------------+  |
+---------------------------------------------------------------------+
                                   |
                           REST / WebSocket
                                   v
       +-------------------------------------------------------+
       |               TraceReplay Web UI                      |
       |      (React 19 + Vite + Tailwind + Scrubber)          |
       +-------------------------------------------------------+
```

## 2. Data Schema

### Trajectory (`trajectories`)
- `id` (TEXT PRIMARY KEY): Unique trajectory identifier (e.g., `trj_01h...`).
- `parent_id` (TEXT NULL): If this run is a fork from another trajectory.
- `forked_at_step` (INTEGER NULL): The step index where this fork originated.
- `name` (TEXT): Human readable run name (e.g., `code-migration-agent-prod`).
- `agent_id` (TEXT): Identifier of the agent workload.
- `model` (TEXT): Default LLM identifier.
- `status` (TEXT): `running`, `completed`, `failed`, `aborted`.
- `failure_reason` (TEXT NULL): Root-cause failure explanation.
- `total_steps` (INTEGER): Total count of steps recorded.
- `total_tokens` (INTEGER): Cumulative token consumption.
- `total_cost_usd` (REAL): Estimated financial cost of external API calls.
- `started_at` (TIMESTAMP): Epoch or ISO timestamp.
- `completed_at` (TIMESTAMP NULL): Completion timestamp.
- `metadata` (JSON): Arbitrary environment labels, git commit SHA, tags.

### Step (`steps`)
- `id` (TEXT PRIMARY KEY): Unique step identifier.
- `trajectory_id` (TEXT NOT NULL REFERENCES trajectories(id)): Parent trajectory.
- `step_index` (INTEGER NOT NULL): 0-indexed execution counter.
- `step_type` (TEXT NOT NULL): `planning`, `reasoning`, `tool_call`, `tool_result`, `llm_generation`, `memory_mutation`, `user_input`.
- `prompt` (TEXT): Input prompt or message payload sent to the LLM.
- `response` (TEXT): Raw text response or reasoning trace generated.
- `state_hash` (TEXT NOT NULL): SHA-256 fingerprint of the cumulative state.
- `duration_ms` (INTEGER): Latency in milliseconds.
- `input_tokens` (INTEGER): Prompt tokens.
- `output_tokens` (INTEGER): Completion tokens.
- `status` (TEXT): `success`, `failed`, `skipped`.
- `error_message` (TEXT NULL): Error text if this step failed.
- `created_at` (TIMESTAMP): Recorded timestamp.

### Tool Call (`tool_calls`)
- `id` (TEXT PRIMARY KEY): Unique call identifier.
- `step_id` (TEXT NOT NULL REFERENCES steps(id)): Parent step.
- `tool_name` (TEXT NOT NULL): Function or tool name (e.g., `run_test`, `fetch_file`).
- `arguments` (JSON NOT NULL): Arguments passed to the tool.
- `output` (TEXT NOT NULL): Raw output or JSON response returned by the tool.
- `is_error` (BOOLEAN NOT NULL): Whether the tool returned an error code or exception.
- `duration_ms` (INTEGER): Tool execution duration in milliseconds.
- `is_virtual` (BOOLEAN NOT NULL): Whether served from replay cache or live execution.

### Memory Snapshot (`memory_snapshots`)
- `id` (TEXT PRIMARY KEY): Snapshot identifier.
- `step_id` (TEXT NOT NULL REFERENCES steps(id)): Step where memory state was recorded.
- `variables` (JSON NOT NULL): Key-value state dictionary.
- `scratchpad` (TEXT): Agent working memory or scratchpad buffer.

## 3. Time-Travel Replay Mechanics

When rewinding to step `K`:
1. **Fork Creation**: A new trajectory row is created with `parent_id = source_trajectory.id` and `forked_at_step = K`.
2. **State Hydration**: Steps `0` through `K - 1` are copied into the new trajectory with their original recorded responses, tool outputs, and memory snapshots marked as `virtual_replay = true`.
3. **Point of Mutation**: Step `K` can accept modified parameters:
   - Updated system prompt or user query.
   - Patched tool call arguments.
   - Altered memory variables.
4. **Resumption**: The agent continues execution from step `K`. Calls to tools before step `K` are resolved instantly from the snapshot cache without hitting external endpoints.

## 4. State Hash Calculation

```
state_hash = SHA256(
    trajectory_id +
    step_index +
    canonical_json(memory_variables) +
    canonical_json(tool_calls_history) +
    prompt
)
```

This guarantees that any non-deterministic change during replay is flagged immediately as a drift event.
