# TraceReplay

> Deterministic Agent Trajectory Recorder, Time-Travel Debugger, and Regression Test Generator

[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat&logo=go)](https://golang.org)
[![React Version](https://img.shields.io/badge/React-19-61DAFB?style=flat&logo=react)](https://react.dev)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Architecture](https://img.shields.io/badge/Single%20Binary-<20MB-emerald.svg)](docs/ARCHITECTURE.md)

---

## Overview

TraceReplay is a local-first trajectory flight recorder and deterministic time-travel debugger for autonomous AI agents.

When an autonomous multi-step agent fails at Step 12 of a 15-step execution in production or staging, reproducing the defect is traditionally difficult. Because LLMs are non-deterministic and external tools produce stateful mutations, running the agent again from Step 0 rarely reproduces the exact failure and wastes substantial API tokens.

TraceReplay solves this by capturing every step (prompts, tool calls, API completions, memory state variables, and execution latencies) into an immutable SQLite flight recording. Engineers can rewind directly to Step 8, modify a prompt or tool argument, and resume execution forward while earlier steps (0 to 7) are resolved instantly from the recorded mock cache with zero network calls and zero token consumption.

---

## Key Capabilities

- **Zero-Code Transparent Proxy**: Intercepts OpenAI, Anthropic, and local Ollama API calls without requiring SDK code modifications. Set `OPENAI_BASE_URL=http://localhost:8080/v1` or use `tracereplay record -- python agent.py`.
- **Deterministic State Fingerprints**: Calculates SHA-256 state hashes across prompt history, canonicalized tool calls, and memory variables to detect non-deterministic trajectory drift.
- **Time-Travel Rewind & Forking**: Rewinds execution to step `K`. Pre-fork steps are served from virtual cache (< 5ms latency). Fork branches allow comparing original failed runs against repaired trajectories.
- **1-Click CI Regression Synthesis**: Auto-generates standalone, offline unit tests for Go (`testing`), Python (`pytest`), and deterministic JSON specifications without live model dependencies.
- **High-Performance Single Binary**: Pure Go backend using embedded SQLite and embedded React 19 studio. Binary size is under 15MB with sub-5ms proxy overhead and zero external CGO dependencies.
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
       |  - Regression Test Exporter (Go, Python, JSON)         |
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
  --id trj_830c39ac-f786-492a-a957-06f05f405365 \
  --step 8 \
  --prompt "Synthesize patch parameters with explicit target version 12"
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
./bin/tracereplay export --id trj_830c39ac --format go --output ./tests/agent_test.go

# Export Python pytest suite
./bin/tracereplay export --id trj_830c39ac --format pytest --output ./tests/test_agent.py

# Export portable JSON suite and verify determinism
./bin/tracereplay export --id trj_830c39ac --format json --output ./tests/suite.json
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
| `export` | Generate regression test suite | `--id`, `--format` (go/python/pytest/json), `--output`, `--db` |
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
| `/api/v1/trajectories/{id}/export` | GET | Export regression test (`format=go\|python\|json`) |
| `/api/v1/demo/run` | POST | Trigger 15-step background demo scenario |
| `/api/v1/ws` | GET | WebSocket live event stream |

---

## Project Structure

```
tracereplay/
├── cmd/
│   └── tracereplay/          # CLI entry point (serve, record, agent, replay, export)
├── pkg/
│   ├── agentdemo/            # Realistic 15-step autonomous agent scenario
│   ├── engine/               # State hashing, time-travel rewind, diffing
│   ├── exporter/             # Go test, Python pytest, and JSON test generators
│   ├── models/               # Core data structures and contracts
│   ├── proxy/                # Transparent reverse proxy (OpenAI & Anthropic)
│   ├── server/               # REST API, WebSocket hub, and embedded UI routing
│   └── storage/              # Pure Go SQLite storage layer with WAL mode
├── web/                      # React 19 Visual Debugger Studio
│   ├── src/
│   │   ├── components/       # Header, TimelineScrubber, StepInspector, DiffViewer
│   │   ├── lib/              # API and WebSocket client
│   │   └── types/            # TypeScript definitions
│   └── package.json
├── docs/                     # Specifications and guides
│   ├── ARCHITECTURE.md
│   ├── API.md
│   └── PLAN.md
├── tests/                    # Integration and demo test scripts
├── Makefile                  # Build and test orchestration
├── MASTER_PLAN.md            # Architecture and engineering master plan
└── README.md
```

---

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.
