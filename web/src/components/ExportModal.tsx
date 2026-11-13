import React, { useState, useEffect } from 'react';
import { X, Copy, Check, Download, Code } from 'lucide-react';
import { exportRegression } from '../lib/api';

interface ExportModalProps {
  isOpen: boolean;
  onClose: () => void;
  trajectoryId: string;
}

export const ExportModal: React.FC<ExportModalProps> = ({ isOpen, onClose, trajectoryId }) => {
  const [format, setFormat] = useState<'go' | 'python' | 'ts' | 'otel' | 'json'>('go');
  const [code, setCode] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(false);
  const [copied, setCopied] = useState<boolean>(false);

  useEffect(() => {
    if (!isOpen || !trajectoryId) return;

    let isMounted = true;
    setLoading(true);

    exportRegression(trajectoryId, format)
      .then((data) => {
        if (isMounted) setCode(data);
      })
      .catch((err) => {
        if (isMounted) setCode(`// Error generating test suite: ${err.message}`);
      })
      .finally(() => {
        if (isMounted) setLoading(false);
      });

    return () => {
      isMounted = false;
    };
  }, [isOpen, trajectoryId, format]);

  if (!isOpen) return null;

  const handleCopy = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  const handleDownload = () => {
    let filename = `trajectory_regression_${trajectoryId}.json`;
    if (format === 'go') filename = `trajectory_regression_test.go`;
    if (format === 'python') filename = `test_trajectory_regression.py`;
    if (format === 'ts') filename = `trajectory_regression.test.ts`;
    if (format === 'otel') filename = `openinference_trace.json`;

    const blob = new Blob([code], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-6 bg-black/75 backdrop-blur-sm animate-in fade-in duration-150">
      <div className="w-full max-w-3xl max-h-[85vh] rounded-2xl bg-surface border border-surface-border shadow-2xl flex flex-col overflow-hidden">
        {/* Header */}
        <div className="px-6 py-4 bg-surface-elevated/70 border-b border-surface-border flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-cyan-500/10 text-cyan-400 border border-cyan-500/30">
              <Code className="w-5 h-5" />
            </div>
            <div>
              <h3 className="font-bold text-sm text-slate-100">Export Regression Suite & OpenTelemetry</h3>
              <p className="text-xs text-slate-400">
                Generate offline tests or OpenInference traces for CI/CD and observability platforms
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

        {/* Format Selector Tabs */}
        <div className="px-6 py-3 border-b border-surface-border flex items-center justify-between bg-surface-elevated/30">
          <div className="flex gap-2">
            {(['go', 'python', 'ts', 'otel', 'json'] as const).map((fmt) => (
              <button
                key={fmt}
                onClick={() => setFormat(fmt)}
                className={`px-3 py-1.5 rounded-lg text-xs font-medium uppercase tracking-wider transition-colors ${
                  format === fmt
                    ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40'
                    : 'bg-surface-elevated text-slate-400 hover:text-slate-200 border border-surface-border'
                }`}
              >
                {fmt === 'go'
                  ? 'Go Test'
                  : fmt === 'python'
                  ? 'Python (pytest)'
                  : fmt === 'ts'
                  ? 'TypeScript (Vitest)'
                  : fmt === 'otel'
                  ? 'OpenInference (OTel)'
                  : 'JSON Spec'}
              </button>
            ))}
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={handleCopy}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-surface-elevated hover:bg-surface-border border border-surface-border text-xs text-slate-200 transition-colors"
            >
              {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copied ? 'Copied' : 'Copy Code'}</span>
            </button>

            <button
              onClick={handleDownload}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-cyan-500 hover:bg-cyan-400 text-slate-950 text-xs font-semibold shadow-md shadow-cyan-950/40 transition-colors"
            >
              <Download className="w-3.5 h-3.5" />
              <span>Download File</span>
            </button>
          </div>
        </div>

        {/* Code Content */}
        <div className="flex-1 overflow-auto p-4 bg-background/80">
          {loading ? (
            <div className="h-64 flex items-center justify-center text-xs text-slate-500 font-mono">
              Generating regression test suite...
            </div>
          ) : (
            <pre className="text-xs text-slate-300 font-mono leading-relaxed whitespace-pre p-4 rounded-xl bg-surface border border-surface-border overflow-x-auto">
              {code}
            </pre>
          )}
        </div>
      </div>
    </div>
  );
};
