import React, { useState } from 'react';
import { Play, Sparkles, GitCompare, RefreshCw, Radio, Menu, X } from 'lucide-react';

interface HeaderProps {
  onRunDemo: (shouldFail: boolean) => void;
  onOpenDiff: () => void;
  onRefresh: () => void;
  isWsConnected: boolean;
  selectedTrajectoryCount: number;
  isMobileSidebarOpen?: boolean;
  onToggleMobileSidebar?: () => void;
}

export const Header: React.FC<HeaderProps> = ({
  onRunDemo,
  onOpenDiff,
  onRefresh,
  isWsConnected,
  selectedTrajectoryCount,
  isMobileSidebarOpen,
  onToggleMobileSidebar,
}) => {
  const [isRunning, setIsRunning] = useState(false);
  const [showDemoMenu, setShowDemoMenu] = useState(false);

  const handleRun = async (shouldFail: boolean) => {
    setIsRunning(true);
    setShowDemoMenu(false);
    try {
      await onRunDemo(shouldFail);
    } finally {
      setTimeout(() => setIsRunning(false), 800);
    }
  };

  return (
    <header className="h-16 border-b border-surface-border bg-surface/80 backdrop-blur-md px-4 sm:px-6 flex items-center justify-between z-20 sticky top-0">
      <div className="flex items-center gap-2.5 sm:gap-3">
        {onToggleMobileSidebar && (
          <button
            onClick={onToggleMobileSidebar}
            className="md:hidden p-2 rounded-lg bg-surface-elevated hover:bg-surface-border border border-surface-border text-slate-300"
            title="Toggle trajectories sidebar"
          >
            {isMobileSidebarOpen ? <X className="w-4 h-4" /> : <Menu className="w-4 h-4" />}
          </button>
        )}

        <div className="w-9 h-9 rounded-lg bg-gradient-to-tr from-cyan-600 to-blue-500 flex items-center justify-center shadow-lg shadow-cyan-950/40 border border-cyan-400/30 shrink-0">
          <svg className="w-5 h-5 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="12" cy="12" r="10" />
            <polygon points="10 8 16 12 10 16 10 8" />
          </svg>
        </div>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <h1 className="font-bold text-base sm:text-lg text-slate-100 tracking-tight">TraceReplay</h1>
            <span className="text-[10px] sm:text-xs px-1.5 sm:px-2 py-0.5 rounded-full font-mono bg-cyan-950/80 text-cyan-400 border border-cyan-800/60">
              v1.0.0
            </span>
          </div>
          <p className="text-xs text-slate-400 hidden sm:block truncate">Deterministic Agent Trajectory Recorder & Time-Travel Debugger</p>
        </div>
      </div>

      <div className="flex items-center gap-2 sm:gap-3">
        {/* WS Status indicator */}
        <div className="flex items-center gap-1.5 sm:gap-2 px-2 sm:px-3 py-1.5 rounded-md bg-surface-elevated/70 border border-surface-border text-xs text-slate-400">
          <Radio className={`w-3.5 h-3.5 shrink-0 ${isWsConnected ? 'text-emerald-400 animate-pulse' : 'text-slate-500'}`} />
          <span className="hidden sm:inline">{isWsConnected ? 'Live Sync Active' : 'Offline'}</span>
        </div>

        {/* Refresh button */}
        <button
          onClick={onRefresh}
          title="Refresh trajectories"
          className="p-2 rounded-lg bg-surface-elevated hover:bg-surface-border border border-surface-border text-slate-300 hover:text-white transition-colors"
        >
          <RefreshCw className="w-4 h-4" />
        </button>

        {/* Diff view button */}
        <button
          onClick={onOpenDiff}
          disabled={selectedTrajectoryCount < 2}
          className="flex items-center gap-1.5 sm:gap-2 px-2.5 sm:px-3 py-1.5 rounded-lg bg-surface-elevated hover:bg-surface-border border border-surface-border text-slate-300 hover:text-white disabled:opacity-50 disabled:cursor-not-allowed text-xs font-medium transition-colors"
        >
          <GitCompare className="w-4 h-4 text-purple-400 shrink-0" />
          <span className="hidden sm:inline">Diff Trajectories</span>
          <span className="sm:hidden">Diff</span>
        </button>

        {/* Run Demo dropdown */}
        <div className="relative">
          <button
            onClick={() => setShowDemoMenu(!showDemoMenu)}
            disabled={isRunning}
            className="flex items-center gap-1.5 sm:gap-2 px-2.5 sm:px-3.5 py-1.5 rounded-lg bg-cyan-500 hover:bg-cyan-400 text-slate-950 font-semibold text-xs transition-colors shadow-md shadow-cyan-950/50"
          >
            <Play className={`w-3.5 h-3.5 fill-current shrink-0 ${isRunning ? 'animate-spin' : ''}`} />
            <span className="hidden sm:inline">{isRunning ? 'Running Demo...' : 'Run 15-Step Demo'}</span>
            <span className="sm:hidden">{isRunning ? 'Running...' : 'Demo'}</span>
          </button>

          {showDemoMenu && (
            <div className="absolute right-0 mt-2 w-64 rounded-xl bg-surface-elevated border border-surface-border shadow-2xl py-1.5 z-30 animate-in fade-in slide-in-from-top-2 duration-150">
              <button
                onClick={() => handleRun(true)}
                className="w-full text-left px-4 py-2.5 hover:bg-surface-border/70 flex flex-col transition-colors"
              >
                <div className="flex items-center gap-2 text-rose-400 font-medium text-xs">
                  <span className="w-2 h-2 rounded-full bg-rose-500"></span>
                  <span>Scenario A: Fails at Step 12</span>
                </div>
                <span className="text-[11px] text-slate-400 mt-0.5">
                  Simulates production bug with hallucinated schema version
                </span>
              </button>

              <div className="my-1 border-t border-surface-border/50"></div>

              <button
                onClick={() => handleRun(false)}
                className="w-full text-left px-4 py-2.5 hover:bg-surface-border/70 flex flex-col transition-colors"
              >
                <div className="flex items-center gap-2 text-emerald-400 font-medium text-xs">
                  <Sparkles className="w-3.5 h-3.5 text-emerald-400" />
                  <span>Scenario B: Clean Run (15 Steps)</span>
                </div>
                <span className="text-[11px] text-slate-400 mt-0.5">
                  Runs all 15 steps with correct schema v12
                </span>
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
};
