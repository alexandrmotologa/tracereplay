# `tracereplay` — Architecture & Engineering Master Plan
> **Deterministic Agent Trajectory Recorder, Time-Travel Debugger & Mock Replay Engine**  
> *Target: High-Performance Single-Binary Go Engine (< 20MB RAM) + Embedded React 19 Studio*

---

## 1. Executive Summary & Market Problem

### 1.1 The Non-Deterministic Debugging Crisis in AI Agents
Autonomous multi-step AI agents (coding agents, research swarms, transactional workflows) execute 10 to 30 sequential steps involving LLM reasoning, structured tool calls, database lookups, and state mutations. When an agent fails at Step 14 in production or staging:
1. **Impossible Reproduction**: Because LLMs are inherently stochastic and external APIs are dynamic, re-running the agent from Step 1 rarely reproduces the exact same failure path.
2. **Excessive Cost & Token Waste**: Debugging requires re-executing steps 1 through 13 repeatedly, wasting dollars in API tokens and waiting minutes per test iteration.
3. **Flaky Regression Testing**: Engineering teams lack a deterministic "flight recorder" format to convert real production failures into offline, zero-token regression test fixtures.

### 1.2 The `tracereplay` Solution
`tracereplay` is an open-source, local-first proxy and time-travel debugger written in Go with an embedded React 19 visual studio:
- **Zero-Overhead Transparent Recording**: Intercepts OpenAI, Anthropic, and local Ollama API calls. Serializes the full execution DAG (prompts, completions, tool call arguments, mock API responses, and memory states) into an immutable SQLite file (`.trace.db`).
- **Time-Travel Step Rewind**: Jump directly to Step $K$ (e.g., Step 8). Modify a system prompt, tweak a tool argument, or change a model parameter.
- **Mocked Pre-Replay (Zero-Cost Fast-Forward)**: Steps $1$ through $K-1$ replay instantly (< 5ms) using recorded deterministic mock outputs without making external network calls.
- **1-Click CI Regression Export**: Converts any recorded failure into an offline unit test (`go test`, `pytest`, or `vitest`) with zero live LLM dependencies.

---

## 2. High-Level System Architecture

```
       +--------------------------------------------------------+
       |             Autonomous Agent Application               |
       |         (LangChain, CrewAI, AutoGen, Custom)           |
       +--------------------------------------------------------+
                                   |
                                   | HTTP / SSE Traffic (OpenAI / Anthropic SDK)
                                   v
       +--------------------------------------------------------+
       |                  tracereplay Proxy                     |
       |          (Single Go Binary, Sub-5ms Overhead)          |
       +--------------------------------------------------------+
                                   |
                     +-------------+-------------+
                     |                           |
                     v                           v
     +-------------------------------+   +-------------------------------+
     |       Recording Engine        |   |       Time-Travel Replayer    |
     | - Hashes Prompt & Tool Args   |   | - Rewinds to Node Index K     |
     | - Stores Streaming Tokens     |   | - Serves Cached Tool Outputs  |
     | - Snapshots Memory State      |   | - Forks New Trajectory Branch |
     +-------------------------------+   +-------------------------------+
                     |                           |
                     +-------------+-------------+
                                   |
                                   v
       +--------------------------------------------------------+
       |         Local SQLite Trajectory Store (.trace.db)      |
       +--------------------------------------------------------+
                                   |
                                   v
       +--------------------------------------------------------+
       |          Embedded React 19 Debugger Studio             |
       |  - Visual Execution DAG / Step Waterfall               |
       |  - Diff Inspector (Prompt V1 vs Prompt V2)             |
       |  - Time-Travel Rewind & Fork Controls                  |
       |  - 1-Click Export to PyTest / Go Test Fixture          |
       +--------------------------------------------------------+
```

---

## 3. SQLite Trajectory Schema (`store/schema.sql`)

```sql
CREATE TABLE IF NOT EXISTS trajectories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    agent_framework TEXT,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    status TEXT NOT NULL -- 'SUCCESS', 'FAILED', 'REPLAYED'
);

CREATE TABLE IF NOT EXISTS trajectory_steps (
    id TEXT PRIMARY KEY,
    trajectory_id TEXT NOT NULL REFERENCES trajectories(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL,
    parent_step_id TEXT,
    node_type TEXT NOT NULL, -- 'LLM_REASONING', 'TOOL_CALL', 'STATE_MUTATION'
    model TEXT,
    prompt_json TEXT NOT NULL,
    completion_json TEXT NOT NULL,
    tool_name TEXT,
    tool_args_json TEXT,
    tool_output_json TEXT,
    memory_snapshot_json TEXT,
    latency_ms INTEGER NOT NULL,
    cost_usd REAL DEFAULT 0.0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_steps_traj ON trajectory_steps(trajectory_id, step_index);
```

---

## 4. CLI Command Surface

```bash
# Start tracereplay proxy & web debugger studio
tracereplay serve --port 8080 --db ./trajectories.db

# Wrap an agent execution directly via CLI
tracereplay record -- python agent.py

# Replay a failed trajectory up to step 7, substituting a new prompt
tracereplay replay --id traj_98a71 --rewind-to-step 7 --patch-prompt ./fix.txt

# Export a recorded failure as an offline pytest fixture
tracereplay export-fixture --id traj_98a71 --format pytest --output ./test_agent_fixed.py
```

---

## 5. Implementation Roadmap (5 Phases)

### Phase 1: High-Speed Reverse Proxy & Trajectory Serialization
- Implement transparent reverse proxy for OpenAI / Anthropic REST and streaming SSE.
- Build SQLite storage layer for step-by-step serialization and latency tracking.

### Phase 2: Time-Travel Engine & Forked Branches
- Implement rewind mechanism restoring memory snapshots and mocking steps $1$ to $K-1$.
- Support branching trajectories (comparing Original Failure vs Forked Success).

### Phase 3: Offline Fixture & Test Synthesizer
- Build exporter generating mock-backed unit tests for PyTest and Go `testing`.
- Zero-token CI regression gate verifying bug fixes deterministically.

### Phase 4: Embedded React 19 Time-Travel Studio
- Interactive step-by-step DAG view with token stream replay controls.
- Visual side-by-side prompt diffing and live re-execution button.

### Phase 5: Verification, Packaging & Docs
- Verify engine overhead is strictly under 5ms per call and binary under 20MB.
- Produce documentation, animated demo GIFs, and comprehensive examples.
