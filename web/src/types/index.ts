export type TrajectoryStatus = 'running' | 'completed' | 'failed' | 'aborted';

export type StepType = 
  | 'planning' 
  | 'reasoning' 
  | 'tool_call' 
  | 'tool_result' 
  | 'llm_generation' 
  | 'memory_mutation' 
  | 'user_input';

export type StepStatus = 'success' | 'failed' | 'skipped';

export interface ToolCall {
  id: string;
  step_id: string;
  tool_name: string;
  arguments: any;
  output: string;
  is_error: boolean;
  duration_ms: number;
  is_virtual: boolean;
  created_at: string;
}

export interface MemorySnapshot {
  id: string;
  step_id: string;
  variables: Record<string, any>;
  scratchpad: string;
  created_at: string;
}

export interface Step {
  id: string;
  trajectory_id: string;
  step_index: number;
  step_type: StepType;
  prompt: string;
  response: string;
  state_hash: string;
  duration_ms: number;
  input_tokens: number;
  output_tokens: number;
  status: StepStatus;
  error_message?: string;
  tool_calls?: ToolCall[];
  memory_snapshot?: MemorySnapshot;
  is_replay_mock: boolean;
  created_at: string;
}

export interface Trajectory {
  id: string;
  parent_id?: string;
  forked_at_step?: number;
  name: string;
  agent_id: string;
  model: string;
  status: TrajectoryStatus;
  failure_reason?: string;
  total_steps: number;
  total_tokens: number;
  total_cost_usd: number;
  started_at: string;
  completed_at?: string;
  metadata?: Record<string, any>;
  steps?: Step[];
  created_at: string;
  updated_at: string;
}

export interface StepDiff {
  step_index: number;
  prompt_equal: boolean;
  prompt_base: string;
  prompt_compare: string;
  response_equal: boolean;
  response_base: string;
  response_compare: string;
  tool_calls_equal: boolean;
  base_tool_calls?: ToolCall[];
  compare_tool_calls?: ToolCall[];
  memory_equal: boolean;
  duration_delta_ms: number;
}

export interface TrajectoryDiff {
  base_trajectory_id: string;
  compare_trajectory_id: string;
  step_diffs: StepDiff[];
  token_delta: number;
  cost_delta_usd: number;
  base_status: TrajectoryStatus;
  compare_status: TrajectoryStatus;
}

export interface RootCauseDiagnostic {
  trajectory_id: string;
  failure_step_index: number;
  root_cause_step_index: number;
  issue_category: string;
  severity: string;
  explanation: string;
  suggested_prompt: string;
  suggested_tool_name?: string;
  suggested_tool_args?: any;
  confidence_score: number;
}

export interface StepStability {
  step_index: number;
  step_type: string;
  stability_score: number;
  is_replay_mock: boolean;
  drift_risk_factors?: string[];
}

export interface TrajectoryStabilityReport {
  trajectory_id: string;
  overall_determinism_score: number;
  total_steps: number;
  zero_drift_steps: number;
  drift_risk_steps: number;
  step_stability: StepStability[];
}
