import { useState, useEffect, useCallback } from 'react';
import { Trajectory, TrajectoryDiff } from './types';
import {
  fetchTrajectories,
  fetchTrajectory,
  deleteTrajectory,
  replayTrajectory,
  fetchDiff,
  triggerDemoRun,
} from './lib/api';
import { Header } from './components/Header';
import { TrajectoryList } from './components/TrajectoryList';
import { TimelineScrubber } from './components/TimelineScrubber';
import { StepInspector } from './components/StepInspector';
import { TimeTravelModal } from './components/TimeTravelModal';
import { DiffViewer } from './components/DiffViewer';
import { ExportModal } from './components/ExportModal';

export function App() {
  const [trajectories, setTrajectories] = useState<Trajectory[]>([]);
  const [selectedTrajectory, setSelectedTrajectory] = useState<Trajectory | null>(null);
  const [currentStepIndex, setCurrentStepIndex] = useState<number>(0);
  const [diffSelection, setDiffSelection] = useState<string[]>([]);
  const [diffData, setDiffData] = useState<TrajectoryDiff | null>(null);

  // Modals state
  const [isRewindModalOpen, setIsRewindModalOpen] = useState(false);
  const [rewindTargetStep, setRewindTargetStep] = useState(0);
  const [isExportModalOpen, setIsExportModalOpen] = useState(false);
  const [isDiffModalOpen, setIsDiffModalOpen] = useState(false);
  const [isWsConnected, setIsWsConnected] = useState(false);

  // Load trajectories list
  const loadTrajectories = useCallback(async () => {
    try {
      const data = await fetchTrajectories();
      setTrajectories(data);
      if (data.length > 0 && !selectedTrajectory) {
        loadTrajectoryDetails(data[0].id);
      }
    } catch (err) {
      console.error('Failed to load trajectories:', err);
    }
  }, [selectedTrajectory]);

  // Load full details for selected trajectory
  const loadTrajectoryDetails = async (id: string) => {
    try {
      const data = await fetchTrajectory(id);
      setSelectedTrajectory(data);
      if (data.steps && data.steps.length > 0) {
        // Default to last step if failed, else step 0
        const failIdx = data.steps.findIndex((s) => s.status === 'failed');
        setCurrentStepIndex(failIdx !== -1 ? failIdx : 0);
      } else {
        setCurrentStepIndex(0);
      }
    } catch (err) {
      console.error('Failed to load trajectory details:', err);
    }
  };

  useEffect(() => {
    loadTrajectories();
  }, []);

  // WebSocket Live Updates
  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws`;
    let ws: WebSocket;

    try {
      ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        setIsWsConnected(true);
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (
            msg.event === 'step_recorded' ||
            msg.event === 'trajectory_started' ||
            msg.event === 'trajectory_completed' ||
            msg.event === 'trajectory_forked'
          ) {
            loadTrajectories();
            if (selectedTrajectory && msg.payload && (msg.payload.trajectory_id === selectedTrajectory.id || msg.payload.id === selectedTrajectory.id)) {
              loadTrajectoryDetails(selectedTrajectory.id);
            }
          }
        } catch (e) {
          console.error('Failed to parse WS message:', e);
        }
      };

      ws.onclose = () => {
        setIsWsConnected(false);
      };
    } catch (e) {
      console.error('WebSocket connection error:', e);
    }

    return () => {
      if (ws) ws.close();
    };
  }, [selectedTrajectory, loadTrajectories]);

  // Handlers
  const handleSelectTrajectory = (id: string) => {
    loadTrajectoryDetails(id);
  };

  const handleDeleteTrajectory = async (id: string) => {
    if (confirm('Delete this trajectory recording?')) {
      await deleteTrajectory(id);
      if (selectedTrajectory?.id === id) {
        setSelectedTrajectory(null);
      }
      loadTrajectories();
    }
  };

  const handleToggleDiffSelection = (id: string) => {
    setDiffSelection((prev) => {
      if (prev.includes(id)) {
        return prev.filter((item) => item !== id);
      }
      if (prev.length >= 2) {
        return [prev[1], id];
      }
      return [...prev, id];
    });
  };

  const handleOpenDiff = async () => {
    if (diffSelection.length < 2) return;
    try {
      const data = await fetchDiff(diffSelection[0], diffSelection[1]);
      setDiffData(data);
      setIsDiffModalOpen(true);
    } catch (err: any) {
      alert('Failed to load diff: ' + err.message);
    }
  };

  const handleRunDemo = async (shouldFail: boolean) => {
    await triggerDemoRun(shouldFail);
    setTimeout(loadTrajectories, 300);
  };

  const handleOpenRewindModal = (stepIndex: number) => {
    setRewindTargetStep(stepIndex);
    setIsRewindModalOpen(true);
  };

  const handleConfirmReplay = async (payload: any) => {
    if (!selectedTrajectory) return;
    const res = await replayTrajectory(selectedTrajectory.id, payload);
    await loadTrajectories();
    await loadTrajectoryDetails(res.trajectory.id);
  };

  const currentStep = selectedTrajectory?.steps ? selectedTrajectory.steps[currentStepIndex] : undefined;

  return (
    <div className="flex flex-col h-screen overflow-hidden bg-background">
      {/* Top Header */}
      <Header
        onRunDemo={handleRunDemo}
        onOpenDiff={handleOpenDiff}
        onRefresh={loadTrajectories}
        isWsConnected={isWsConnected}
        selectedTrajectoryCount={diffSelection.length}
      />

      {/* Main Workspace Body */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left Trajectory Sidebar */}
        <TrajectoryList
          trajectories={trajectories}
          selectedId={selectedTrajectory?.id || null}
          onSelect={handleSelectTrajectory}
          onDelete={handleDeleteTrajectory}
          diffSelection={diffSelection}
          onToggleDiffSelection={handleToggleDiffSelection}
        />

        {/* Center / Right Content Area */}
        <main className="flex-1 flex flex-col overflow-hidden bg-background">
          {selectedTrajectory ? (
            <>
              {/* Interactive Scrubber */}
              <TimelineScrubber
                steps={selectedTrajectory.steps || []}
                currentStepIndex={currentStepIndex}
                onSelectStep={(idx) => setCurrentStepIndex(idx)}
                onOpenRewindModal={handleOpenRewindModal}
              />

              {/* Step Deep Inspector */}
              <StepInspector
                step={currentStep}
                trajectory={selectedTrajectory}
                onOpenRewindModal={handleOpenRewindModal}
                onOpenExportModal={() => setIsExportModalOpen(true)}
              />
            </>
          ) : (
            <div className="flex-1 flex flex-col items-center justify-center p-8 text-center space-y-4">
              <div className="w-16 h-16 rounded-2xl bg-surface-elevated border border-surface-border flex items-center justify-center text-slate-500">
                <svg className="w-8 h-8 text-cyan-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <circle cx="12" cy="12" r="10" />
                  <polygon points="10 8 16 12 10 16 10 8" />
                </svg>
              </div>
              <div className="max-w-md space-y-1">
                <h3 className="text-base font-semibold text-slate-200">No Trajectory Selected</h3>
                <p className="text-xs text-slate-400 leading-relaxed">
                  Select a trajectory from the sidebar, or run the built-in 15-step autonomous agent scenario to inspect multi-tool executions and time-travel debugging.
                </p>
              </div>
              <button
                onClick={() => handleRunDemo(true)}
                className="px-4 py-2 rounded-xl bg-cyan-500 hover:bg-cyan-400 text-slate-950 text-xs font-semibold shadow-lg shadow-cyan-950/40 transition-colors"
              >
                Run 15-Step Agent Scenario
              </button>
            </div>
          )}
        </main>
      </div>

      {/* Modals */}
      <TimeTravelModal
        isOpen={isRewindModalOpen}
        onClose={() => setIsRewindModalOpen(false)}
        stepIndex={rewindTargetStep}
        totalSteps={selectedTrajectory?.steps?.length || 0}
        step={selectedTrajectory?.steps ? selectedTrajectory.steps[rewindTargetStep] : undefined}
        onConfirmReplay={handleConfirmReplay}
      />

      {isDiffModalOpen && diffData && (
        <DiffViewer
          diff={diffData}
          onClose={() => setIsDiffModalOpen(false)}
        />
      )}

      {selectedTrajectory && (
        <ExportModal
          isOpen={isExportModalOpen}
          onClose={() => setIsExportModalOpen(false)}
          trajectoryId={selectedTrajectory.id}
        />
      )}
    </div>
  );
}

export default App;
