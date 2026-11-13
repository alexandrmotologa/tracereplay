import React, { useState } from 'react';
import { Trajectory, TrajectoryStatus } from '../types';
import { GitBranch, Clock, Hash, Trash2, CheckCircle, AlertTriangle, Loader2 } from 'lucide-react';

interface TrajectoryListProps {
  trajectories: Trajectory[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  diffSelection: string[];
  onToggleDiffSelection: (id: string) => void;
}

export const TrajectoryList: React.FC<TrajectoryListProps> = ({
  trajectories,
  selectedId,
  onSelect,
  onDelete,
  diffSelection,
  onToggleDiffSelection,
}) => {
  const [filter, setFilter] = useState<string>('all');
  const [search, setSearch] = useState<string>('');

  const filtered = trajectories.filter((t) => {
    if (filter !== 'all' && t.status !== filter) return false;
    if (search.trim()) {
      const q = search.toLowerCase();
      return (
        t.name.toLowerCase().includes(q) ||
        t.agent_id.toLowerCase().includes(q) ||
        t.id.toLowerCase().includes(q)
      );
    }
    return true;
  });

  const getStatusIcon = (status: TrajectoryStatus) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="w-3.5 h-3.5 text-emerald-400" />;
      case 'failed':
        return <AlertTriangle className="w-3.5 h-3.5 text-rose-400" />;
      case 'running':
        return <Loader2 className="w-3.5 h-3.5 text-cyan-400 animate-spin" />;
      default:
        return <Clock className="w-3.5 h-3.5 text-slate-400" />;
    }
  };

  return (
    <aside className="w-80 border-r border-surface-border bg-surface flex flex-col h-full overflow-hidden">
      {/* Search & Filters */}
      <div className="p-3 border-b border-surface-border space-y-2.5">
        <input
          type="text"
          placeholder="Filter trajectories..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full bg-surface-elevated border border-surface-border rounded-lg px-3 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-cyan-500/50"
        />

        <div className="flex gap-1 bg-surface-elevated/60 p-1 rounded-lg border border-surface-border/50 text-[11px]">
          {(['all', 'failed', 'completed', 'running'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setFilter(tab)}
              className={`flex-1 py-1 rounded text-center capitalize transition-colors font-medium ${
                filter === tab
                  ? 'bg-cyan-500/10 text-cyan-400 border border-cyan-500/30'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              {tab}
            </button>
          ))}
        </div>
      </div>

      {/* Trajectories list */}
      <div className="flex-1 overflow-y-auto divide-y divide-surface-border/40 p-2 space-y-1">
        {filtered.length === 0 ? (
          <div className="p-6 text-center text-xs text-slate-500">
            No trajectories found. Run a demo or connect an agent via proxy.
          </div>
        ) : (
          filtered.map((t) => {
            const isSelected = selectedId === t.id;
            const isDiffChecked = diffSelection.includes(t.id);

            return (
              <div
                key={t.id}
                onClick={() => onSelect(t.id)}
                className={`group relative rounded-xl p-3 cursor-pointer transition-all border ${
                  isSelected
                    ? 'bg-surface-elevated border-cyan-500/50 shadow-md shadow-cyan-950/20'
                    : 'bg-surface hover:bg-surface-elevated/50 border-transparent hover:border-surface-border'
                }`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-1.5 flex-1 min-w-0">
                    {getStatusIcon(t.status)}
                    <span className="font-semibold text-xs text-slate-200 truncate group-hover:text-cyan-300 transition-colors">
                      {t.name}
                    </span>
                  </div>

                  <div className="flex items-center gap-1">
                    <input
                      type="checkbox"
                      title="Select for Diff"
                      checked={isDiffChecked}
                      onChange={(e) => {
                        e.stopPropagation();
                        onToggleDiffSelection(t.id);
                      }}
                      className="rounded bg-surface-elevated border-surface-border text-cyan-500 focus:ring-0 cursor-pointer w-3.5 h-3.5"
                    />

                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onDelete(t.id);
                      }}
                      title="Delete trajectory"
                      className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-rose-500/20 text-slate-400 hover:text-rose-400 transition-all"
                    >
                      <Trash2 className="w-3 h-3" />
                    </button>
                  </div>
                </div>

                {t.parent_id && (
                  <div className="mt-1 flex items-center gap-1.5 text-[11px] text-purple-400">
                    <GitBranch className="w-3 h-3" />
                    <span>Forked from step {t.forked_at_step}</span>
                  </div>
                )}

                <div className="mt-2.5 flex items-center justify-between text-[11px] text-slate-400">
                  <div className="flex items-center gap-3">
                    <span className="flex items-center gap-1">
                      <Hash className="w-3 h-3 text-slate-500" />
                      <span>{t.total_steps} steps</span>
                    </span>
                    <span className="font-mono text-slate-500">
                      {t.total_tokens.toLocaleString()} tok
                    </span>
                  </div>
                  <span className="font-mono text-slate-400">
                    ${t.total_cost_usd.toFixed(4)}
                  </span>
                </div>
              </div>
            );
          })
        )}
      </div>
    </aside>
  );
};
