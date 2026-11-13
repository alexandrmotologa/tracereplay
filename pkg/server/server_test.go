package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexandrmotologa/tracereplay/pkg/agentdemo"
	"github.com/alexandrmotologa/tracereplay/pkg/engine"
	"github.com/alexandrmotologa/tracereplay/pkg/models"
	"github.com/alexandrmotologa/tracereplay/pkg/storage"
)

func setupTestServer(t *testing.T) (*Server, *storage.Storage, *engine.Engine) {
	store, err := storage.New(":memory:")
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}

	hub := NewHub()
	eng := engine.New(store, hub)
	srv := New(Config{
		Engine: eng,
		Hub:    hub,
	})

	return srv, store, eng
}

func TestServerEndpoints(t *testing.T) {
	srv, store, eng := setupTestServer(t)
	defer store.Close()

	// 1. Health
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("health returned status %d", w.Code)
	}

	// 2. Run Demo Agent
	demoTraj, err := agentdemo.Run15StepDemo(eng, agentdemo.DemoConfig{
		ShouldFailAtStep12: true,
	})
	if err != nil {
		t.Fatalf("demo run failed: %v", err)
	}

	// 3. GET /api/v1/trajectories
	req = httptest.NewRequest(http.MethodGet, "/api/v1/trajectories", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("list trajectories returned %d", w.Code)
	}

	var trajs []models.Trajectory
	json.Unmarshal(w.Body.Bytes(), &trajs)
	if len(trajs) != 1 {
		t.Fatalf("expected 1 trajectory, got %d", len(trajs))
	}

	// 4. GET /api/v1/trajectories/{id}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/trajectories/"+demoTraj.ID, nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get trajectory returned %d", w.Code)
	}

	var loaded models.Trajectory
	json.Unmarshal(w.Body.Bytes(), &loaded)
	if len(loaded.Steps) != 13 {
		t.Errorf("expected 13 steps in failed trajectory, got %d", len(loaded.Steps))
	}

	// 5. POST /api/v1/trajectories/{id}/replay (Time-Travel from step 8)
	newPrompt := "Synthesize patch parameters with explicit v12 version."
	replayPayload := map[string]any{
		"from_step": 8,
		"name":      "repaired-api-replay",
		"mutation": map[string]any{
			"prompt": newPrompt,
		},
	}
	bodyBytes, _ := json.Marshal(replayPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/trajectories/"+demoTraj.ID+"/replay", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("replay returned status %d, body: %s", w.Code, w.Body.String())
	}

	var replayResp struct {
		Trajectory models.Trajectory `json:"trajectory"`
	}
	json.Unmarshal(w.Body.Bytes(), &replayResp)
	repairedID := replayResp.Trajectory.ID

	// 6. GET /api/v1/diff?base=...&compare=...
	req = httptest.NewRequest(http.MethodGet, "/api/v1/diff?base="+demoTraj.ID+"&compare="+repairedID, nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("diff returned %d: %s", w.Code, w.Body.String())
	}

	// 7. GET /api/v1/trajectories/{id}/export?format=json
	req = httptest.NewRequest(http.MethodGet, "/api/v1/trajectories/"+demoTraj.ID+"/export?format=json", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "auth-migration-bot") {
		t.Fatalf("export json failed: %s", w.Body.String())
	}

	// 8. GET /api/v1/trajectories/{id}/export?format=go
	req = httptest.NewRequest(http.MethodGet, "/api/v1/trajectories/"+demoTraj.ID+"/export?format=go", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "package regression_test") {
		t.Fatalf("export go test failed: %s", w.Body.String())
	}
}
