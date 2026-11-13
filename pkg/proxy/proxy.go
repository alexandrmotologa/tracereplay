package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/tracereplay/pkg/engine"
	"github.com/alexandrmotologa/tracereplay/pkg/models"
	"github.com/google/uuid"
)

// Proxy handles transparent recording of LLM completions for OpenAI and Anthropic clients.
type Proxy struct {
	engine          *engine.Engine
	upstreamBaseURL string
	apiKey          string
	activeSessions  map[string]*models.Trajectory
	stepCounters    map[string]int
	mu              sync.Mutex
}

// New creates an OpenAI/Anthropic compatible reverse proxy interceptor.
func New(eng *engine.Engine, upstreamURL string, apiKey string) *Proxy {
	return &Proxy{
		engine:          eng,
		upstreamBaseURL: strings.TrimSuffix(upstreamURL, "/"),
		apiKey:          apiKey,
		activeSessions:  make(map[string]*models.Trajectory),
		stepCounters:    make(map[string]int),
	}
}

// ServeHTTP routes incoming proxy calls.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	if strings.HasSuffix(path, "/chat/completions") {
		p.handleOpenAIChat(w, r)
		return
	}
	if strings.HasSuffix(path, "/messages") {
		p.handleAnthropicMessages(w, r)
		return
	}

	// Default fallback to chat completions
	p.handleOpenAIChat(w, r)
}

func (p *Proxy) getOrCreateTrajectory(sessionID, model, framework string) (*models.Trajectory, int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	traj, exists := p.activeSessions[sessionID]
	if !exists {
		newTraj, err := p.engine.StartRecording(
			fmt.Sprintf("proxy-%s", sessionID),
			"interceptor-client",
			model,
			map[string]any{
				"source":          "proxy_interceptor",
				"agent_framework": framework,
			},
		)
		if err == nil {
			traj = newTraj
			p.activeSessions[sessionID] = traj
			p.stepCounters[sessionID] = 0
		}
	}
	stepIdx := p.stepCounters[sessionID]
	p.stepCounters[sessionID]++
	return traj, stepIdx
}

func (p *Proxy) handleOpenAIChat(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var chatReq struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
		Stream bool `json:"stream"`
		Tools  []struct {
			Type     string `json:"type"`
			Function struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Parameters  any    `json:"parameters"`
			} `json:"function"`
		} `json:"tools"`
	}

	if err := json.Unmarshal(bodyBytes, &chatReq); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	sessionID := r.Header.Get("X-TraceReplay-Session")
	if sessionID == "" {
		sessionID = "session_" + uuid.New().String()[:8]
	}

	traj, stepIdx := p.getOrCreateTrajectory(sessionID, chatReq.Model, "openai")
	startTime := time.Now()

	var lastPrompt string
	for i := len(chatReq.Messages) - 1; i >= 0; i-- {
		if chatReq.Messages[i].Role == "user" {
			if strVal, ok := chatReq.Messages[i].Content.(string); ok {
				lastPrompt = strVal
			} else {
				bytesVal, _ := json.Marshal(chatReq.Messages[i].Content)
				lastPrompt = string(bytesVal)
			}
			break
		}
	}
	if lastPrompt == "" && len(chatReq.Messages) > 0 {
		bytesVal, _ := json.Marshal(chatReq.Messages[len(chatReq.Messages)-1].Content)
		lastPrompt = string(bytesVal)
	}

	var responseText string
	var inputTokens, outputTokens int
	var detectedToolCalls []models.ToolCall

	if p.upstreamBaseURL != "" {
		upstreamURL := p.upstreamBaseURL + "/chat/completions"
		upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, upstreamURL, bytes.NewReader(bodyBytes))
		if err != nil {
			http.Error(w, "Failed to create upstream request", http.StatusInternalServerError)
			return
		}
		upstreamReq.Header = r.Header.Clone()
		if p.apiKey != "" {
			upstreamReq.Header.Set("Authorization", "Bearer "+p.apiKey)
		}

		client := &http.Client{Timeout: 90 * time.Second}
		resp, err := client.Do(upstreamReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("Upstream error: %v", err), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)

		if chatReq.Stream {
			flusher, ok := w.(http.Flusher)
			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				line := scanner.Text()
				fmt.Fprintf(w, "%s\n", line)
				if ok {
					flusher.Flush()
				}
				if strings.HasPrefix(line, "data: ") && !strings.Contains(line, "[DONE]") {
					var chunk struct {
						Choices []struct {
							Delta struct {
								Content string `json:"content"`
							} `json:"delta"`
						} `json:"choices"`
					}
					if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &chunk); err == nil && len(chunk.Choices) > 0 {
						responseText += chunk.Choices[0].Delta.Content
					}
				}
			}
			inputTokens = len(lastPrompt) / 4
			outputTokens = len(responseText) / 4
		} else {
			respBytes, _ := io.ReadAll(resp.Body)
			w.Write(respBytes)

			var upstreamJSON struct {
				Choices []struct {
					Message struct {
						Content   string `json:"content"`
						ToolCalls []struct {
							ID       string `json:"id"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls"`
					} `json:"message"`
				} `json:"choices"`
				Usage struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal(respBytes, &upstreamJSON); err == nil && len(upstreamJSON.Choices) > 0 {
				msg := upstreamJSON.Choices[0].Message
				responseText = msg.Content
				inputTokens = upstreamJSON.Usage.PromptTokens
				outputTokens = upstreamJSON.Usage.CompletionTokens

				for _, tc := range msg.ToolCalls {
					detectedToolCalls = append(detectedToolCalls, models.ToolCall{
						ToolName:   tc.Function.Name,
						Arguments:  json.RawMessage(tc.Function.Arguments),
						Output:     "Tool scheduled for execution",
						DurationMS: 5,
					})
				}
			}
		}
	} else {
		// Mock Standalone Mode
		responseText = fmt.Sprintf("TraceReplay recorded response for prompt: %s", lastPrompt)
		inputTokens = len(lastPrompt) / 4
		if inputTokens < 12 {
			inputTokens = 12
		}
		outputTokens = len(responseText) / 4
		if outputTokens < 18 {
			outputTokens = 18
		}

		if chatReq.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			flusher, _ := w.(http.Flusher)

			chunkID := "chatcmpl-" + uuid.New().String()
			words := strings.Fields(responseText)
			for i, word := range words {
				chunk := map[string]any{
					"id":      chunkID,
					"object":  "chat.completion.chunk",
					"created": time.Now().Unix(),
					"model":   chatReq.Model,
					"choices": []map[string]any{
						{
							"index": 0,
							"delta": map[string]any{
								"content": word + " ",
							},
						},
					},
				}
				b, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", string(b))
				if flusher != nil {
					flusher.Flush()
				}
				if i < len(words)-1 {
					time.Sleep(10 * time.Millisecond)
				}
			}
			fmt.Fprintf(w, "data: [DONE]\n\n")
			if flusher != nil {
				flusher.Flush()
			}
		} else {
			mockResp := map[string]any{
				"id":      "chatcmpl-" + uuid.New().String(),
				"object":  "chat.completion",
				"created": time.Now().Unix(),
				"model":   chatReq.Model,
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role":    "assistant",
							"content": responseText,
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]any{
					"prompt_tokens":     inputTokens,
					"completion_tokens": outputTokens,
					"total_tokens":      inputTokens + outputTokens,
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResp)
		}
	}

	durationMS := time.Since(startTime).Milliseconds()

	if traj != nil {
		stepType := models.StepTypeLLMGeneration
		if len(detectedToolCalls) > 0 {
			stepType = models.StepTypeToolCall
		}

		step := &models.Step{
			TrajectoryID: traj.ID,
			StepIndex:    stepIdx,
			StepType:     stepType,
			Prompt:       lastPrompt,
			Response:     responseText,
			DurationMS:   durationMS,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			Status:       models.StepStatusSuccess,
			ToolCalls:    detectedToolCalls,
		}
		_ = p.engine.RecordStep(step)
	}
}

func (p *Proxy) handleAnthropicMessages(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var anthropicReq struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
		System string `json:"system"`
		Stream bool   `json:"stream"`
	}

	if err := json.Unmarshal(bodyBytes, &anthropicReq); err != nil {
		http.Error(w, "Invalid Anthropic JSON payload", http.StatusBadRequest)
		return
	}

	sessionID := r.Header.Get("X-TraceReplay-Session")
	if sessionID == "" {
		sessionID = "anthropic_" + uuid.New().String()[:8]
	}

	traj, stepIdx := p.getOrCreateTrajectory(sessionID, anthropicReq.Model, "anthropic")
	startTime := time.Now()

	var lastPrompt string
	for i := len(anthropicReq.Messages) - 1; i >= 0; i-- {
		if anthropicReq.Messages[i].Role == "user" {
			if strVal, ok := anthropicReq.Messages[i].Content.(string); ok {
				lastPrompt = strVal
			} else {
				bytesVal, _ := json.Marshal(anthropicReq.Messages[i].Content)
				lastPrompt = string(bytesVal)
			}
			break
		}
	}

	responseText := fmt.Sprintf("TraceReplay recorded Anthropic response for: %s", lastPrompt)
	inputTokens := len(lastPrompt)/4 + 20
	outputTokens := len(responseText)/4 + 15

	mockResp := map[string]any{
		"id":      "msg_" + uuid.New().String(),
		"type":    "message",
		"role":    "assistant",
		"model":   anthropicReq.Model,
		"content": []map[string]any{{"type": "text", "text": responseText}},
		"usage": map[string]any{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mockResp)

	durationMS := time.Since(startTime).Milliseconds()

	if traj != nil {
		step := &models.Step{
			TrajectoryID: traj.ID,
			StepIndex:    stepIdx,
			StepType:     models.StepTypeLLMGeneration,
			Prompt:       lastPrompt,
			Response:     responseText,
			DurationMS:   durationMS,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			Status:       models.StepStatusSuccess,
		}
		_ = p.engine.RecordStep(step)
	}
}
