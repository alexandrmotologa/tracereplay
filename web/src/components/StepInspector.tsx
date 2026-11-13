import React, { useState } from 'react';
import { Step, Trajectory, RootCauseDiagnostic, TrajectoryStabilityReport } from '../types';
import {
  Copy,
  Check,
  RotateCcw,
  Download,
  AlertTriangle,
  Clock,
  Coins,
  Cpu,
  Layers,
  ShieldCheck,
  FileText,
  Wrench,
  Terminal,
  Sparkles,
  Zap,
} from 'lucide-react';

interface StepInspectorProps {
  step?: Step;
  trajectory: Trajectory;
  diagnostic?: RootCauseDiagnostic | null;
  stabilityReport?: TrajectoryStabilityReport | null;
  onOpenRewindModal: (stepIndex: number, suggestedPrompt?: string, suggestedToolArgs?: any) => void;
  onOpenExportModal: () => void;
}

export const StepInspector: React.FC<StepInspectorProps> = ({
  step,
  trajectory,
  diagnostic,
  stabilityReport,
  onOpenRewindModal,
  onOpenExportModal,
}) => {
  const [copiedHash, setCopiedHash] = useState(false);
  const [copiedPrompt, setCopiedPrompt] = useState(false);

  if (!step) {
    return (
      <div className="flex-1 flex items-center justify-center text-slate-500 text-xs">
        No step selected. Select a step from the timeline scrubber above.
      </div>
    );
  }

  const copyToClipboard = (text: string, setter: (val: boolean) => void) => {
    navigator.clipboard.writeText(text);
    setter(true);
    setTimeout(() => setter(false), 1500);
  };

  const isFail = step.status === 'failed';

  return (
    <div className="flex-1 flex flex-col overflow-y-auto bg-background/50 p-3 sm:p-6 space-y-4 sm:space-y-6">
      {/* Top Banner / Metrics Card */}
      <div className="rounded-2xl bg-surface border border-surface-border p-3.5 sm:p-4 shadow-sm flex flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div className="flex items-center gap-3">
          <div
            className={`w-10 h-10 rounded-xl flex items-center justify-center font-bold text-sm shrink-0 ${
              isFail
                ? 'bg-rose-500/20 text-rose-400 border border-rose-500/40'
                : 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40'
            }`}
          >
            #{step.step_index}
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="font-bold text-sm text-slate-100 capitalize">
                {step.step_type.replace('_', ' ')}
              </h2>
              {step.is_replay_mock && (
                <span className="flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-mono bg-purple-950/80 text-purple-300 border border-purple-800">
                  <ShieldCheck className="w-3 h-3" />
                  <span>Virtual Mock</span>
                </span>
              )}
            </div>
            <div className="flex flex-wrap items-center gap-2 mt-1">
              <span className="text-[11px] text-slate-400 font-mono">
                Hash: {step.state_hash ? step.state_hash.substring(0, 16) + '...' : 'none'}
              </span>
              <button
                onClick={() => copyToClipboard(step.state_hash, setCopiedHash)}
                title="Copy full SHA-256 state hash"
                className="text-slate-500 hover:text-slate-200"
              >
                {copiedHash ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
              </button>
              {stabilityReport && stabilityReport.step_stability[step.step_index] && (
                <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-emerald-950/60 text-emerald-300 border border-emerald-800">
                  Stability: {(stabilityReport.step_stability[step.step_index].stability_score * 100).toFixed(0)}%
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex flex-wrap items-center gap-2">
          <button
            onClick={onOpenExportModal}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-surface-elevated hover:bg-surface-border border border-surface-border text-xs font-medium text-slate-200 transition-colors"
          >
            <Download className="w-3.5 h-3.5 text-cyan-400" />
            <span className="hidden sm:inline">Export Regression Suite</span>
            <span className="sm:hidden">Export Suite</span>
          </button>

          <button
            onClick={() => onOpenRewindModal(step.step_index)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-purple-600 hover:bg-purple-500 text-white text-xs font-semibold shadow-md shadow-purple-950/50 transition-colors"
          >
            <RotateCcw className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">Time-Travel Rewind Here</span>
            <span className="sm:hidden">Rewind</span>
          </button>
        </div>
      </div>

      {/* Metrics Row */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div className="rounded-xl bg-surface/80 border border-surface-border p-3 flex items-center gap-3">
          <div className="p-2 rounded-lg bg-blue-500/10 text-blue-400 border border-blue-500/20">
            <Clock className="w-4 h-4" />
          </div>
          <div>
            <div className="text-[11px] text-slate-400">Duration</div>
            <div className="font-mono text-xs font-semibold text-slate-200">{step.duration_ms} ms</div>
          </div>
        </div>

        <div className="rounded-xl bg-surface/80 border border-surface-border p-3 flex items-center gap-3">
          <div className="p-2 rounded-lg bg-purple-500/10 text-purple-400 border border-purple-500/20">
            <Cpu className="w-4 h-4" />
          </div>
          <div>
            <div className="text-[11px] text-slate-400">Tokens In / Out</div>
            <div className="font-mono text-xs font-semibold text-slate-200">
              {step.input_tokens} / {step.output_tokens}
            </div>
          </div>
        </div>

        <div className="rounded-xl bg-surface/80 border border-surface-border p-3 flex items-center gap-3">
          <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <Coins className="w-4 h-4" />
          </div>
          <div>
            <div className="text-[11px] text-slate-400">Estimated Cost</div>
            <div className="font-mono text-xs font-semibold text-slate-200">
              ${(step.input_tokens * 0.000003 + step.output_tokens * 0.000015).toFixed(5)}
            </div>
          </div>
        </div>

        <div className="rounded-xl bg-surface/80 border border-surface-border p-3 flex items-center gap-3">
          <div className="p-2 rounded-lg bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
            <Layers className="w-4 h-4" />
          </div>
          <div>
            <div className="text-[11px] text-slate-400">Tool Calls</div>
            <div className="font-mono text-xs font-semibold text-slate-200">
              {step.tool_calls ? step.tool_calls.length : 0} executed
            </div>
          </div>
        </div>
      </div>

      {/* Failure Diagnostic Banner */}
      {isFail && (
        <div className="rounded-xl bg-rose-950/40 border border-rose-500/60 p-4 space-y-2">
          <div className="flex items-center gap-2 text-rose-400 font-semibold text-xs">
            <AlertTriangle className="w-4 h-4" />
            <span>Step Execution Failure Detected</span>
          </div>
          <p className="text-xs text-rose-200 font-mono bg-rose-950/70 p-2.5 rounded-lg border border-rose-900/60">
            {step.error_message || trajectory.failure_reason || 'Unknown execution failure'}
          </p>
          <div className="pt-1 flex items-center justify-between">
            <span className="text-[11px] text-slate-400">
              Use Time-Travel Replay to rewind to an earlier step, fix the prompt or parameters, and resume.
            </span>
            <button
              onClick={() => onOpenRewindModal(step.step_index)}
              className="text-xs px-3 py-1 rounded bg-rose-500 hover:bg-rose-400 text-slate-950 font-semibold transition-colors"
            >
              Rewind & Fix
            </button>
          </div>
        </div>
      )}

      {/* AI Root-Cause Diagnostic & Auto-Fix Card */}
      {diagnostic && (
        <div className="rounded-2xl bg-gradient-to-r from-purple-950/40 via-surface to-cyan-950/30 border border-purple-500/40 p-4.5 space-y-3 shadow-lg">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="p-1.5 rounded-lg bg-purple-500/20 text-purple-300 border border-purple-500/40">
                <Sparkles className="w-4 h-4" />
              </div>
              <div>
                <h3 className="font-bold text-xs text-purple-200">
                  AI Root-Cause Diagnostic ({diagnostic.issue_category})
                </h3>
                <span className="text-[11px] text-slate-400 font-mono">
                  Origin isolated at Step #{diagnostic.root_cause_step_index} • Confidence: {(diagnostic.confidence_score * 100).toFixed(0)}%
                </span>
              </div>
            </div>

            <button
              onClick={() =>
                onOpenRewindModal(
                  diagnostic.root_cause_step_index,
                  diagnostic.suggested_prompt,
                  diagnostic.suggested_tool_args
                )
              }
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-purple-600 hover:bg-purple-500 text-white text-xs font-semibold shadow-md shadow-purple-950/40 transition-colors"
            >
              <Zap className="w-3.5 h-3.5 fill-current" />
              <span>Apply Auto-Fix at Step {diagnostic.root_cause_step_index}</span>
            </button>
          </div>

          <p className="text-xs text-slate-300 leading-relaxed bg-surface-elevated/70 p-3 rounded-xl border border-surface-border">
            {diagnostic.explanation}
          </p>

          <div className="text-[11px] text-purple-300 font-mono flex items-center gap-2">
            <span className="text-slate-400">Suggested Action:</span>
            <span>Rewind to step {diagnostic.root_cause_step_index} with validated parameters</span>
          </div>
        </div>
      )}

      {/* Prompt Card */}
      <div className="rounded-xl bg-surface border border-surface-border overflow-hidden">
        <div className="px-4 py-2.5 bg-surface-elevated/70 border-b border-surface-border flex items-center justify-between">
          <div className="flex items-center gap-2 text-xs font-semibold text-slate-200">
            <FileText className="w-3.5 h-3.5 text-blue-400" />
            <span>Prompt / Input Payload</span>
          </div>
          <button
            onClick={() => copyToClipboard(step.prompt, setCopiedPrompt)}
            className="text-slate-400 hover:text-slate-200 flex items-center gap-1 text-[11px]"
          >
            {copiedPrompt ? <Check className="w-3 h-3 text-emerald-400" /> : <Copy className="w-3 h-3" />}
            <span>{copiedPrompt ? 'Copied' : 'Copy'}</span>
          </button>
        </div>
        <div className="p-4">
          <pre className="text-xs text-slate-200 whitespace-pre-wrap font-mono leading-relaxed bg-surface-elevated/40 p-3 rounded-lg border border-surface-border/50">
            {step.prompt || '(No input prompt recorded)'}
          </pre>
        </div>
      </div>

      {/* Response Card */}
      <div className="rounded-xl bg-surface border border-surface-border overflow-hidden">
        <div className="px-4 py-2.5 bg-surface-elevated/70 border-b border-surface-border flex items-center gap-2 text-xs font-semibold text-slate-200">
          <Terminal className="w-3.5 h-3.5 text-purple-400" />
          <span>LLM Reasoning & Output</span>
        </div>
        <div className="p-4">
          <pre className="text-xs text-slate-200 whitespace-pre-wrap font-mono leading-relaxed bg-surface-elevated/40 p-3 rounded-lg border border-surface-border/50">
            {step.response || '(No output recorded)'}
          </pre>
        </div>
      </div>

      {/* Tool Calls Section */}
      {step.tool_calls && step.tool_calls.length > 0 && (
        <div className="rounded-xl bg-surface border border-surface-border overflow-hidden space-y-3 p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 text-xs font-semibold text-slate-200">
              <Wrench className="w-3.5 h-3.5 text-cyan-400" />
              <span>Executed Tool Calls ({step.tool_calls.length})</span>
            </div>
          </div>

          <div className="space-y-3">
            {step.tool_calls.map((tc, idx) => (
              <div
                key={tc.id || idx}
                className={`rounded-xl border p-3.5 space-y-2.5 ${
                  tc.is_error
                    ? 'bg-rose-950/20 border-rose-500/40'
                    : 'bg-surface-elevated/50 border-surface-border'
                }`}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-xs font-bold text-cyan-300">
                      {tc.tool_name}
                    </span>
                    {tc.is_virtual && (
                      <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-purple-950 text-purple-300 border border-purple-800">
                        Virtual Replay Cache
                      </span>
                    )}
                  </div>
                  <span className="text-[11px] font-mono text-slate-400">
                    {tc.duration_ms} ms
                  </span>
                </div>

                <div className="space-y-1">
                  <div className="text-[11px] text-slate-400 font-medium">Arguments:</div>
                  <pre className="text-xs text-slate-300 bg-background/60 p-2.5 rounded-lg font-mono border border-surface-border/60 overflow-x-auto">
                    {typeof tc.arguments === 'string'
                      ? tc.arguments
                      : JSON.stringify(tc.arguments, null, 2)}
                  </pre>
                </div>

                <div className="space-y-1">
                  <div className="text-[11px] text-slate-400 font-medium">Output:</div>
                  <pre className={`text-xs p-2.5 rounded-lg font-mono border overflow-x-auto ${
                    tc.is_error
                      ? 'bg-rose-950/40 text-rose-200 border-rose-900/60'
                      : 'bg-background/60 text-slate-300 border-surface-border/60'
                  }`}>
                    {tc.output}
                  </pre>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Memory Snapshot */}
      {step.memory_snapshot && (
        <div className="rounded-xl bg-surface border border-surface-border overflow-hidden">
          <div className="px-4 py-2.5 bg-surface-elevated/70 border-b border-surface-border flex items-center gap-2 text-xs font-semibold text-slate-200">
            <Layers className="w-3.5 h-3.5 text-amber-400" />
            <span>Agent Memory Snapshot & Scratchpad</span>
          </div>
          <div className="p-4 space-y-3">
            {step.memory_snapshot.scratchpad && (
              <div>
                <div className="text-[11px] text-slate-400 mb-1 font-medium">Scratchpad:</div>
                <div className="text-xs text-slate-200 bg-surface-elevated/50 p-2.5 rounded-lg border border-surface-border font-mono">
                  {step.memory_snapshot.scratchpad}
                </div>
              </div>
            )}
            <div>
              <div className="text-[11px] text-slate-400 mb-1 font-medium">State Variables:</div>
              <pre className="text-xs text-slate-300 bg-surface-elevated/50 p-2.5 rounded-lg border border-surface-border font-mono overflow-x-auto">
                {JSON.stringify(step.memory_snapshot.variables, null, 2)}
              </pre>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
