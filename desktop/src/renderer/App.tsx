import { Routes, Route, NavLink, useLocation } from 'react-router-dom';
import {
  Brain,
  Search,
  MessageSquare,
  Network,
  Download,
  Settings,
  Activity,
  Database,
} from 'lucide-react';
import { useEffect, useState, useRef, useCallback } from 'react';
import Dashboard from './pages/Dashboard';
import MemoryBrowser from './pages/MemoryBrowser';
import ClaudeSessions from './pages/ClaudeSessions';
import KnowledgeGraph from './pages/KnowledgeGraph';
import Extraction from './pages/Extraction';
import DataSources from './pages/DataSources';
import SettingsPage from './pages/Settings';
import type { HealthStatus } from '../shared/types';

interface NavItemProps {
  to: string;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
}

function NavItem({ to, icon: Icon, label }: NavItemProps) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
          isActive
            ? 'bg-primary-500/20 text-primary-400'
            : 'text-slate-400 hover:bg-slate-700 hover:text-slate-200'
        }`
      }
    >
      <Icon className="w-5 h-5" />
      {label}
    </NavLink>
  );
}

function App() {
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const location = useLocation();
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const checkHealth = useCallback(async () => {
    try {
      const status = await window.mycelicMemory.stats.health();
      setHealth(status);
      return status.api;
    } catch {
      setHealth({ api: false, ollama: false, qdrant: false, database: false });
      return false;
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    // Aggressive initial polling: every 3s until connected, then every 30s
    const startPolling = async () => {
      const connected = await checkHealth();

      if (cancelled) return;

      if (intervalRef.current) clearInterval(intervalRef.current);

      if (connected) {
        // Connected — slow poll to detect disconnects
        intervalRef.current = setInterval(async () => {
          const stillConnected = await checkHealth();
          if (!stillConnected && !cancelled) {
            // Lost connection — switch to fast polling
            if (intervalRef.current) clearInterval(intervalRef.current);
            startPolling();
          }
        }, 30000);
      } else {
        // Disconnected — fast poll to detect connection
        intervalRef.current = setInterval(async () => {
          const nowConnected = await checkHealth();
          if (nowConnected && !cancelled) {
            // Connected — switch to slow polling
            if (intervalRef.current) clearInterval(intervalRef.current);
            startPolling();
          }
        }, 3000);
      }
    };

    startPolling();

    // Listen for service status updates (from ServiceManager via IPC)
    let cleanupServiceListener: (() => void) | undefined;
    if (window.mycelicMemory.services?.onStatusUpdate) {
      cleanupServiceListener = window.mycelicMemory.services.onStatusUpdate((status) => {
        if (status.backend.running) {
          // Backend just came up — refresh health immediately
          checkHealth().then((connected) => {
            if (connected && !cancelled) {
              if (intervalRef.current) clearInterval(intervalRef.current);
              startPolling();
            }
          });
        }
      });
    }

    return () => {
      cancelled = true;
      if (intervalRef.current) clearInterval(intervalRef.current);
      cleanupServiceListener?.();
    };
  }, [checkHealth]);

  const isConnected = health?.api;

  return (
    <div className="min-h-screen flex bg-slate-900">
      {/* Sidebar */}
      <aside className="w-64 bg-slate-800 border-r border-slate-700 flex flex-col">
        {/* Logo */}
        <div className="p-6 border-b border-slate-700">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-gradient-to-br from-primary-500 to-mycelium-500 rounded-xl flex items-center justify-center">
              <Brain className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="font-semibold text-lg text-white">MycelicMemory</h1>
              <p className="text-xs text-slate-400">Desktop</p>
            </div>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-4 space-y-1">
          <NavItem to="/" icon={Activity} label="Dashboard" />
          <NavItem to="/memories" icon={Search} label="Memory Browser" />
          <NavItem to="/sessions" icon={MessageSquare} label="Claude Sessions" />
          <NavItem to="/graph" icon={Network} label="Knowledge Graph" />
          <NavItem to="/extraction" icon={Download} label="Extraction" />
          <NavItem to="/sources" icon={Database} label="Data Sources" />

          <div className="pt-4 mt-4 border-t border-slate-700">
            <NavItem to="/settings" icon={Settings} label="Settings" />
          </div>
        </nav>

        {/* Status */}
        <div className="p-4 border-t border-slate-700">
          <div className="flex items-center gap-2 text-sm">
            <div
              className={`w-2 h-2 rounded-full ${
                isConnected ? 'bg-green-500 animate-pulse' : 'bg-red-500'
              }`}
            />
            <span className="text-slate-400">
              {isConnected ? 'Connected to API' : 'Disconnected'}
            </span>
          </div>
          {health && (
            <div className="mt-2 flex gap-2">
              <span
                className={`text-xs px-2 py-0.5 rounded ${
                  health.ollama ? 'bg-green-500/20 text-green-400' : 'bg-slate-700 text-slate-500'
                }`}
              >
                Ollama
              </span>
              <span
                className={`text-xs px-2 py-0.5 rounded ${
                  health.qdrant ? 'bg-green-500/20 text-green-400' : 'bg-slate-700 text-slate-500'
                }`}
              >
                Qdrant
              </span>
            </div>
          )}
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-auto">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/memories" element={<MemoryBrowser />} />
          <Route path="/sessions" element={<ClaudeSessions />} />
          <Route path="/graph" element={<KnowledgeGraph />} />
          <Route path="/extraction" element={<Extraction />} />
          <Route path="/sources" element={<DataSources />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Routes>
      </main>
    </div>
  );
}

export default App;
