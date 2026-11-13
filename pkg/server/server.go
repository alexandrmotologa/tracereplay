package server

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexandrmotologa/tracereplay/pkg/agentdemo"
	"github.com/alexandrmotologa/tracereplay/pkg/engine"
	"github.com/alexandrmotologa/tracereplay/pkg/exporter"
	"github.com/alexandrmotologa/tracereplay/pkg/models"
	"github.com/alexandrmotologa/tracereplay/pkg/proxy"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Server configures the HTTP server for TraceReplay.
type Server struct {
	router  *chi.Mux
	engine  *engine.Engine
	hub     *Hub
	proxy   *proxy.Proxy
	staticFS fs.FS
}

// Config defines server parameters.
type Config struct {
	Engine   *engine.Engine
	Hub      *Hub
	Proxy    *proxy.Proxy
	StaticFS fs.FS
}

// New creates an initialized Server instance.
func New(cfg Config) *Server {
	staticFS := cfg.StaticFS
	if staticFS == nil {
		staticFS = GetEmbeddedWebFS()
	}

	s := &Server{
		router:   chi.NewRouter(),
		engine:   cfg.Engine,
		hub:      cfg.Hub,
		proxy:    cfg.Proxy,
		staticFS: staticFS,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	r := s.router

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-TraceReplay-Session"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","version":"1.0.0"}`))
	})

	// WebSocket live streaming
	if s.hub != nil {
		r.Get("/api/v1/ws", s.hub.HandleWS)
	}

	// OpenAI & Anthropic-compatible Proxy
	if s.proxy != nil {
		r.Post("/v1/chat/completions", s.proxy.ServeHTTP)
		r.Post("/v1/messages", s.proxy.ServeHTTP)
		r.Post("/api/v1/proxy/v1/chat/completions", s.proxy.ServeHTTP)
		r.Post("/api/v1/proxy/v1/messages", s.proxy.ServeHTTP)
	}

	// REST API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Trajectories
		r.Get("/trajectories", s.handleListTrajectories)
		r.Post("/trajectories", s.handleCreateTrajectory)
		r.Get("/trajectories/{id}", s.handleGetTrajectory)
		r.Delete("/trajectories/{id}", s.handleDeleteTrajectory)
		r.Post("/trajectories/{id}/steps", s.handleCreateStep)
		r.Post("/trajectories/{id}/complete", s.handleCompleteTrajectory)

		// Time-Travel & Branching
		r.Post("/trajectories/{id}/fork", s.handleForkTrajectory)
		r.Post("/trajectories/{id}/replay", s.handleReplayTrajectory)

		// Diffing
		r.Get("/diff", s.handleDiffTrajectories)

		// Regression Exporter
		r.Get("/trajectories/{id}/export", s.handleExportRegression)

		// Demo Agent Scenario
		r.Post("/demo/run", s.handleRunDemo)
	})

	// Serve Static Frontend if embedded or provided
	if s.staticFS != nil {
		fileServer := http.FileServer(http.FS(s.staticFS))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			f, err := s.staticFS.Open(path)
			if err != nil {
				// Fallback to index.html for SPA router
				indexFile, err := s.staticFS.Open("index.html")
				if err != nil {
					http.NotFound(w, r)
					return
				}
				defer indexFile.Close()
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				io.Copy(w, indexFile)
				return
			}
			f.Close()
			fileServer.ServeHTTP(w, r)
		})
	}
}

// Router returns the chi router for mounting or testing.
func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) handleListTrajectories(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	agentID := r.URL.Query().Get("agent_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	params := models.FilterParams{
		Status:  status,
		AgentID: agentID,
		Limit:   limit,
		Offset:  offset,
	}

	trajs, err := s.engine.GetStorage().ListTrajectories(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, trajs)
}

func (s *Server) handleCreateTrajectory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string         `json:"name"`
		AgentID  string         `json:"agent_id"`
		Model    string         `json:"model"`
		Metadata map[string]any `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	traj, err := s.engine.StartRecording(req.Name, req.AgentID, req.Model, req.Metadata)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, traj)
}

func (s *Server) handleGetTrajectory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	traj, err := s.engine.GetStorage().GetTrajectory(id)
	if err != nil {
		http.Error(w, "Trajectory not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, traj)
}

func (s *Server) handleDeleteTrajectory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.engine.GetStorage().DeleteTrajectory(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateStep(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var step models.Step
	if err := json.NewDecoder(r.Body).Decode(&step); err != nil {
		http.Error(w, "Invalid step body", http.StatusBadRequest)
		return
	}

	step.TrajectoryID = id
	if err := s.engine.RecordStep(&step); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, step)
}

func (s *Server) handleCompleteTrajectory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Status        models.TrajectoryStatus `json:"status"`
		FailureReason *string                 `json:"failure_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		req.Status = models.StatusCompleted
	}

	traj, err := s.engine.CompleteTrajectory(id, req.Status, req.FailureReason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, traj)
}

func (s *Server) handleForkTrajectory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req models.ForkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid fork body", http.StatusBadRequest)
		return
	}

	forked, err := s.engine.ForkTrajectory(id, req.ForkAtStep, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, forked)
}

func (s *Server) handleReplayTrajectory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req models.ReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid replay body", http.StatusBadRequest)
		return
	}

	forked, mutatedStep, err := s.engine.ReplayWithMutation(id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := map[string]any{
		"trajectory":   forked,
		"mutated_step": mutatedStep,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDiffTrajectories(w http.ResponseWriter, r *http.Request) {
	baseID := r.URL.Query().Get("base")
	compareID := r.URL.Query().Get("compare")

	if baseID == "" || compareID == "" {
		http.Error(w, "Query parameters 'base' and 'compare' are required", http.StatusBadRequest)
		return
	}

	base, err := s.engine.GetStorage().GetTrajectory(baseID)
	if err != nil {
		http.Error(w, "Base trajectory not found: "+err.Error(), http.StatusNotFound)
		return
	}

	compare, err := s.engine.GetStorage().GetTrajectory(compareID)
	if err != nil {
		http.Error(w, "Compare trajectory not found: "+err.Error(), http.StatusNotFound)
		return
	}

	diff := engine.DiffTrajectories(base, compare)
	writeJSON(w, http.StatusOK, diff)
}

func (s *Server) handleExportRegression(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "json"
	}

	traj, err := s.engine.GetStorage().GetTrajectory(id)
	if err != nil {
		http.Error(w, "Trajectory not found", http.StatusNotFound)
		return
	}

	switch format {
	case "json":
		bytes, err := exporter.ExportJSON(traj)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="trajectory_regression.json"`)
		w.Write(bytes)

	case "go":
		code, err := exporter.ExportGoTest(traj)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="trajectory_regression_test.go"`)
		w.Write([]byte(code))

	case "python", "py":
		code, err := exporter.ExportPythonTest(traj)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="test_trajectory_regression.py"`)
		w.Write([]byte(code))

	default:
		http.Error(w, "Unsupported export format (supported: json, go, python)", http.StatusBadRequest)
	}
}

func (s *Server) handleRunDemo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ShouldFailAtStep12 bool `json:"should_fail"`
		FixedVersion       int  `json:"fixed_version"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Default to failing at step 12 to demonstrate the problem
	if !strings.Contains(r.URL.RawQuery, "should_fail") && req.FixedVersion == 0 {
		req.ShouldFailAtStep12 = true
	}

	go func() {
		_, _ = agentdemo.Run15StepDemo(s.engine, agentdemo.DemoConfig{
			ShouldFailAtStep12: req.ShouldFailAtStep12,
			FixedVersion:       req.FixedVersion,
			DelayBetweenSteps:  60 * time.Millisecond,
		})
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "Demo agent trajectory started in background",
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
