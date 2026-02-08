import { useState, useEffect } from 'react';
import {
  Download,
  Play,
  Pause,
  Settings,
  CheckCircle,
  XCircle,
  Clock,
  Loader2,
  RefreshCw,
  AlertCircle,
} from 'lucide-react';
import type { ExtractionJob, ExtractionConfig, ClaudeSession } from '../../shared/types';

interface JobCardProps {
  job: ExtractionJob;
}

function JobCard({ job }: JobCardProps) {
  const statusIcons = {
    pending: <Clock className="w-4 h-4 text-slate-400" />,
    processing: <Loader2 className="w-4 h-4 text-blue-400 animate-spin" />,
    completed: <CheckCircle className="w-4 h-4 text-green-400" />,
    failed: <XCircle className="w-4 h-4 text-red-400" />,
  };

  const statusColors = {
    pending: 'border-slate-600',
    processing: 'border-blue-500',
    completed: 'border-green-500',
    failed: 'border-red-500',
  };

  return (
    <div className={`p-4 bg-slate-800 rounded-lg border-l-4 ${statusColors[job.status]}`}>
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          {statusIcons[job.status]}
          <div>
            <p className="font-medium text-sm">Session: {job.session_id.substring(0, 8)}...</p>
            <p className="text-xs text-slate-400 mt-1">
              {job.messages_processed} messages processed • {job.memories_created} memories created
            </p>
          </div>
        </div>
        <span className="text-xs text-slate-500">
          {job.started_at && new Date(job.started_at).toLocaleTimeString()}
        </span>
      </div>
      {job.error && (
        <div className="mt-2 p-2 bg-red-500/10 rounded text-xs text-red-400">{job.error}</div>
      )}
    </div>
  );
}

export default function Extraction() {
  const [jobs, setJobs] = useState<ExtractionJob[]>([]);
  const [config, setConfig] = useState<ExtractionConfig | null>(null);
  const [sessions, setSessions] = useState<ClaudeSession[]>([]);
  const [loading, setLoading] = useState(true);
  const [configOpen, setConfigOpen] = useState(false);
  const [pendingConfig, setPendingConfig] = useState<ExtractionConfig | null>(null);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchJobs, 5000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    // Listen for extraction progress
    const unsubscribe = window.mycelicMemory.extraction.onProgress((job) => {
      setJobs((prev) => {
        const existing = prev.findIndex((j) => j.id === job.id);
        if (existing >= 0) {
          const updated = [...prev];
          updated[existing] = job;
          return updated;
        }
        return [job, ...prev];
      });
    });

    return unsubscribe;
  }, []);

  async function fetchData() {
    try {
      setLoading(true);
      const [jobsRes, configRes, sessionsRes] = await Promise.all([
        window.mycelicMemory.extraction.status(),
        window.mycelicMemory.extraction.getConfig(),
        window.mycelicMemory.claude.sessions().catch(() => []),
      ]);
      setJobs(jobsRes || []);
      setConfig(configRes);
      setPendingConfig(configRes);
      setSessions(sessionsRes || []);
    } catch (err) {
      console.error('Failed to fetch data:', err);
    } finally {
      setLoading(false);
    }
  }

  async function fetchJobs() {
    try {
      const jobsRes = await window.mycelicMemory.extraction.status();
      setJobs(jobsRes || []);
    } catch (err) {
      console.error('Failed to fetch jobs:', err);
    }
  }

  async function handleExtractSession(sessionId: string) {
    try {
      await window.mycelicMemory.extraction.start(sessionId);
      fetchJobs();
    } catch (err) {
      console.error('Failed to start extraction:', err);
    }
  }

  async function handleSaveConfig() {
    if (!pendingConfig) return;
    try {
      const updated = await window.mycelicMemory.extraction.updateConfig(pendingConfig);
      setConfig(updated);
      setConfigOpen(false);
    } catch (err) {
      console.error('Failed to save config:', err);
    }
  }

  const recentSessions = sessions.slice(0, 10);
  const pendingJobs = jobs.filter((j) => j.status === 'pending' || j.status === 'processing');
  const completedJobs = jobs.filter((j) => j.status === 'completed' || j.status === 'failed');

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="animate-spin w-8 h-8 border-4 border-primary-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  return (
    <div className="p-8 animate-fade-in">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Memory Extraction</h1>
        <div className="flex items-center gap-2">
          <button
            onClick={fetchJobs}
            className="p-2 hover:bg-slate-700 rounded-lg transition-colors"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
          <button
            onClick={() => setConfigOpen(true)}
            className="px-4 py-2 bg-slate-800 hover:bg-slate-700 rounded-lg transition-colors flex items-center gap-2"
          >
            <Settings className="w-4 h-4" />
            Configure
          </button>
        </div>
      </div>

      {/* Status Bar */}
      <div className="bg-slate-800 rounded-xl p-4 border border-slate-700 mb-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div
              className={`w-3 h-3 rounded-full ${
                config?.auto_extract ? 'bg-green-500 animate-pulse' : 'bg-slate-500'
              }`}
            />
            <span className="text-sm">
              Auto-extraction: {config?.auto_extract ? 'Enabled' : 'Disabled'}
            </span>
          </div>
          <div className="flex items-center gap-6 text-sm text-slate-400">
            <span>Poll interval: {(config?.poll_interval_ms || 0) / 1000}s</span>
            <span>Min message length: {config?.min_message_length}</span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Extract from Sessions */}
        <div className="bg-slate-800 rounded-xl border border-slate-700">
          <div className="p-4 border-b border-slate-700">
            <h2 className="font-semibold">Recent Sessions</h2>
            <p className="text-sm text-slate-400 mt-1">Click to extract memories from a session</p>
          </div>
          <div className="p-4 space-y-2 max-h-96 overflow-auto">
            {recentSessions.length > 0 ? (
              recentSessions.map((session) => (
                <div
                  key={session.id}
                  className="p-3 bg-slate-700/50 rounded-lg flex items-center justify-between"
                >
                  <div className="flex-1 min-w-0 mr-4">
                    <p className="text-sm truncate">
                      {session.summary || session.first_prompt || 'No summary'}
                    </p>
                    <p className="text-xs text-slate-400 mt-1">
                      {session.message_count} messages •{' '}
                      {new Date(session.created_at).toLocaleDateString()}
                    </p>
                  </div>
                  <button
                    onClick={() => handleExtractSession(session.id)}
                    className="p-2 bg-primary-500/20 text-primary-400 rounded-lg hover:bg-primary-500/30 transition-colors"
                  >
                    <Download className="w-4 h-4" />
                  </button>
                </div>
              ))
            ) : (
              <div className="text-center py-8 text-slate-500">
                <AlertCircle className="w-8 h-8 mx-auto mb-2 opacity-50" />
                <p className="text-sm">No sessions available</p>
                <p className="text-xs mt-1">Make sure claude-chat-stream is running</p>
              </div>
            )}
          </div>
        </div>

        {/* Job Queue */}
        <div className="bg-slate-800 rounded-xl border border-slate-700">
          <div className="p-4 border-b border-slate-700">
            <h2 className="font-semibold">Extraction Queue</h2>
            <p className="text-sm text-slate-400 mt-1">
              {pendingJobs.length} active • {completedJobs.length} completed
            </p>
          </div>
          <div className="p-4 space-y-3 max-h-96 overflow-auto">
            {jobs.length > 0 ? (
              jobs.map((job) => <JobCard key={job.id} job={job} />)
            ) : (
              <div className="text-center py-8 text-slate-500">
                <Clock className="w-8 h-8 mx-auto mb-2 opacity-50" />
                <p className="text-sm">No extraction jobs</p>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Config Modal */}
      {configOpen && pendingConfig && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-slate-800 rounded-xl border border-slate-700 w-full max-w-md">
            <div className="p-4 border-b border-slate-700">
              <h3 className="font-semibold">Extraction Configuration</h3>
            </div>
            <div className="p-4 space-y-4">
              <div className="flex items-center justify-between">
                <label className="text-sm">Auto-extract new messages</label>
                <button
                  onClick={() =>
                    setPendingConfig({ ...pendingConfig, auto_extract: !pendingConfig.auto_extract })
                  }
                  className={`w-12 h-6 rounded-full transition-colors relative ${
                    pendingConfig.auto_extract ? 'bg-primary-500' : 'bg-slate-600'
                  }`}
                >
                  <div
                    className={`absolute top-1 w-4 h-4 bg-white rounded-full transition-transform ${
                      pendingConfig.auto_extract ? 'translate-x-7' : 'translate-x-1'
                    }`}
                  />
                </button>
              </div>

              <div>
                <label className="text-sm text-slate-400">Poll interval (ms)</label>
                <input
                  type="number"
                  value={pendingConfig.poll_interval_ms}
                  onChange={(e) =>
                    setPendingConfig({
                      ...pendingConfig,
                      poll_interval_ms: parseInt(e.target.value) || 5000,
                    })
                  }
                  className="w-full mt-1 p-2 bg-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>

              <div>
                <label className="text-sm text-slate-400">Minimum message length</label>
                <input
                  type="number"
                  value={pendingConfig.min_message_length}
                  onChange={(e) =>
                    setPendingConfig({
                      ...pendingConfig,
                      min_message_length: parseInt(e.target.value) || 50,
                    })
                  }
                  className="w-full mt-1 p-2 bg-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>

              <div className="flex items-center justify-between">
                <label className="text-sm">Extract tool calls</label>
                <button
                  onClick={() =>
                    setPendingConfig({
                      ...pendingConfig,
                      extract_tool_calls: !pendingConfig.extract_tool_calls,
                    })
                  }
                  className={`w-12 h-6 rounded-full transition-colors relative ${
                    pendingConfig.extract_tool_calls ? 'bg-primary-500' : 'bg-slate-600'
                  }`}
                >
                  <div
                    className={`absolute top-1 w-4 h-4 bg-white rounded-full transition-transform ${
                      pendingConfig.extract_tool_calls ? 'translate-x-7' : 'translate-x-1'
                    }`}
                  />
                </button>
              </div>

              <div className="flex items-center justify-between">
                <label className="text-sm">Extract file operations</label>
                <button
                  onClick={() =>
                    setPendingConfig({
                      ...pendingConfig,
                      extract_file_operations: !pendingConfig.extract_file_operations,
                    })
                  }
                  className={`w-12 h-6 rounded-full transition-colors relative ${
                    pendingConfig.extract_file_operations ? 'bg-primary-500' : 'bg-slate-600'
                  }`}
                >
                  <div
                    className={`absolute top-1 w-4 h-4 bg-white rounded-full transition-transform ${
                      pendingConfig.extract_file_operations ? 'translate-x-7' : 'translate-x-1'
                    }`}
                  />
                </button>
              </div>
            </div>
            <div className="p-4 border-t border-slate-700 flex gap-2">
              <button
                onClick={() => setConfigOpen(false)}
                className="flex-1 py-2 px-4 bg-slate-700 rounded-lg hover:bg-slate-600 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleSaveConfig}
                className="flex-1 py-2 px-4 bg-primary-500 rounded-lg hover:bg-primary-600 transition-colors"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
