import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  CheckCircle,
  AlertCircle,
  ArrowRight,
  ArrowLeft,
  X,
  Wifi,
  Brain,
  Database,
  Download,
  Sparkles,
  Loader2,
} from 'lucide-react';

interface OnboardingWizardProps {
  onComplete: () => void;
}

interface StepStatus {
  health: { api: boolean; ollama: boolean; qdrant: boolean; database: boolean } | null;
  ingesting: boolean;
  ingestResult: string | null;
}

const STEPS = [
  { title: 'Connection Check', icon: Wifi },
  { title: 'AI Setup', icon: Brain },
  { title: 'Add Data Source', icon: Database },
  { title: 'First Ingest', icon: Download },
  { title: 'Ready!', icon: Sparkles },
];

export function OnboardingWizard({ onComplete }: OnboardingWizardProps) {
  const navigate = useNavigate();
  const [step, setStep] = useState(0);
  const [status, setStatus] = useState<StepStatus>({
    health: null,
    ingesting: false,
    ingestResult: null,
  });

  const checkHealth = useCallback(async () => {
    try {
      const health = await window.mycelicMemory.stats.health();
      setStatus((s) => ({ ...s, health }));
    } catch {
      setStatus((s) => ({ ...s, health: { api: false, ollama: false, qdrant: false, database: false } }));
    }
  }, []);

  useEffect(() => {
    checkHealth();
  }, [checkHealth]);

  const handleIngest = async () => {
    setStatus((s) => ({ ...s, ingesting: true, ingestResult: null }));
    try {
      const result = await window.mycelicMemory.claude.ingest();
      setStatus((s) => ({
        ...s,
        ingesting: false,
        ingestResult: `${result.sessions_created} sessions, ${result.messages_created} messages ingested`,
      }));
    } catch {
      setStatus((s) => ({ ...s, ingesting: false, ingestResult: 'Ingestion failed. You can try again later from Sessions.' }));
    }
  };

  const handleComplete = () => {
    localStorage.setItem('mycelicmemory_onboarding_done', 'true');
    onComplete();
  };

  const handleSkip = () => {
    localStorage.setItem('mycelicmemory_onboarding_done', 'true');
    onComplete();
  };

  const renderStep = () => {
    switch (step) {
      case 0: // Connection Check
        return (
          <div className="space-y-4">
            <p className="text-slate-400">
              Let's verify your MycelicMemory backend is running and accessible.
            </p>
            <div className="space-y-3">
              <StatusItem label="MycelicMemory API" ok={status.health?.api} />
              <StatusItem label="Database (SQLite)" ok={status.health?.database} />
              <StatusItem label="Ollama (AI)" ok={status.health?.ollama} optional />
              <StatusItem label="Qdrant (Vector DB)" ok={status.health?.qdrant} optional />
            </div>
            {status.health && !status.health.api && (
              <div className="p-3 bg-red-500/10 border border-red-500/30 rounded-lg text-sm text-red-400">
                API not connected. Make sure MycelicMemory is running:
                <code className="block mt-1 bg-slate-900 px-2 py-1 rounded text-xs">mycelicmemory start</code>
              </div>
            )}
            <button
              onClick={checkHealth}
              className="text-sm text-primary-400 hover:text-primary-300"
            >
              Retry connection check
            </button>
          </div>
        );

      case 1: // AI Setup
        return (
          <div className="space-y-4">
            <p className="text-slate-400">
              MycelicMemory uses Ollama for AI-powered features like semantic search and analysis.
            </p>
            {status.health?.ollama ? (
              <div className="p-4 bg-green-500/10 border border-green-500/30 rounded-lg">
                <div className="flex items-center gap-2 text-green-400">
                  <CheckCircle className="w-5 h-5" />
                  <span className="font-medium">Ollama is connected</span>
                </div>
                <p className="text-sm text-slate-400 mt-2">
                  AI features are ready. You can configure models in Settings.
                </p>
              </div>
            ) : (
              <div className="p-4 bg-amber-500/10 border border-amber-500/30 rounded-lg">
                <div className="flex items-center gap-2 text-amber-400">
                  <AlertCircle className="w-5 h-5" />
                  <span className="font-medium">Ollama not detected</span>
                </div>
                <p className="text-sm text-slate-400 mt-2">
                  Ollama is optional but recommended. Install it from{' '}
                  <span className="text-primary-400">ollama.com</span> to enable semantic search.
                </p>
              </div>
            )}
            <button
              onClick={() => navigate('/settings')}
              className="text-sm text-primary-400 hover:text-primary-300"
            >
              Go to Settings to configure models
            </button>
          </div>
        );

      case 2: // Add Data Source
        return (
          <div className="space-y-4">
            <p className="text-slate-400">
              MycelicMemory can ingest your Claude Code conversations automatically.
            </p>
            <div className="p-4 bg-slate-700/50 rounded-lg">
              <h4 className="font-medium mb-2">Claude Code History</h4>
              <p className="text-sm text-slate-400">
                Your local Claude Code sessions are stored in{' '}
                <code className="bg-slate-900 px-1 rounded text-xs">~/.claude/projects/</code>.
                We'll import these in the next step.
              </p>
            </div>
            <div className="p-4 bg-slate-700/50 rounded-lg opacity-60">
              <h4 className="font-medium mb-2">Other Sources (Coming Soon)</h4>
              <p className="text-sm text-slate-400">
                Slack, email, Notion, Obsidian, and more can be added from the Data Sources page.
              </p>
            </div>
          </div>
        );

      case 3: // First Ingest
        return (
          <div className="space-y-4">
            <p className="text-slate-400">
              Let's import your Claude Code conversations to populate your memory graph.
            </p>
            {!status.ingestResult ? (
              <button
                onClick={handleIngest}
                disabled={status.ingesting}
                className="w-full py-3 bg-primary-500 hover:bg-primary-600 disabled:opacity-50 rounded-lg transition-colors flex items-center justify-center gap-2"
              >
                {status.ingesting ? (
                  <>
                    <Loader2 className="w-5 h-5 animate-spin" />
                    Ingesting conversations...
                  </>
                ) : (
                  <>
                    <Download className="w-5 h-5" />
                    Import Claude Code History
                  </>
                )}
              </button>
            ) : (
              <div className="p-4 bg-green-500/10 border border-green-500/30 rounded-lg">
                <div className="flex items-center gap-2 text-green-400">
                  <CheckCircle className="w-5 h-5" />
                  <span className="font-medium">Import complete</span>
                </div>
                <p className="text-sm text-slate-400 mt-1">{status.ingestResult}</p>
              </div>
            )}
            <p className="text-xs text-slate-500">
              You can skip this step and import later from the Sessions page.
            </p>
          </div>
        );

      case 4: // Complete
        return (
          <div className="space-y-4 text-center">
            <div className="w-16 h-16 bg-gradient-to-br from-primary-500 to-mycelium-500 rounded-2xl flex items-center justify-center mx-auto">
              <Sparkles className="w-8 h-8 text-white" />
            </div>
            <h3 className="text-xl font-semibold">You're all set!</h3>
            <p className="text-slate-400">
              Start exploring your memories, browse sessions, or check out the knowledge graph.
            </p>
            <div className="grid grid-cols-3 gap-3 mt-4">
              <button
                onClick={() => { handleComplete(); navigate('/memories'); }}
                className="p-3 bg-slate-700 hover:bg-slate-600 rounded-lg text-sm transition-colors"
              >
                Memories
              </button>
              <button
                onClick={() => { handleComplete(); navigate('/sessions'); }}
                className="p-3 bg-slate-700 hover:bg-slate-600 rounded-lg text-sm transition-colors"
              >
                Sessions
              </button>
              <button
                onClick={() => { handleComplete(); navigate('/graph'); }}
                className="p-3 bg-slate-700 hover:bg-slate-600 rounded-lg text-sm transition-colors"
              >
                Graph
              </button>
            </div>
          </div>
        );
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60" />
      <div className="relative bg-slate-800 border border-slate-700 rounded-xl shadow-2xl max-w-lg w-full mx-4 animate-fade-in">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-700">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 bg-gradient-to-br from-primary-500 to-mycelium-500 rounded-lg flex items-center justify-center">
              <Brain className="w-5 h-5 text-white" />
            </div>
            <div>
              <h2 className="font-semibold">Welcome to MycelicMemory</h2>
              <p className="text-xs text-slate-400">Step {step + 1} of {STEPS.length}: {STEPS[step].title}</p>
            </div>
          </div>
          <button
            onClick={handleSkip}
            className="p-1 hover:bg-slate-700 rounded-lg transition-colors"
            title="Skip onboarding"
          >
            <X className="w-4 h-4 text-slate-400" />
          </button>
        </div>

        {/* Progress */}
        <div className="flex gap-1 px-4 pt-4">
          {STEPS.map((_, i) => (
            <div
              key={i}
              className={`h-1 flex-1 rounded-full transition-colors ${
                i <= step ? 'bg-primary-500' : 'bg-slate-700'
              }`}
            />
          ))}
        </div>

        {/* Content */}
        <div className="p-6 min-h-[240px]">
          {renderStep()}
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-slate-700 flex items-center justify-between">
          <button
            onClick={() => setStep(Math.max(0, step - 1))}
            disabled={step === 0}
            className="flex items-center gap-1 text-sm text-slate-400 hover:text-slate-300 disabled:opacity-30"
          >
            <ArrowLeft className="w-4 h-4" /> Back
          </button>
          {step < STEPS.length - 1 ? (
            <button
              onClick={() => setStep(step + 1)}
              className="flex items-center gap-1 px-4 py-2 bg-primary-500 hover:bg-primary-600 rounded-lg text-sm transition-colors"
            >
              Next <ArrowRight className="w-4 h-4" />
            </button>
          ) : (
            <button
              onClick={handleComplete}
              className="px-4 py-2 bg-primary-500 hover:bg-primary-600 rounded-lg text-sm transition-colors"
            >
              Get Started
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

function StatusItem({ label, ok, optional }: { label: string; ok?: boolean; optional?: boolean }) {
  return (
    <div className="flex items-center justify-between py-2">
      <span className="text-sm">{label}</span>
      <div className="flex items-center gap-2">
        {ok === undefined || ok === null ? (
          <span className="text-xs text-slate-500">Checking...</span>
        ) : ok ? (
          <CheckCircle className="w-4 h-4 text-green-400" />
        ) : (
          <>
            <AlertCircle className={`w-4 h-4 ${optional ? 'text-amber-400' : 'text-red-400'}`} />
            {optional && <span className="text-xs text-slate-500">Optional</span>}
          </>
        )}
      </div>
    </div>
  );
}
