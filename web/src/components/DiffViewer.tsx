import React, { useState } from 'react';
import { TrajectoryDiff } from '../types';
import { X, Check, AlertCircle, ChevronDown, ChevronRight, GitCompare, ArrowRight } from 'lucide-react';

interface DiffViewerProps {
  diff: TrajectoryDiff;
  onClose: () => void;
}

export const DiffViewer: React.FC<DiffViewerProps> = ({ diff, onClose }) => {
  const [expandedStep, setExpandedStep] = useState<number | null>(null);

  const toggleStep = (idx: number) => {
    setExpandedStep(expandedStep === idx ? null : idx);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-6 bg-black/75 backdrop-blur-sm animate-in fade-in duration-150">
      <div className="w-full max-w-4xl max-h-[85vh] rounded-2xl bg-surface border border-surface-border shadow-2xl flex flex-col overflow-hidden">
        {/* Header */}
        <div className="px-6 py-4 bg-surface-elevated/70 border-b border-surface-border flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-purple-500/10 text-purple-400 border border-purple-500/30">
              <GitCompare className="w-5 h-5" />
            </div>
            <div>
              <h3 className="font-bold text-sm text-slate-100">Trajectory Diff Inspector</h3>
              <div className="flex items-center gap-2 text-xs text-slate-400 mt-0.5">
                <span className="font-mono text-cyan-400">{diff.base_trajectory_id.substring(0, 14)}...</span>
                <ArrowRight className="w-3 h-3 text-slate-500" />
                <span className="font-mono text-purple-400">{diff.compare_trajectory_id.substring(0, 14)}...</span>
              </div>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-surface-border transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Delta metrics banner */}
        <div className="px-6 py-3 bg-surface-elevated/40 border-b border-surface-border grid grid-cols-4 gap-4 text-xs">
          <div>
            <div className="text-slate-400">Base Status</div>
            <div className="font-semibold capitalize text-rose-400">{diff.base_status}</div>
          </div>
          <div>
            <div className="text-slate-400">Compare Status</div>
            <div className="font-semibold capitalize text-emerald-400">{diff.compare_status}</div>
          </div>
          <div>
            <div className="text-slate-400">Token Delta</div>
            <div className={`font-mono font-semibold ${diff.token_delta <= 0 ? 'text-emerald-400' : 'text-amber-400'}`}>
              {diff.token_delta > 0 ? `+${diff.token_delta}` : diff.token_delta}
            </div>
          </div>
          <div>
            <div className="text-slate-400">Cost Delta</div>
            <div className={`font-mono font-semibold ${diff.cost_delta_usd <= 0 ? 'text-emerald-400' : 'text-amber-400'}`}>
              ${diff.cost_delta_usd.toFixed(4)}
            </div>
          </div>
        </div>

        {/* Steps comparison list */}
        <div className="flex-1 overflow-y-auto p-6 space-y-3">
          {diff.step_diffs.map((sd) => {
            const isExpanded = expandedStep === sd.step_index;
            const hasDiff = !sd.prompt_equal || !sd.tool_calls_equal || !sd.response_equal || !sd.memory_equal;

            return (
              <div
                key={sd.step_index}
                className={`rounded-xl border transition-all ${
                  hasDiff
                    ? 'bg-purple-950/20 border-purple-800/60'
                    : 'bg-surface-elevated/30 border-surface-border'
                }`}
              >
                <div
                  onClick={() => toggleStep(sd.step_index)}
                  className="p-3.5 flex items-center justify-between cursor-pointer select-none"
                >
                  <div className="flex items-center gap-3">
                    <span className="font-mono text-xs font-bold px-2 py-0.5 rounded bg-surface-elevated text-slate-300 border border-surface-border">
                      Step #{sd.step_index}
                    </span>

                    <div className="flex items-center gap-2 text-xs">
                      {sd.prompt_equal ? (
                        <span className="flex items-center gap-1 text-slate-400">
                          <Check className="w-3 h-3 text-emerald-400" />
                          <span>Prompt Identical</span>
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 text-purple-300 font-medium">
                          <AlertCircle className="w-3 h-3 text-purple-400" />
                          <span>Prompt Mutated</span>
                        </span>
                      )}

                      <span className="text-slate-600">•</span>

                      {sd.tool_calls_equal ? (
                        <span className="flex items-center gap-1 text-slate-400">
                          <Check className="w-3 h-3 text-emerald-400" />
                          <span>Tools Identical</span>
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 text-cyan-300 font-medium">
                          <AlertCircle className="w-3 h-3 text-cyan-400" />
                          <span>Tool Calls Differ</span>
                        </span>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center gap-3">
                    <span className="font-mono text-[11px] text-slate-400">
                      Δ {sd.duration_delta_ms > 0 ? `+${sd.duration_delta_ms}` : sd.duration_delta_ms}ms
                    </span>
                    {isExpanded ? (
                      <ChevronDown className="w-4 h-4 text-slate-400" />
                    ) : (
                      <ChevronRight className="w-4 h-4 text-slate-400" />
                    )}
                  </div>
                </div>

                {/* Expanded side-by-side details */}
                {isExpanded && (
                  <div className="p-4 border-t border-surface-border/60 bg-background/50 grid grid-cols-2 gap-4 text-xs">
                    {/* Base Run */}
                    <div className="space-y-2">
                      <div className="font-bold text-slate-400 text-[11px] uppercase tracking-wider">
                        Base Trajectory (Step {sd.step_index})
                      </div>
                      <div className="space-y-1">
                        <span className="text-[10px] text-slate-500 font-medium">Prompt:</span>
                        <pre className="p-2.5 rounded bg-surface border border-surface-border font-mono text-[11px] text-slate-300 whitespace-pre-wrap">
                          {sd.prompt_base || '(empty)'}
                        </pre>
                      </div>
                      {sd.base_tool_calls && sd.base_tool_calls.length > 0 && (
                        <div className="space-y-1">
                          <span className="text-[10px] text-slate-500 font-medium">Tool Call Output:</span>
                          <pre className="p-2.5 rounded bg-surface border border-surface-border font-mono text-[11px] text-rose-300 whitespace-pre-wrap">
                            {sd.base_tool_calls[0].output}
                          </pre>
                        </div>
                      )}
                    </div>

                    {/* Compare Run */}
                    <div className="space-y-2">
                      <div className="font-bold text-purple-400 text-[11px] uppercase tracking-wider">
                        Compare Trajectory (Step {sd.step_index})
                      </div>
                      <div className="space-y-1">
                        <span className="text-[10px] text-slate-500 font-medium">Prompt:</span>
                        <pre className="p-2.5 rounded bg-surface border border-purple-900/50 font-mono text-[11px] text-purple-200 whitespace-pre-wrap">
                          {sd.prompt_compare || '(empty)'}
                        </pre>
                      </div>
                      {sd.compare_tool_calls && sd.compare_tool_calls.length > 0 && (
                        <div className="space-y-1">
                          <span className="text-[10px] text-slate-500 font-medium">Tool Call Output:</span>
                          <pre className="p-2.5 rounded bg-surface border border-cyan-900/50 font-mono text-[11px] text-cyan-200 whitespace-pre-wrap">
                            {sd.compare_tool_calls[0].output}
                          </pre>
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};
