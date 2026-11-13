# TraceReplay Engineering Plan

## 1. Overview
TraceReplay is a deterministic trajectory recorder, time-travel debugger, and regression test generator designed for autonomous AI agents. When a multi-step agent fails at an intermediate step (such as step 12 of a 15-step execution), engineers need deterministic reproduction without re-running previous external tool calls, without incurring redundant API costs, and without non-deterministic drift.

## 2. Core Architecture Components

### Component 1: Deterministic Storage & State Engine (Go + SQLite)
- Pure Go SQLite storage engine using `modernc.org/sqlite` for zero-CGO portable builds.
- Tables for `trajectories`, `steps`, `tool_calls`, `memory_snapshots`, and `fork_metadata`.
- Merkle / SHA-256 state hashing at each step to detect non-deterministic trajectory drift.
- Full support for branching: each rewind and mutation creates a fork child linked to its parent trajectory and fork step.

### Component 2: Virtual Replay Engine & Side-Effect Virtualizer
- Three execution modes:
  1. `RECORD`: Passes through requests to live models and tools while recording inputs, outputs, timestamps, and memory states.
  2. `MOCK_REPLAY`: Replays pre-recorded tool calls and LLM responses without external side effects or network calls.
  3. `HYBRID_MUTATE`: Steps prior to the fork point are served from recorded cache. From the fork point onward, mutated prompts, tool arguments, or memory states are executed through live models or user-defined mutations.

### Component 3: Regression Test Generator
- Auto-generates standalone regression suites from any failed or repaired trajectory:
  - Standalone Go test (`*_test.go`) with embedded HTTP mock server and step assertions.
  - Python pytest suite (`test_trajectory_regression.py`) with mock fixtures.
  - Deterministic JSON specification (`suite.json`) executable via `tracereplay test <suite.json>`.

### Component 4: OpenAI-Compatible Interceptor Proxy
- Reverse proxy listening on `/v1/chat/completions` and `/v1/responses`.
- Seamless integration with any existing agent framework (LangChain, LlamaIndex, AutoGen, CrewAI, custom loops) by configuring `OPENAI_BASE_URL`.
- Automatic trajectory session tracking via header `X-TraceReplay-Session` or auto-generated session IDs.

### Component 5: Realistic 15-Step Autonomous Agent Sandbox
- Built-in multi-tool agent demonstrating a production-grade problem:
  - Multi-file code refactoring and dependency migration scenario across 15 steps with 6 distinct tool calls.
  - Fails deterministically at step 12 due to a schema mismatch and hallucinated environment variable.
  - Provides a repeatable baseline to test time-travel rewind to step 8, prompt/argument mutation, and green completion.

### Component 6: React 19 Frontend Web UI
- Modern, clean, dark-mode first interface.
- Interactive timeline scrubber with step-by-step navigation, rewind, fast-forward, and jump-to-step.
- Execution DAG visualizer displaying tool calls, reasoning traces, and memory transitions.
- Interactive fork and mutation modal: edit prompt inputs, tool call arguments, and memory variables.
- Trajectory diff viewer comparing original failed run vs mutated fork run side by side.
- Regression test export drawer with code generation for Go, Python, and JSON.
- Live WebSocket streaming for real-time trajectory recording and replay status.

### Component 7: CLI Interface & Server
- `tracereplay serve --port 8080 --db ./tracereplay.db`
- `tracereplay agent run` (launches built-in 15-step demo agent)
- `tracereplay replay --id <id> --step <step>`
- `tracereplay export --id <id> --format <go|python|json> --output ./tests`
- `tracereplay test <suite.json>`

## 3. Implementation Phases

- [x] Phase 0: Repository setup, topics, plan, and documentation.
- [ ] Phase 1: Go backend data models, SQLite storage layer, and migrations.
- [ ] Phase 2: Deterministic state engine, SHA-256 state hashing, and trajectory recorder.
- [ ] Phase 3: Time-travel replay engine with mock virtualization and fork branching.
- [ ] Phase 4: OpenAI-compatible interceptor proxy.
- [ ] Phase 5: Multi-target regression test generator (Go, Python, JSON).
- [ ] Phase 6: Built-in 15-step autonomous agent demo with deterministic failure and recovery.
- [ ] Phase 7: REST API and WebSocket streaming server.
- [ ] Phase 8: React 19 frontend application (Vite, Tailwind CSS, Lucide icons, full UI controls).
- [ ] Phase 9: End-to-end integration, CLI verification, unit and regression tests.
- [ ] Phase 10: Complete documentation, humanized README, and repository finalization.
