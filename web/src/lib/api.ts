import { Trajectory, TrajectoryDiff } from '../types';

const API_BASE = '/api/v1';

export async function fetchTrajectories(status?: string, agentId?: string): Promise<Trajectory[]> {
  const params = new URLSearchParams();
  if (status) params.append('status', status);
  if (agentId) params.append('agent_id', agentId);

  const res = await fetch(`${API_BASE}/trajectories?${params.toString()}`);
  if (!res.ok) throw new Error('Failed to fetch trajectories');
  return res.json();
}

export async function fetchTrajectory(id: string): Promise<Trajectory> {
  const res = await fetch(`${API_BASE}/trajectories/${id}`);
  if (!res.ok) throw new Error('Failed to fetch trajectory details');
  return res.json();
}

export async function deleteTrajectory(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/trajectories/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error('Failed to delete trajectory');
}

export interface ReplayPayload {
  from_step: number;
  name?: string;
  mutation?: {
    prompt?: string;
    tool_call_override?: {
      tool_name: string;
      arguments: any;
    };
    memory_overrides?: any;
    scratchpad_override?: string;
  };
}

export async function replayTrajectory(id: string, payload: ReplayPayload): Promise<{ trajectory: Trajectory; mutated_step: any }> {
  const res = await fetch(`${API_BASE}/trajectories/${id}/replay`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const errText = await res.text();
    throw new Error(errText || 'Failed to replay trajectory');
  }
  return res.json();
}

export async function fetchDiff(baseId: string, compareId: string): Promise<TrajectoryDiff> {
  const res = await fetch(`${API_BASE}/diff?base=${encodeURIComponent(baseId)}&compare=${encodeURIComponent(compareId)}`);
  if (!res.ok) throw new Error('Failed to compute trajectory diff');
  return res.json();
}

export async function exportRegression(id: string, format: 'go' | 'python' | 'json'): Promise<string> {
  const res = await fetch(`${API_BASE}/trajectories/${id}/export?format=${format}`);
  if (!res.ok) throw new Error(`Failed to export regression test in format ${format}`);
  return res.text();
}

export async function triggerDemoRun(shouldFail: boolean = true, fixedVersion: number = 12): Promise<void> {
  const res = await fetch(`${API_BASE}/demo/run`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ should_fail: shouldFail, fixed_version: fixedVersion }),
  });
  if (!res.ok) throw new Error('Failed to start demo agent run');
}
