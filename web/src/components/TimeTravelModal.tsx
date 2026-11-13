import React, { useState } from 'react';
import { RotateCcw, X, Sparkles, AlertCircle } from 'lucide-react';
import { Step } from '../types';

interface TimeTravelModalProps {
  isOpen: boolean;
  onClose: () => void;
  stepIndex: number;
  totalSteps: number;
  step?: Step;
  onConfirmReplay: (payload: {
    from_step: number;
    name?: string;
    mutation?: {
      prompt?: string;
      tool_call_override?: {
        tool_name: string;
        arguments: any;
      };
      memory_overrides?: any;
    };
  }) => Promise<void>;
}

export const TimeTravelModal: React.FC<TimeTravelModalProps> = ({
  isOpen,
  onClose,
  stepIndex,
  totalSteps,
  step,
  onConfirmReplay,
}) => {
  const [targetStep, setTargetStep] = useState<number>(stepIndex);
  const [branchName, setBranchName] = useState<string>('');
  const [mutatedPrompt, setMutatedPrompt] = useState<string>('');
  const [mutatedToolName, setMutatedToolName] = useState<string>('');
  const [mutatedToolArgs, setMutatedToolArgs] = useState<string>('');
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  // Sync state when step changes
  React.useEffect(() => {
    setTargetStep(stepIndex);
    setBranchName(`repaired-run-from-step-${stepIndex}`);
    if (step) {
      setMutatedPrompt(step.prompt || '');
      if (step.tool_calls && step.tool_calls.length > 0) {
        setMutatedToolName(step.tool_calls[0].tool_name);
        setMutatedToolArgs(
          typeof step.tool_calls[0].arguments === 'string'
            ? step.tool_calls[0].arguments
            : JSON.stringify(step.tool_calls[0].arguments, null, 2)
        );
      } else {
        setMutatedToolName('');
        setMutatedToolArgs('');
      }
    }
  }, [stepIndex, step]);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    try {
      let parsedToolArgs = undefined;
      if (mutatedToolArgs.trim()) {
        try {
          parsedToolArgs = JSON.parse(mutatedToolArgs);
        } catch (err: any) {
          throw new Error('Tool arguments must be valid JSON: ' + err.message);
        }
      }

      await onConfirmReplay({
        from_step: targetStep,
        name: branchName.trim() || undefined,
        mutation: {
          prompt: mutatedPrompt !== step?.prompt ? mutatedPrompt : undefined,
          tool_call_override:
            mutatedToolName && parsedToolArgs
              ? {
                  tool_name: mutatedToolName,
                  arguments: parsedToolArgs,
                }
              : undefined,
        },
      });
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to resume replay');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-in fade-in duration-150">
      <div className="w-full max-w-xl rounded-2xl bg-surface border border-surface-border shadow-2xl overflow-hidden flex flex-col">
        {/* Modal Header */}
        <div className="px-6 py-4 bg-surface-elevated/70 border-b border-surface-border flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-lg bg-purple-500/10 text-purple-400 border border-purple-500/30">
              <RotateCcw className="w-4 h-4" />
            </div>
            <div>
              <h3 className="font-bold text-sm text-slate-100">Time-Travel Rewind & Mutate</h3>
              <p className="text-xs text-slate-400">
                Rewind to step {targetStep}, modify parameters, and re-execute deterministically
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-surface-border transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="p-6 space-y-4 flex-1 overflow-y-auto">
          {error && (
            <div className="p-3 rounded-xl bg-rose-950/40 border border-rose-500/50 flex items-start gap-2 text-xs text-rose-300">
              <AlertCircle className="w-4 h-4 shrink-0 text-rose-400 mt-0.5" />
              <span>{error}</span>
            </div>
          )}

          {/* Explanation banner */}
          <div className="p-3.5 rounded-xl bg-cyan-950/30 border border-cyan-800/40 flex items-start gap-3">
            <Sparkles className="w-4 h-4 text-cyan-400 shrink-0 mt-0.5" />
            <div className="text-xs text-slate-300 space-y-1">
              <span className="font-semibold text-cyan-300">Deterministic Virtual Execution:</span>
              <p className="text-slate-400 leading-relaxed">
                Steps 0 through {Math.max(0, targetStep - 1)} will be instantly resolved from the recorded cache without re-invoking external tools or incurring API costs.
              </p>
            </div>
          </div>

          {/* Step Selector & Branch Name */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1.5">
                Rewind Step Index
              </label>
              <select
                value={targetStep}
                onChange={(e) => setTargetStep(Number(e.target.value))}
                className="w-full bg-surface-elevated border border-surface-border rounded-lg px-3 py-2 text-xs text-slate-200 focus:outline-none focus:border-cyan-500"
              >
                {Array.from({ length: totalSteps }).map((_, i) => (
                  <option key={i} value={i}>
                    Step {i} {i === stepIndex ? '(Current)' : ''}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-xs font-medium text-slate-300 mb-1.5">
                Branch Trajectory Name
              </label>
              <input
                type="text"
                value={branchName}
                onChange={(e) => setBranchName(e.target.value)}
                placeholder="repaired-run-v1"
                className="w-full bg-surface-elevated border border-surface-border rounded-lg px-3 py-2 text-xs text-slate-200 focus:outline-none focus:border-cyan-500"
              />
            </div>
          </div>

          {/* Mutate Prompt */}
          <div>
            <label className="block text-xs font-medium text-slate-300 mb-1.5">
              Mutate Input Prompt at Step {targetStep}
            </label>
            <textarea
              rows={3}
              value={mutatedPrompt}
              onChange={(e) => setMutatedPrompt(e.target.value)}
              className="w-full bg-surface-elevated border border-surface-border rounded-lg p-3 text-xs text-slate-200 font-mono focus:outline-none focus:border-cyan-500"
              placeholder="Enter patched prompt instructions..."
            />
          </div>

          {/* Mutate Tool Call */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <label className="block text-xs font-medium text-slate-300">
                Override Tool Call Arguments (Optional)
              </label>
              <span className="text-[10px] text-slate-500 font-mono">JSON format</span>
            </div>

            <div className="grid grid-cols-3 gap-2">
              <input
                type="text"
                value={mutatedToolName}
                onChange={(e) => setMutatedToolName(e.target.value)}
                placeholder="tool_name (e.g. run_migration)"
                className="col-span-1 bg-surface-elevated border border-surface-border rounded-lg px-3 py-2 text-xs text-slate-200 font-mono focus:outline-none focus:border-cyan-500"
              />
              <textarea
                rows={3}
                value={mutatedToolArgs}
                onChange={(e) => setMutatedToolArgs(e.target.value)}
                placeholder='{"target_version": 12}'
                className="col-span-2 bg-surface-elevated border border-surface-border rounded-lg p-2.5 text-xs text-slate-200 font-mono focus:outline-none focus:border-cyan-500"
              />
            </div>
          </div>

          {/* Footer controls */}
          <div className="pt-3 border-t border-surface-border flex items-center justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-lg bg-surface-elevated hover:bg-surface-border border border-surface-border text-xs text-slate-300 transition-colors"
            >
              Cancel
            </button>

            <button
              type="submit"
              disabled={isSubmitting}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-purple-600 hover:bg-purple-500 text-white text-xs font-semibold shadow-lg shadow-purple-950/50 transition-colors disabled:opacity-50"
            >
              <RotateCcw className={`w-3.5 h-3.5 ${isSubmitting ? 'animate-spin' : ''}`} />
              <span>{isSubmitting ? 'Resuming Replay...' : 'Fork & Replay From Step'}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
