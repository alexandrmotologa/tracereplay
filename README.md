<p align="center">
  <img src="docs/images/logo.png?raw=true" alt="TraceReplay Logo" width="130" style="border-radius: 24px;" />
</p>

<h1 align="center">TraceReplay</h1>

<p align="center">
  <strong>Deterministic Agent Trajectory Recorder, Time-Travel Debugger, and Regression Test Generator</strong>
</p>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat&logo=go" alt="Go Version" /></a>
  <a href="https://react.dev"><img src="https://img.shields.io/badge/React-19-61DAFB?style=flat&logo=react" alt="React 19" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License" /></a>
  <a href="docs/ARCHITECTURE.md"><img src="https://img.shields.io/badge/Single%20Binary-<20MB-emerald.svg" alt="Single Binary" /></a>
</p>

<p align="center">
  <img src="docs/images/tracereplay_demo.gif?raw=true" alt="TraceReplay Live Demo" width="100%" style="border-radius: 12px; border: 1px solid #1e293b;" />
</p>

---

## Overview

TraceReplay is a local-first flight recorder and deterministic time-travel debugger for autonomous AI agents.

When an autonomous multi-step agent fails at Step 12 of a 15-step execution in production or staging, reproducing the defect is traditionally difficult. Because LLMs are non-deterministic and external tools produce stateful mutations, running the agent again from Step 0 rarely reproduces the exact failure and wastes substantial API tokens.

TraceReplay solves this by capturing every step (prompts, tool calls, API completions, memory state variables, and execution latencies) into an immutable SQLite flight recording. Engineers can rewind directly to Step 8, modify a prompt or tool argument, and resume execution forward while earlier steps (0 to 7) are resolved instantly from the recorded mock cache with zero network calls and zero token consumption.

---

## Visual Studio Interface

TraceReplay includes an embedded React 19 visual debugging studio compiled directly into the Go binary.

### 1. Step Inspector, Failure Detection & AI Root-Cause Diagnostic

Inspect execution timelines, SHA-256 state fingerprints, token metrics, and automated root-cause diagnostics isolating synthesized bugs.

<p align="center">
  <img src="docs/images/tracereplay_dashboard.png?raw=true" alt="TraceReplay Dashboard & Failure Diagnostic" width="100%" style="border-radius: 10px; border: 1px solid #1e293b;" />
</p>

### 2. Execution DAG Flow & Timeline Genealogy

Visualize sequential execution nodes, branch genealogies, and tool execution states in real time.

<p align="center">
  <img src="docs/images/tracereplay_dag_flow.png?raw=true" alt="Execution DAG Flow" width="100%" style="border-radius: 10px; border: 1px solid #1e293b;" />
</p>

### 3. Time-Travel Rewind & State Mutation

Rewind to any historical step, modify prompt instructions or tool arguments, and resume execution with deterministic caching for pre-rewind steps.

<p align="center">
  <img src="docs/images/tracereplay_rewind.png?raw=true" alt="Time-Travel Rewind Modal" width="100%" style="border-radius: 10px; border: 1px solid #1e293b;" />
</p>

### 4. Side-by-Side Trajectory Diff Inspector

Compare baseline failed executions against repaired replays to verify latency delta, token savings, and state convergence.

<p align="center">
  <img src="docs/images/tracereplay_diff.png?raw=true" alt="Side-by-Side Trajectory Diff" width="100%" style="border-radius: 10px; border: 1px solid #1e293b;" />
</p>

### 5. Responsive Mobile & Tablet Layout

Full debugging capabilities on mobile devices with sliding drawer navigation and touch-optimized scrubbers.

<p align="center">
  <img src="docs/images/tracereplay_mobile.png?raw=true" alt="TraceReplay Mobile Interface" width="340" style="border-radius: 16px; border: 1px solid #1e293b;" />
</p>

---

## Key Capabilities

- **Zero-Code Transparent Proxy**: Intercepts OpenAI, Anthropic, and local Ollama API calls without requiring SDK code modifications. Set `OPENAI_BASE_URL=http://localhost:8080/v1` or use `tracereplay record -- python agent.py`.
- **Deterministic State Fingerprints**: Calculates SHA-256 state hashes across prompt history, canonicalized tool calls, and memory variables to detect non-deterministic trajectory drift.
- **Time-Travel Rewind, Forking & Hot-Switching**: Rewinds execution to step K. Pre-fork steps are served from virtual cache (< 5ms latency). Supports model hot-switching (testing with local Ollama or Claude).
- **Tool Side-Effect Virtualizer (Mock Gateway)**: Intercepts external tool invocations at `/api/v1/mock/tool`. Pre-fork steps return recorded mock outputs, guaranteeing zero accidental side-effects during replay.
- **Root-Cause Diagnostic & AI Auto-Fix**: Automatically isolates the step where hallucinations or invalid parameters were synthesized, offering one-click rewind mutations.
- **1-Click Multi-Language Regression Exporter**: Auto-generates standalone, offline unit tests for Go (`testing`), Python (`pytest`), TypeScript (`vitest`/`jest`), and OpenInference/OTel traces.
- **Execution DAG Flow in React 19 Studio**: Visual graph showing sequential nodes, branch genealogies, and tool execution states.
- **High-Performance Single Binary**: Pure Go backend using embedded SQLite and embedded React 19 studio. Binary size is under 16MB with sub-5ms proxy overhead and zero external CGO dependencies.
- **Built-in 15-Step Agent Sandbox**: Includes a reproducible, multi-tool autonomous code migration scenario demonstrating failure at step 12 and time-travel recovery from step 8.

---

## Architecture

```
       +--------------------------------------------------------+
       |             Autonomous Agent Application               |
       |       (LangChain, CrewAI, AutoGen, Custom Loop)        |
       +--------------------------------------------------------+
                                   |
                                   | HTTP / SSE Traffic
                                   v
       +--------------------------------------------------------+
       |                  TraceReplay Engine                    |
       |   +-------------------+       +--------------------+   |
       |   | Interceptor Proxy |       | Time-Travel Replay |   |
       |   +-------------------+       +--------------------+   |
       |   | State Hash Engine |       | Test Synthesizer   |   |
       |   +-------------------+       +--------------------+   |
       +--------------------------------------------------------+
                                   |
                                   v
       +--------------------------------------------------------+
       |        Local SQLite Trajectory Store (.trace.db)       |
       +--------------------------------------------------------+
                                   |
                                   v
       +--------------------------------------------------------+
       |          Embedded React 19 Studio Interface            |
       |  - Scrubber Timeline & Step Inspector                  |
       |  - Side-by-Side Trajectory Diff Viewer                 |
       |  - Time-Travel Rewind & State Mutation Modal           |
       |  - Regression Test Exporter (Go, Python, Vitest, OTel) |
       +--------------------------------------------------------+
```

---

## Quick Start

### Installation

Download pre-built binaries from Releases or build from source:

```bash
git clone https://github.com/alexandrmotologa/tracereplay.git
cd tracereplay
make build
```

The resulting standalone binary is located at `bin/tracereplay` (or `bin/tracereplay.exe` on Windows).

### 1. Launch the Server & Visual Debugger Studio

```bash
./bin/tracereplay serve --port 8080 --db ./trajectories.db
```

Open `http://localhost:8080` in your browser to inspect recorded runs.

### 2. Wrap and Record an Agent Run

Wrap any existing agent without modifying application code:

```bash
# Wrap a Python agent
./bin/tracereplay record -- python agent.py

# Wrap a Node.js agent
./bin/tracereplay record -- node agent.js
```

The CLI starts a background proxy, injects `OPENAI_BASE_URL` and `ANTHROPIC_BASE_URL`, records all completions, and outputs a flight summary upon exit.

### 3. Run the Built-in 15-Step Agent Demo

TraceReplay includes a reproducible 15-step scenario:

```bash
# Run baseline scenario that fails at step 12
./bin/tracereplay agent run --fail=true

# Run clean scenario where all 15 steps succeed
./bin/tracereplay agent run --fail=false
```

### 4. Time-Travel Rewind via CLI

Rewind a failed run to step 8, inject a corrected prompt, and fork a new branch:

```bash
./bin/tracereplay replay \
  --id trj_f613bf9d-1316-48d8-ba91-17f2441ebfdd \
  --step 8 \
  --prompt "Inject validated migration schema v12 with Argon2id parameters and execute DDL transaction."
```

### 5. Compare Trajectories (Diff View)

Compare original failed execution against the repaired forked execution:

```bash
./bin/tracereplay diff trj_original trj_forked
```

### 6. Export Regression Tests

Generate offline regression suites for CI/CD pipelines:

```bash
# Export Go regression test
./bin/tracereplay export-fixture -id trj_f613bf9d -format go -output ./tests/agent_test.go

# Export Python pytest suite
./bin/tracereplay export-fixture -id trj_f613bf9d -format pytest -output ./tests/test_agent.py

# Export TypeScript Vitest suite
./bin/tracereplay export-fixture -id trj_f613bf9d -format vitest -output ./tests/agent.test.ts

# Export portable JSON suite and verify determinism
./bin/tracereplay export-fixture -id trj_f613bf9d -format json -output ./tests/suite.json
./bin/tracereplay test ./tests/suite.json
```

---

## CLI Reference

| Command | Description | Flags |
|---|---|---|
| `serve` | Start HTTP/WebSocket server and embedded UI | `--port` (default: 8080), `--db` (default: ./trajectories.db), `--upstream` |
| `record` | Wrap an external process and record API calls | `--port`, `--db`, `-- <command> [args...]` |
| `agent run` | Execute built-in 15-step demo scenario | `--fail` (default: true), `--fixed-version`, `--db` |
| `replay` | Rewind to step K, mutate state, and resume | `--id`, `--step`, `--rewind-to-step`, `--prompt`, `--patch-prompt`, `--db` |
| `export-fixture` | Generate regression test suite or OTel trace | `--id`, `--format` (go/python/pytest/ts/vitest/otel/json), `--output`, `--db` |
| `test` | Verify deterministic JSON test suite | `<suite.json>` |
| `diff` | Compare two trajectories side-by-side | `<id1> <id2> [--db path]` |
| `version` | Print current version | |

---

## REST API Reference

| Endpoint | Method | Description |
|---|---|---|
| `/health` | GET | Health and version status |
| `/v1/chat/completions` | POST | Transparent OpenAI completion proxy |
| `/v1/messages` | POST | Transparent Anthropic message proxy |
| `/api/v1/trajectories` | GET | List recorded trajectories with optional filters |
| `/api/v1/trajectories` | POST | Initialize a new trajectory recording |
| `/api/v1/trajectories/{id}` | GET | Retrieve full trajectory with all steps and tools |
| `/api/v1/trajectories/{id}` | DELETE | Delete trajectory recording |
| `/api/v1/trajectories/{id}/steps` | POST | Append a discrete step |
| `/api/v1/trajectories/{id}/fork` | POST | Fork trajectory at step K |
| `/api/v1/trajectories/{id}/replay` | POST | Rewind to step K, apply mutation, and resume |
| `/api/v1/diff` | GET | Compute step-by-step diff between two trajectories |
| `/api/v1/trajectories/{id}/export` | GET | Export regression test (`format=go\|python\|vitest\|json\|otel`) |
| `/api/v1/demo/run` | POST | Trigger 15-step background demo scenario |
| `/api/v1/mock/tool` | POST | Virtualized tool mock gateway for deterministic replays |
| `/api/v1/ws` | GET | WebSocket live event stream |

---

## Project Structure

```
tracereplay/
├── cmd/
│   └── tracereplay/          # CLI entry point (serve, record, agent, replay, export)
├── pkg/
│   ├── agentdemo/            # Realistic 15-step autonomous agent scenario
│   ├── engine/               # State hashing, time-travel rewind, diffing, diagnostics
│   ├── exporter/             # Go, Python, TypeScript, and OpenInference exporters
│   ├── models/               # Core data structures and contracts
│   ├── proxy/                # Transparent reverse proxy (OpenAI & Anthropic)
│   ├── server/               # REST API, WebSocket hub, and embedded UI routing
│   └── storage/              # Pure Go SQLite storage layer with WAL mode
├── web/                      # React 19 Visual Debugger Studio
│   ├── src/
│   │   ├── components/       # Header, TimelineScrubber, DagView, StepInspector, DiffViewer
│   │   ├── lib/              # API and WebSocket client
│   │   └── types/            # TypeScript definitions
│   └── package.json
├── docs/                     # Specifications, documentation, and media assets
│   ├── images/               # High-fidelity live UI screenshots, GIF, and branding mark
│   ├── ARCHITECTURE.md
│   ├── API.md
│   └── PLAN.md
├── scripts/                  # Automated branding mark and asset generation
├── Makefile                  # Build and test orchestration
├── MASTER_PLAN.md            # Architecture and engineering master plan
└── README.md
```

---

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.
