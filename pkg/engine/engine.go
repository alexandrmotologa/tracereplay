package engine

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/alexandrmotologa/tracereplay/pkg/models"
	"github.com/alexandrmotologa/tracereplay/pkg/storage"
	"github.com/google/uuid"
)

// EventBroadcaster allows notifying connected clients about live trajectory progress.
type EventBroadcaster interface {
	Broadcast(eventType string, payload any)
}

// Engine coordinates trajectory recording, deterministic hashing, and time-travel replay.
type Engine struct {
	storage     *storage.Storage
	broadcaster EventBroadcaster
	mu          sync.RWMutex
}

// New creates an Engine instance.
func New(s *storage.Storage, b EventBroadcaster) *Engine {
	return &Engine{
		storage:     s,
		broadcaster: b,
	}
}

// StartRecording initializes a new trajectory session.
func (e *Engine) StartRecording(name, agentID, model string, metadata map[string]any) (*models.Trajectory, error) {
	traj := &models.Trajectory{
		ID:        "trj_" + uuid.New().String(),
		Name:      name,
		AgentID:   agentID,
		Model:     model,
		Status:    models.StatusRunning,
		StartedAt: time.Now().UTC(),
		Metadata:  metadata,
	}

	if err := e.storage.CreateTrajectory(traj); err != nil {
		return nil, fmt.Errorf("failed to create trajectory: %w", err)
	}

	if e.broadcaster != nil {
		e.broadcaster.Broadcast("trajectory_started", traj)
	}

	return traj, nil
}

// RecordStep appends a validated step with computed state hash to the trajectory.
func (e *Engine) RecordStep(step *models.Step) error {
	if step.TrajectoryID == "" {
		return fmt.Errorf("trajectory_id is required")
	}

	// Compute state hash if not already provided
	if step.StateHash == "" {
		step.StateHash = ComputeStateHash(step.TrajectoryID, step.StepIndex, step.Prompt, step.ToolCalls, step.MemorySnapshot)
	}

	if err := e.storage.CreateStep(step); err != nil {
		return fmt.Errorf("failed to persist step: %w", err)
	}

	// Update trajectory metrics
	traj, err := e.storage.GetTrajectory(step.TrajectoryID)
	if err == nil {
		traj.TotalSteps++
		traj.TotalTokens += (step.InputTokens + step.OutputTokens)
		// Standard token cost approximation ($3 per 1M input, $15 per 1M output)
		traj.TotalCostUSD += (float64(step.InputTokens)*0.000003 + float64(step.OutputTokens)*0.000015)
		_ = e.storage.UpdateTrajectory(traj)
	}

	if e.broadcaster != nil {
		e.broadcaster.Broadcast("step_recorded", step)
	}

	return nil
}

// CompleteTrajectory finalizes a trajectory run.
func (e *Engine) CompleteTrajectory(id string, status models.TrajectoryStatus, failureReason *string) (*models.Trajectory, error) {
	traj, err := e.storage.GetTrajectory(id)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	traj.CompletedAt = &now
	traj.Status = status
	traj.FailureReason = failureReason

	if err := e.storage.UpdateTrajectory(traj); err != nil {
		return nil, err
	}

	if e.broadcaster != nil {
		e.broadcaster.Broadcast("trajectory_completed", traj)
	}

	return traj, nil
}

// ForkTrajectory creates a branch from an existing trajectory at forkAtStep.
func (e *Engine) ForkTrajectory(parentID string, forkAtStep int, name string) (*models.Trajectory, error) {
	parent, err := e.storage.GetTrajectory(parentID)
	if err != nil {
		return nil, fmt.Errorf("parent trajectory not found: %w", err)
	}

	if forkAtStep < 0 || forkAtStep > len(parent.Steps) {
		return nil, fmt.Errorf("invalid fork_at_step: %d (parent has %d steps)", forkAtStep, len(parent.Steps))
	}

	branchName := name
	if branchName == "" {
		branchName = fmt.Sprintf("%s-fork-step%d", parent.Name, forkAtStep)
	}

	newTraj := &models.Trajectory{
		ID:           "trj_" + uuid.New().String(),
		ParentID:     &parent.ID,
		ForkedAtStep: &forkAtStep,
		Name:         branchName,
		AgentID:      parent.AgentID,
		Model:        parent.Model,
		Status:       models.StatusRunning,
		StartedAt:    time.Now().UTC(),
		Metadata: map[string]any{
			"forked_from":    parent.ID,
			"forked_at_step": forkAtStep,
		},
	}

	if err := e.storage.CreateTrajectory(newTraj); err != nil {
		return nil, fmt.Errorf("failed to create forked trajectory: %w", err)
	}

	// Copy steps 0 .. forkAtStep-1 as replay mock steps
	var totalTokens int
	var totalCost float64
	for i := 0; i < forkAtStep && i < len(parent.Steps); i++ {
		origStep := parent.Steps[i]
		clonedStep := models.Step{
			ID:           "stp_" + uuid.New().String(),
			TrajectoryID: newTraj.ID,
			StepIndex:    origStep.StepIndex,
			StepType:     origStep.StepType,
			Prompt:       origStep.Prompt,
			Response:     origStep.Response,
			StateHash:    origStep.StateHash,
			DurationMS:   0, // Instantaneous virtual replay
			InputTokens:  origStep.InputTokens,
			OutputTokens: origStep.OutputTokens,
			Status:       origStep.Status,
			ErrorMessage: origStep.ErrorMessage,
			IsReplayMock: true,
		}

		// Virtualize tool calls
		for _, tc := range origStep.ToolCalls {
			clonedTC := models.ToolCall{
				ToolName:   tc.ToolName,
				Arguments:  tc.Arguments,
				Output:     tc.Output,
				IsError:    tc.IsError,
				DurationMS: 0,
				IsVirtual:  true,
			}
			clonedStep.ToolCalls = append(clonedStep.ToolCalls, clonedTC)
		}

		if origStep.MemorySnapshot != nil {
			clonedStep.MemorySnapshot = &models.MemorySnapshot{
				Variables:  origStep.MemorySnapshot.Variables,
				Scratchpad: origStep.MemorySnapshot.Scratchpad,
			}
		}

		if err := e.storage.CreateStep(&clonedStep); err != nil {
			return nil, fmt.Errorf("failed to clone step %d: %w", i, err)
		}

		totalTokens += (clonedStep.InputTokens + clonedStep.OutputTokens)
		totalCost += (float64(clonedStep.InputTokens)*0.000003 + float64(clonedStep.OutputTokens)*0.000015)
		newTraj.Steps = append(newTraj.Steps, clonedStep)
	}

	newTraj.TotalSteps = len(newTraj.Steps)
	newTraj.TotalTokens = totalTokens
	newTraj.TotalCostUSD = totalCost
	_ = e.storage.UpdateTrajectory(newTraj)

	if e.broadcaster != nil {
		e.broadcaster.Broadcast("trajectory_forked", newTraj)
	}

	return newTraj, nil
}

// ReplayWithMutation creates a fork at req.FromStep and applies the specified mutations at that step.
func (e *Engine) ReplayWithMutation(sourceID string, req models.ReplayRequest) (*models.Trajectory, *models.Step, error) {
	parent, err := e.storage.GetTrajectory(sourceID)
	if err != nil {
		return nil, nil, fmt.Errorf("source trajectory not found: %w", err)
	}

	if req.FromStep < 0 || req.FromStep >= len(parent.Steps) {
		return nil, nil, fmt.Errorf("from_step %d out of bounds (trajectory has %d steps)", req.FromStep, len(parent.Steps))
	}

	// 1. Fork trajectory up to req.FromStep
	forkedTraj, err := e.ForkTrajectory(sourceID, req.FromStep, req.Name)
	if err != nil {
		return nil, nil, err
	}

	// 2. Hydrate mutated step
	origStep := parent.Steps[req.FromStep]
	mutatedPrompt := origStep.Prompt
	if req.Mutation != nil && req.Mutation.Prompt != nil {
		mutatedPrompt = *req.Mutation.Prompt
	}

	mutatedStep := &models.Step{
		ID:           "stp_" + uuid.New().String(),
		TrajectoryID: forkedTraj.ID,
		StepIndex:    req.FromStep,
		StepType:     origStep.StepType,
		Prompt:       mutatedPrompt,
		Response:     origStep.Response,
		DurationMS:   origStep.DurationMS,
		InputTokens:  origStep.InputTokens,
		OutputTokens: origStep.OutputTokens,
		Status:       models.StepStatusSuccess,
		IsReplayMock: false,
	}

	// Mutate tool call if specified
	if req.Mutation != nil && req.Mutation.ToolCallOverride != nil {
		override := req.Mutation.ToolCallOverride
		mutatedStep.ToolCalls = []models.ToolCall{
			{
				ToolName:   override.ToolName,
				Arguments:  override.Arguments,
				Output:     "Executed with mutated arguments",
				IsError:    false,
				DurationMS: 15,
				IsVirtual:  false,
			},
		}
	} else {
		// Inherit tool calls
		for _, tc := range origStep.ToolCalls {
			mutatedStep.ToolCalls = append(mutatedStep.ToolCalls, models.ToolCall{
				ToolName:   tc.ToolName,
				Arguments:  tc.Arguments,
				Output:     tc.Output,
				IsError:    tc.IsError,
				DurationMS: tc.DurationMS,
				IsVirtual:  false,
			})
		}
	}

	// Mutate memory snapshot
	var memVars json.RawMessage
	var scratchpad string
	if origStep.MemorySnapshot != nil {
		memVars = origStep.MemorySnapshot.Variables
		scratchpad = origStep.MemorySnapshot.Scratchpad
	}

	if req.Mutation != nil && len(req.Mutation.MemoryOverrides) > 0 {
		memVars = req.Mutation.MemoryOverrides
	}
	if req.Mutation != nil && req.Mutation.ScratchpadOverride != nil {
		scratchpad = *req.Mutation.ScratchpadOverride
	}

	if len(memVars) > 0 || scratchpad != "" {
		mutatedStep.MemorySnapshot = &models.MemorySnapshot{
			Variables:  memVars,
			Scratchpad: scratchpad,
		}
	}

	mutatedStep.StateHash = ComputeStateHash(mutatedStep.TrajectoryID, mutatedStep.StepIndex, mutatedStep.Prompt, mutatedStep.ToolCalls, mutatedStep.MemorySnapshot)

	if err := e.RecordStep(mutatedStep); err != nil {
		return nil, nil, fmt.Errorf("failed to record mutated step: %w", err)
	}

	forkedTraj, _ = e.storage.GetTrajectory(forkedTraj.ID)
	return forkedTraj, mutatedStep, nil
}

// GetStorage returns the underlying storage handle.
func (e *Engine) GetStorage() *storage.Storage {
	return e.storage
}
