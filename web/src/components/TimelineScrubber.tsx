import React, { useState, useEffect } from 'react';
import { Step } from '../types';
import {
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  Play,
  Pause,
  RotateCcw,
  Zap,
  Wrench,
  Brain,
  FileCode,
  ShieldCheck,
  AlertCircle,
} from 'lucide-react';

interface TimelineScrubberProps {
  steps: Step[];
  currentStepIndex: number;
  onSelectStep: (index: number) => void;
  onOpenRewindModal: (stepIndex: number) => void;
}

export const TimelineScrubber: React.FC<TimelineScrubberProps> = ({
  steps,
  currentStepIndex,
  onSelectStep,
  onOpenRewindModal,
}) => {
  const [isPlaying, setIsPlaying] = useState(false);

  useEffect(() => {
    let timer: NodeJS.Timeout;
    if (isPlaying) {
      timer = setInterval(() => {
        if (currentStepIndex >= steps.length - 1) {
          setIsPlaying(false);
        } else {
          onSelectStep(currentStepIndex + 1);
        }
      }, 900);
    }
    return () => clearInterval(timer);
  }, [isPlaying, currentStepIndex, steps.length, onSelectStep]);

  if (!steps || steps.length === 0) {
    return null;
  }

  const currentStep = steps[currentStepIndex] || steps[0];

  const getStepIcon = (type: string, isFail: boolean) => {
    if (isFail) return <AlertCircle className="w-3.5 h-3.5 text-rose-400" />;
    switch (type) {
      case 'tool_call':
        return <Wrench className="w-3.5 h-3.5 text-cyan-400" />;
      case 'reasoning':
        return <Brain className="w-3.5 h-3.5 text-purple-400" />;
      case 'planning':
        return <FileCode className="w-3.5 h-3.5 text-blue-400" />;
      case 'memory_mutation':
        return <Zap className="w-3.5 h-3.5 text-amber-400" />;
      default:
        return <Zap className="w-3.5 h-3.5 text-slate-400" />;
    }
  };

  return (
    <div className="border-b border-surface-border bg-surface-elevated/40 p-3 sm:p-4 space-y-3">
      {/* Top controls: Step selector & Rewind Button */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          {/* Nav buttons */}
          <div className="flex items-center gap-1 bg-surface-elevated p-1 rounded-lg border border-surface-border">
            <button
              onClick={() => onSelectStep(0)}
              disabled={currentStepIndex === 0}
              title="First step"
              className="p-1.5 rounded hover:bg-surface-border text-slate-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed"
            >
              <ChevronsLeft className="w-3.5 h-3.5" />
            </button>
            <button
              onClick={() => onSelectStep(Math.max(0, currentStepIndex - 1))}
              disabled={currentStepIndex === 0}
              title="Previous step"
              className="p-1.5 rounded hover:bg-surface-border text-slate-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed"
            >
              <ChevronLeft className="w-3.5 h-3.5" />
            </button>

            <button
              onClick={() => setIsPlaying(!isPlaying)}
              title={isPlaying ? 'Pause replay' : 'Auto play steps'}
              className="px-2 py-1 rounded bg-cyan-500/10 hover:bg-cyan-500/20 text-cyan-400 border border-cyan-500/30 flex items-center gap-1.5 text-xs font-medium"
            >
              {isPlaying ? <Pause className="w-3 h-3 fill-current" /> : <Play className="w-3 h-3 fill-current" />}
              <span>{isPlaying ? 'Pause' : 'Play'}</span>
            </button>

            <button
              onClick={() => onSelectStep(Math.min(steps.length - 1, currentStepIndex + 1))}
              disabled={currentStepIndex === steps.length - 1}
              title="Next step"
              className="p-1.5 rounded hover:bg-surface-border text-slate-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed"
            >
              <ChevronRight className="w-3.5 h-3.5" />
            </button>
            <button
              onClick={() => onSelectStep(steps.length - 1)}
              disabled={currentStepIndex === steps.length - 1}
              title="Last step"
              className="p-1.5 rounded hover:bg-surface-border text-slate-400 hover:text-white disabled:opacity-30 disabled:cursor-not-allowed"
            >
              <ChevronsRight className="w-3.5 h-3.5" />
            </button>
          </div>

          <div className="flex items-center gap-2 text-xs">
            <span className="font-semibold text-slate-200">
              Step {currentStepIndex + 1} of {steps.length}
            </span>
            <span className="text-slate-500">•</span>
            <span className="capitalize px-2 py-0.5 rounded text-[11px] font-medium bg-surface-elevated border border-surface-border text-slate-300">
              {currentStep.step_type.replace('_', ' ')}
            </span>
            {currentStep.is_replay_mock && (
              <span className="flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-mono bg-purple-950/60 text-purple-300 border border-purple-800/60">
                <ShieldCheck className="w-3 h-3" />
                <span>Virtual Cache Replay</span>
              </span>
            )}
          </div>
        </div>

        {/* Rewind Trigger */}
        <button
          onClick={() => onOpenRewindModal(currentStepIndex)}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-purple-600/20 hover:bg-purple-600/30 text-purple-300 border border-purple-500/40 text-xs font-medium transition-colors shadow-sm"
        >
          <RotateCcw className="w-3.5 h-3.5" />
          <span>Rewind to Step {currentStepIndex} & Mutate</span>
        </button>
      </div>

      {/* Horizontal Interactive Step Strip */}
      <div className="relative pt-2 pb-1 overflow-x-auto">
        <div className="flex items-center gap-2 min-w-max px-1">
          {steps.map((step, idx) => {
            const isSelected = idx === currentStepIndex;
            const isFail = step.status === 'failed';
            const isMock = step.is_replay_mock;

            return (
              <button
                key={step.id || idx}
                onClick={() => onSelectStep(idx)}
                title={`Step ${idx}: ${step.step_type} (${step.status})`}
                className={`group relative flex flex-col items-center gap-1.5 p-2 rounded-xl transition-all border ${
                  isSelected
                    ? isFail
                      ? 'bg-rose-950/40 border-rose-500 ring-2 ring-rose-500/20'
                      : 'bg-cyan-950/40 border-cyan-500 ring-2 ring-cyan-500/20 shadow-lg shadow-cyan-950/40'
                    : isFail
                    ? 'bg-rose-950/20 border-rose-900/60 hover:border-rose-500/50'
                    : isMock
                    ? 'bg-purple-950/20 border-purple-900/50 hover:border-purple-500/50'
                    : 'bg-surface-elevated/70 border-surface-border hover:border-slate-500'
                }`}
              >
                <div className="flex items-center gap-1.5">
                  {getStepIcon(step.step_type, isFail)}
                  <span
                    className={`font-mono text-xs font-bold ${
                      isSelected ? 'text-white' : 'text-slate-400 group-hover:text-slate-200'
                    }`}
                  >
                    #{idx}
                  </span>
                </div>

                <div className="flex items-center gap-1 text-[10px]">
                  {step.tool_calls && step.tool_calls.length > 0 && (
                    <span className="text-cyan-400 font-mono">
                      {step.tool_calls.length}t
                    </span>
                  )}
                  <span className="text-slate-500 font-mono">
                    {step.duration_ms}ms
                  </span>
                </div>

                {isSelected && (
                  <div className="absolute -bottom-1 left-1/2 -translate-x-1/2 w-1.5 h-1.5 rounded-full bg-cyan-400"></div>
                )}
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
};
