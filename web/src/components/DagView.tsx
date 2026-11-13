import React from 'react';
import { Step, Trajectory } from '../types';
import {
  Wrench,
  Brain,
  FileCode,
  Zap,
  ShieldCheck,
  AlertCircle,
  GitBranch,
  ArrowRight,
} from 'lucide-react';

interface DagViewProps {
  trajectory: Trajectory;
  steps: Step[];
  currentStepIndex: number;
  onSelectStep: (index: number) => void;
  onOpenRewindModal: (stepIndex: number) => void;
}

export const DagView: React.FC<DagViewProps> = ({
  trajectory,
  steps,
  currentStepIndex,
  onSelectStep,
  onOpenRewindModal,
}) => {
  if (!steps || steps.length === 0) return null;

  const getNodeIcon = (type: string, isFail: boolean) => {
    if (isFail) return <AlertCircle className="w-4 h-4 text-rose-400" />;
    switch (type) {
      case 'tool_call':
        return <Wrench className="w-4 h-4 text-cyan-400" />;
      case 'reasoning':
        return <Brain className="w-4 h-4 text-purple-400" />;
      case 'planning':
        return <FileCode className="w-4 h-4 text-blue-400" />;
      case 'memory_mutation':
        return <Zap className="w-4 h-4 text-amber-400" />;
      default:
        return <Zap className="w-4 h-4 text-slate-400" />;
    }
  };

  return (
    <div className="border-b border-surface-border bg-surface-elevated/20 p-5 overflow-x-auto select-none">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <span className="text-xs font-bold uppercase tracking-wider text-slate-400">
            Execution DAG Flow
          </span>
          {trajectory.parent_id && (
            <span className="flex items-center gap-1 text-[11px] font-mono px-2 py-0.5 rounded bg-purple-950/80 text-purple-300 border border-purple-800">
              <GitBranch className="w-3 h-3" />
              <span>Branched from step {trajectory.forked_at_step}</span>
            </span>
          )}
        </div>
        <span className="text-xs text-slate-500 font-mono">
          {steps.length} sequential execution nodes
        </span>
      </div>

      <div className="flex items-center gap-2 min-w-max py-2 px-1">
        {/* Parent branch marker if forked */}
        {trajectory.parent_id && (
          <div className="flex items-center gap-2">
            <div className="p-2.5 rounded-xl bg-purple-950/40 border border-purple-700/60 flex flex-col items-center">
              <GitBranch className="w-4 h-4 text-purple-400" />
              <span className="text-[10px] text-purple-300 font-mono mt-0.5">Parent</span>
            </div>
            <div className="w-6 h-0.5 bg-gradient-to-r from-purple-500 to-cyan-500"></div>
          </div>
        )}

        {/* Sequential Step Nodes */}
        {steps.map((step, idx) => {
          const isSelected = idx === currentStepIndex;
          const isFail = step.status === 'failed';
          const isMock = step.is_replay_mock;
          const isLast = idx === steps.length - 1;

          return (
            <React.Fragment key={step.id || idx}>
              <div
                onClick={() => onSelectStep(idx)}
                onDoubleClick={() => onOpenRewindModal(idx)}
                className={`relative group cursor-pointer p-3 rounded-2xl border transition-all flex flex-col gap-1.5 w-44 shrink-0 shadow-sm ${
                  isSelected
                    ? isFail
                      ? 'bg-rose-950/50 border-rose-500 ring-2 ring-rose-500/20'
                      : 'bg-cyan-950/40 border-cyan-400 ring-2 ring-cyan-400/20 shadow-lg shadow-cyan-950/40'
                    : isFail
                    ? 'bg-rose-950/20 border-rose-900/60 hover:border-rose-500/50'
                    : isMock
                    ? 'bg-purple-950/30 border-purple-900/50 hover:border-purple-500/50'
                    : 'bg-surface border-surface-border hover:border-slate-500'
                }`}
              >
                {/* Node Header */}
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-1.5">
                    {getNodeIcon(step.step_type, isFail)}
                    <span className="font-mono text-xs font-bold text-slate-200">
                      Step #{idx}
                    </span>
                  </div>
                  {isMock ? (
                    <span className="flex items-center gap-0.5 text-[9px] font-mono text-purple-300 px-1.5 py-0.5 rounded bg-purple-950 border border-purple-800">
                      <ShieldCheck className="w-2.5 h-2.5" />
                      Mock
                    </span>
                  ) : (
                    <span className="text-[10px] font-mono text-slate-400">
                      {step.duration_ms}ms
                    </span>
                  )}
                </div>

                {/* Node Subtitle */}
                <div className="text-[11px] text-slate-300 capitalize font-medium truncate">
                  {step.tool_calls && step.tool_calls.length > 0
                    ? `tool: ${step.tool_calls[0].tool_name}`
                    : step.step_type.replace('_', ' ')}
                </div>

                {/* Tokens and status */}
                <div className="flex items-center justify-between text-[10px] text-slate-400 font-mono mt-0.5">
                  <span>{step.input_tokens + step.output_tokens} tok</span>
                  <span
                    className={
                      isFail
                        ? 'text-rose-400 font-semibold'
                        : isMock
                        ? 'text-purple-400'
                        : 'text-emerald-400'
                    }
                  >
                    {isFail ? 'FAIL' : isMock ? 'CACHED' : 'SUCCESS'}
                  </span>
                </div>

                {/* Rewind indicator hover */}
                <div className="absolute -top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
                  <span className="text-[9px] px-1.5 py-0.5 rounded bg-purple-600 text-white font-sans font-medium">
                    Double-click to Rewind
                  </span>
                </div>
              </div>

              {/* Connecting DAG Edge */}
              {!isLast && (
                <div className="flex items-center justify-center text-slate-600 px-0.5 shrink-0">
                  <ArrowRight className="w-3.5 h-3.5" />
                </div>
              )}
            </React.Fragment>
          );
        })}
      </div>
    </div>
  );
};
