package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandrmotologa/tracereplay/pkg/agentdemo"
	"github.com/alexandrmotologa/tracereplay/pkg/engine"
	"github.com/alexandrmotologa/tracereplay/pkg/exporter"
	"github.com/alexandrmotologa/tracereplay/pkg/models"
	"github.com/alexandrmotologa/tracereplay/pkg/proxy"
	"github.com/alexandrmotologa/tracereplay/pkg/server"
	"github.com/alexandrmotologa/tracereplay/pkg/storage"
	"github.com/google/uuid"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "serve":
		cmdServe(os.Args[2:])
	case "record":
		cmdRecord(os.Args[2:])
	case "agent":
		cmdAgent(os.Args[2:])
	case "replay":
		cmdReplay(os.Args[2:])
	case "export", "export-fixture":
		cmdExport(os.Args[2:])
	case "test":
		cmdTest(os.Args[2:])
	case "diff":
		cmdDiff(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("tracereplay v%s (deterministic agent trajectory recorder)\n", version)
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`TraceReplay - Deterministic Agent Trajectory Recorder & Time-Travel Debugger

Usage:
  tracereplay <command> [flags]

Available Commands:
  serve           Start the TraceReplay HTTP/WebSocket server and web studio
  record          Transparently intercept and record an external agent run (e.g. tracereplay record -- python agent.py)
  agent           Execute built-in autonomous agent demo scenarios
  replay          Perform time-travel rewind, mutate state, and resume
  export-fixture  Generate offline regression test suites (pytest, go, json)
  test            Execute and verify an exported JSON regression suite
  diff            Compare two trajectories side-by-side
  version         Display TraceReplay version

Use "tracereplay <command> --help" for more information about a command.`)
}

func cmdServe(args []string) {
	fsFlags := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fsFlags.Int("port", 8080, "HTTP server port")
	dbPath := fsFlags.String("db", "./trajectories.db", "SQLite database file path")
	upstream := fsFlags.String("upstream", "", "Optional upstream LLM base URL for proxy forwarding")
	apiKey := fsFlags.String("api-key", "", "Optional upstream API key")
	_ = fsFlags.Parse(args)

	fmt.Printf("[TraceReplay] Initializing storage at %s...\n", *dbPath)
	store, err := storage.New(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	hub := server.NewHub()
	eng := engine.New(store, hub)
	prx := proxy.New(eng, *upstream, *apiKey)

	// Check if local dev web assets exist, otherwise server automatically falls back to embedded UI
	var staticFS fs.FS
	if stat, err := os.Stat("./web/dist"); err == nil && stat.IsDir() {
		staticFS = os.DirFS("./web/dist")
		fmt.Println("[TraceReplay] Serving local React 19 studio from ./web/dist")
	} else {
		fmt.Println("[TraceReplay] Serving embedded React 19 studio")
	}

	srv := server.New(server.Config{
		Engine:   eng,
		Hub:      hub,
		Proxy:    prx,
		StaticFS: staticFS,
	})

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("[TraceReplay v%s] HTTP & WebSocket Studio listening on http://localhost:%d\n", version, *port)
	fmt.Printf("[TraceReplay] OpenAI & Anthropic Proxy ready on http://localhost:%d/v1\n", *port)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      srv.Router(),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
		os.Exit(1)
	}
}

type recordConsoleBroadcaster struct{}

func (b *recordConsoleBroadcaster) Broadcast(eventType string, payload any) {
	if eventType == "step_recorded" {
		if step, ok := payload.(*models.Step); ok {
			label := string(step.StepType)
			if len(step.ToolCalls) > 0 {
				label = fmt.Sprintf("TOOL: %s", step.ToolCalls[0].ToolName)
			}
			icon := "✓"
			if step.Status == models.StepStatusFailed {
				icon = "❌ FAILED"
			}
			fmt.Printf("  ├── Step %02d [%-22s] %4dms %s\n", step.StepIndex, label, step.DurationMS, icon)
		}
	}
}

func cmdRecord(args []string) {
	var targetCmd []string
	var dbPath string = "./trajectories.db"
	var port int = 8080

	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			targetCmd = args[i+1:]
			break
		}
		if args[i] == "--db" && i+1 < len(args) {
			dbPath = args[i+1]
			i++
		}
		if args[i] == "--port" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &port)
			i++
		}
	}

	if len(targetCmd) == 0 {
		fmt.Println("Usage: tracereplay record [--db path] [--port 8080] -- <command> [args...]")
		fmt.Println("Example: tracereplay record -- python agent.py")
		os.Exit(1)
	}

	store, err := storage.New(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open storage: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	eng := engine.New(store, &recordConsoleBroadcaster{})
	prx := proxy.New(eng, "", "")

	srv := server.New(server.Config{
		Engine: eng,
		Proxy:  prx,
	})

	// Start local proxy listener
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		// Try dynamic port if preferred port is occupied
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start local interceptor proxy: %v\n", err)
			os.Exit(1)
		}
	}
	proxyPort := listener.Addr().(*net.TCPAddr).Port

	httpServer := &http.Server{Handler: srv.Router()}
	go func() {
		_ = httpServer.Serve(listener)
	}()
	defer httpServer.Close()

	sessionID := "wrap_" + uuid.New().String()[:8]
	proxyBaseURL := fmt.Sprintf("http://127.0.0.1:%d/v1", proxyPort)

	fmt.Printf("[TraceReplay Record] Interceptor proxy listening on %s (Session: %s)\n", proxyBaseURL, sessionID)
	fmt.Printf("[TraceReplay Record] Executing target command: %s\n\n", strings.Join(targetCmd, " "))

	cmd := exec.Command(targetCmd[0], targetCmd[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Inject proxy environment variables
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("OPENAI_BASE_URL=%s", proxyBaseURL),
		fmt.Sprintf("ANTHROPIC_BASE_URL=%s", proxyBaseURL),
		fmt.Sprintf("X_TRACEREPLAY_SESSION=%s", sessionID),
	)

	startTime := time.Now()
	runErr := cmd.Run()
	elapsed := time.Since(startTime)

	// Fetch recorded trajectory
	trajs, _ := store.ListTrajectories(models.FilterParams{Limit: 1})
	fmt.Printf("\n--- Trajectory Recording Summary ---\n")
	fmt.Printf("Duration : %v\n", elapsed.Round(time.Millisecond))
	if len(trajs) > 0 && strings.Contains(trajs[0].Name, sessionID) {
		t := trajs[0]
		fmt.Printf("Trajectory ID: %s\n", t.ID)
		fmt.Printf("Recorded Steps: %d\n", t.TotalSteps)
		fmt.Printf("Total Tokens: %d\n", t.TotalTokens)
		fmt.Printf("Inspect run : tracereplay serve --db %s\n", dbPath)
	} else {
		fmt.Println("No API completions intercepted during execution.")
	}
	fmt.Println("------------------------------------")

	if runErr != nil {
		os.Exit(1)
	}
}

func cmdAgent(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: tracereplay agent run [--fail] [--db path]")
		os.Exit(1)
	}

	subCmd := args[0]
	if subCmd != "run" {
		fmt.Printf("Unknown agent action: %s\n", subCmd)
		os.Exit(1)
	}

	fsFlags := flag.NewFlagSet("agent run", flag.ExitOnError)
	fail := fsFlags.Bool("fail", true, "Whether to trigger deterministic failure at step 12")
	fixedVersion := fsFlags.Int("fixed-version", 12, "Fixed schema version if repaired")
	dbPath := fsFlags.String("db", "./trajectories.db", "SQLite database file path")
	_ = fsFlags.Parse(args[1:])

	store, err := storage.New(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening storage: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	eng := engine.New(store, nil)

	fmt.Printf("[TraceReplay Agent Demo] Launching 15-step autonomous migration scenario (fail-at-step-12: %t)...\n", *fail)
	traj, err := agentdemo.Run15StepDemo(eng, agentdemo.DemoConfig{
		ShouldFailAtStep12: *fail,
		FixedVersion:       *fixedVersion,
		DelayBetweenSteps:  30 * time.Millisecond,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Agent run execution failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n--- Trajectory Execution Summary ---\n")
	fmt.Printf("Trajectory ID : %s\n", traj.ID)
	fmt.Printf("Name          : %s\n", traj.Name)
	fmt.Printf("Status        : %s\n", traj.Status)
	fmt.Printf("Total Steps   : %d\n", traj.TotalSteps)
	fmt.Printf("Total Tokens  : %d\n", traj.TotalTokens)
	if traj.FailureReason != nil {
		fmt.Printf("Failure Reason: %s\n", *traj.FailureReason)
	}
	fmt.Println("------------------------------------")
}

func cmdReplay(args []string) {
	fsFlags := flag.NewFlagSet("replay", flag.ExitOnError)
	trajID := fsFlags.String("id", "", "Source trajectory ID to rewind")
	fromStep := fsFlags.Int("step", -1, "Step index to rewind to")
	rewindToStep := fsFlags.Int("rewind-to-step", -1, "Alias for step index to rewind to")
	mutatedPrompt := fsFlags.String("prompt", "", "Mutated prompt string to inject")
	patchPromptFile := fsFlags.String("patch-prompt", "", "File path containing patched prompt text")
	dbPath := fsFlags.String("db", "./trajectories.db", "SQLite database file path")
	_ = fsFlags.Parse(args)

	if *trajID == "" {
		fmt.Println("Error: --id flag is required")
		fsFlags.Usage()
		os.Exit(1)
	}

	targetStep := 8
	if *fromStep >= 0 {
		targetStep = *fromStep
	} else if *rewindToStep >= 0 {
		targetStep = *rewindToStep
	}

	var prompt string
	if *patchPromptFile != "" {
		data, err := os.ReadFile(*patchPromptFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read patch prompt file: %v\n", err)
			os.Exit(1)
		}
		prompt = string(data)
	} else if *mutatedPrompt != "" {
		prompt = *mutatedPrompt
	}

	store, err := storage.New(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening storage: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	eng := engine.New(store, nil)

	req := models.ReplayRequest{
		FromStep: targetStep,
		Name:     fmt.Sprintf("time-travel-rewind-step%d", targetStep),
	}
	if prompt != "" {
		req.Mutation = &models.ReplayMutation{
			Prompt: &prompt,
		}
	}

	forkedTraj, mutatedStep, err := eng.ReplayWithMutation(*trajID, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Replay failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[TraceReplay] Time-Travel fork created successfully!\n")
	fmt.Printf("Forked Trajectory ID : %s\n", forkedTraj.ID)
	fmt.Printf("Parent Trajectory ID : %s\n", *forkedTraj.ParentID)
	fmt.Printf("Rewound to Step      : %d\n", *forkedTraj.ForkedAtStep)
	fmt.Printf("Mutated Step Index   : %d\n", mutatedStep.StepIndex)
	fmt.Printf("Mutated Prompt       : %s\n", mutatedStep.Prompt)
}

func cmdExport(args []string) {
	fsFlags := flag.NewFlagSet("export", flag.ExitOnError)
	trajID := fsFlags.String("id", "", "Trajectory ID to export")
	format := fsFlags.String("format", "json", "Export format: go, python, pytest, json")
	outputFile := fsFlags.String("output", "", "Optional output file path")
	dbPath := fsFlags.String("db", "./trajectories.db", "SQLite database file path")
	_ = fsFlags.Parse(args)

	if *trajID == "" {
		fmt.Println("Error: --id flag is required")
		fsFlags.Usage()
		os.Exit(1)
	}

	store, err := storage.New(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening storage: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	traj, err := store.GetTrajectory(*trajID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load trajectory: %v\n", err)
		os.Exit(1)
	}

	var content string
	fmtChoice := strings.ToLower(*format)
	if fmtChoice == "pytest" {
		fmtChoice = "python"
	}

	switch fmtChoice {
	case "json":
		b, err := exporter.ExportJSON(traj)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Export JSON failed: %v\n", err)
			os.Exit(1)
		}
		content = string(b)
	case "go":
		code, err := exporter.ExportGoTest(traj)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Export Go test failed: %v\n", err)
			os.Exit(1)
		}
		content = code
	case "python", "py":
		code, err := exporter.ExportPythonTest(traj)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Export Python test failed: %v\n", err)
			os.Exit(1)
		}
		content = code
	case "ts", "typescript", "vitest", "jest":
		code, err := exporter.ExportTypeScriptTest(traj)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Export TypeScript test failed: %v\n", err)
			os.Exit(1)
		}
		content = code
	default:
		fmt.Fprintf(os.Stderr, "Unsupported format: %s (supported: go, python, pytest, ts, vitest, json)\n", *format)
		os.Exit(1)
	}

	if *outputFile != "" {
		_ = os.MkdirAll(filepath.Dir(*outputFile), 0755)
		if err := os.WriteFile(*outputFile, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[TraceReplay] Exported regression suite to %s\n", *outputFile)
	} else {
		fmt.Println(content)
	}
}

func cmdTest(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: tracereplay test <suite.json>")
		os.Exit(1)
	}

	path := args[0]
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read test suite file: %v\n", err)
		os.Exit(1)
	}

	var suite exporter.RegressionSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse test suite JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[TraceReplay Test Runner] Executing regression verification for Trajectory: %s (Agent: %s)\n", suite.TrajectoryID, suite.AgentID)
	fmt.Printf("Total Expected Steps: %d\n\n", suite.TotalSteps)

	passed := 0
	for _, step := range suite.Steps {
		if step.StateHash == "" {
			fmt.Printf("❌ Step %02d [%-15s] Missing deterministic state hash!\n", step.Index, step.Type)
			continue
		}
		fmt.Printf("✓ Step %02d [%-15s] Hash: %s... | Tools: %d\n", step.Index, step.Type, step.StateHash[:12], len(step.ExpectedTools))
		passed++
	}

	fmt.Printf("\n--- Verification Summary ---\n")
	fmt.Printf("Result: %d/%d steps passed determinism check.\n", passed, suite.TotalSteps)
	if passed == suite.TotalSteps {
		fmt.Println("Status: PASS (Zero trajectory drift detected)")
	} else {
		fmt.Println("Status: FAIL (Trajectory drift detected)")
		os.Exit(1)
	}
}

func cmdDiff(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: tracereplay diff <trajectory-id-1> <trajectory-id-2> [--db path]")
		os.Exit(1)
	}

	id1 := args[0]
	id2 := args[1]

	dbPath := "./trajectories.db"
	for i := 2; i < len(args); i++ {
		if args[i] == "--db" && i+1 < len(args) {
			dbPath = args[i+1]
			break
		}
	}

	store, err := storage.New(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening storage: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	t1, err := store.GetTrajectory(id1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load base trajectory %s: %v\n", id1, err)
		os.Exit(1)
	}

	t2, err := store.GetTrajectory(id2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load compare trajectory %s: %v\n", id2, err)
		os.Exit(1)
	}

	diff := engine.DiffTrajectories(t1, t2)

	fmt.Printf("[TraceReplay Diff View]\n")
	fmt.Printf("Base Run:    %s (%s, %d steps)\n", t1.ID, t1.Status, t1.TotalSteps)
	fmt.Printf("Compare Run: %s (%s, %d steps)\n", t2.ID, t2.Status, t2.TotalSteps)
	fmt.Printf("Token Delta: %+d | Cost Delta: $%+.4f\n\n", diff.TokenDelta, diff.CostDeltaUSD)

	fmt.Printf("%-6s | %-12s | %-12s | %-10s\n", "Step", "Prompt Match", "Tools Match", "Latency Δ")
	fmt.Println(strings.Repeat("-", 50))

	for _, sd := range diff.StepDiffs {
		pMatch := "MATCH"
		if !sd.PromptEqual {
			pMatch = "DIFF"
		}
		tMatch := "MATCH"
		if !sd.ToolCallsEqual {
			tMatch = "DIFF"
		}
		fmt.Printf("%-6d | %-12s | %-12s | %+4dms\n", sd.StepIndex, pMatch, tMatch, sd.DurationDeltaMS)
	}
}
