import { Routes, Route, NavLink, useLocation, useNavigate } from 'react-router-dom';
import {
  Brain,
  Search,
  MessageSquare,
  Network,
  Settings,
  Activity,
  Plus,
  PanelLeftClose,
  PanelLeftOpen,
  Database,
} from 'lucide-react';
import { useEffect, useState, useRef, useCallback } from 'react';
import Dashboard from './pages/Dashboard';
import MemoryBrowser from './pages/MemoryBrowser';
import ClaudeSessions from './pages/ClaudeSessions';
import KnowledgeGraph from './pages/KnowledgeGraph';
import SettingsPage from './pages/Settings';
import DataSources from './pages/DataSources';
import { ToastProvider } from './components/Toast';
import { ErrorBoundary } from './components/ErrorBoundary';
import { CommandPalette } from './components/CommandPalette';
import { CreateMemoryModal } from './components/CreateMemoryModal';
import type { HealthStatus } from '../shared/types';

interface NavItemProps {
  to: string;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  collapsed: boolean;
}

function NavItem({ to, icon: Icon, label, collapsed }: NavItemProps) {
  return (
    <NavLink
      to={to}
      title={collapsed ? label : undefined}
      className={({ isActive }) =>
        `flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
          collapsed ? 'justify-center' : ''
        } ${
          isActive
            ? 'bg-primary-500/20 text-primary-400'
            : 'text-slate-400 hover:bg-slate-700 hover:text-slate-200'
        }`
      }
    >
      <Icon className="w-5 h-5 shrink-0" />
      {!collapsed && label}
    </NavLink>
  );
}

function App() {
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const location = useLocation();
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Load sidebar collapsed state from settings
  useEffect(() => {
    window.mycelicMemory.settings?.get?.()
      .then((s: { sidebar_collapsed?: boolean }) => {
        if (s?.sidebar_collapsed) setSidebarCollapsed(true);
      })
      .catch(() => {});
  }, []);

  const toggleSidebar = () => {
    const next = !sidebarCollapsed;
    setSidebarCollapsed(next);
    window.mycelicMemory.settings?.update?.({ sidebar_collapsed: next }).catch(() => {});
  };

  // Global keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const mod = e.metaKey || e.ctrlKey;
      if (mod && e.key === 'k') {
        e.preventDefault();
        setCommandPaletteOpen(true);
      }
      if (mod && e.key === 'n') {
        e.preventDefault();
        setCreateModalOpen(true);
      }
      if (mod && e.key === 'b') {
        e.preventDefault();
        toggleSidebar();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [sidebarCollapsed]);

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

    const startPolling = async () => {
      const connected = await checkHealth();

      if (cancelled) return;

      if (intervalRef.current) clearInterval(intervalRef.current);

      if (connected) {
        intervalRef.current = setInterval(async () => {
          const stillConnected = await checkHealth();
          if (!stillConnected && !cancelled) {
            if (intervalRef.current) clearInterval(intervalRef.current);
            startPolling();
          }
        }, 30000);
      } else {
        intervalRef.current = setInterval(async () => {
          const nowConnected = await checkHealth();
          if (nowConnected && !cancelled) {
            if (intervalRef.current) clearInterval(intervalRef.current);
            startPolling();
          }
        }, 3000);
      }
    };

    startPolling();

    let cleanupServiceListener: (() => void) | undefined;
    if (window.mycelicMemory.services?.onStatusUpdate) {
      cleanupServiceListener = window.mycelicMemory.services.onStatusUpdate((status) => {
        if (status.backend.running) {
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
      <aside
        className={`${
          sidebarCollapsed ? 'w-16' : 'w-64'
        } bg-slate-800 border-r border-slate-700 flex flex-col transition-all duration-200`}
      >
        {/* Logo */}
        <div className="p-4 border-b border-slate-700">
          <div className={`flex items-center ${sidebarCollapsed ? 'justify-center' : 'gap-3'}`}>
            <div className="w-8 h-8 bg-gradient-to-br from-primary-500 to-mycelium-500 rounded-xl flex items-center justify-center shrink-0">
              <Brain className="w-5 h-5 text-white" />
            </div>
            {!sidebarCollapsed && (
              <div className="min-w-0">
                <h1 className="font-semibold text-sm text-white truncate">MycelicMemory</h1>
                <p className="text-xs text-slate-400">Desktop</p>
              </div>
            )}
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-2 space-y-1">
          <NavItem to="/" icon={Activity} label="Dashboard" collapsed={sidebarCollapsed} />
          <NavItem to="/memories" icon={Search} label="Memories" collapsed={sidebarCollapsed} />
          <NavItem to="/sessions" icon={MessageSquare} label="Sessions" collapsed={sidebarCollapsed} />
          <NavItem to="/graph" icon={Network} label="Graph" collapsed={sidebarCollapsed} />
          <NavItem to="/sources" icon={Database} label="Sources" collapsed={sidebarCollapsed} />

          <div className="pt-3 mt-3 border-t border-slate-700">
            <NavItem to="/settings" icon={Settings} label="Settings" collapsed={sidebarCollapsed} />
          </div>
        </nav>

        {/* Quick Actions */}
        <div className="p-2 space-y-1 border-t border-slate-700">
          <button
            onClick={() => setCreateModalOpen(true)}
            title="Create Memory (Ctrl+N)"
            className={`w-full flex items-center gap-3 px-4 py-2.5 rounded-lg text-slate-400 hover:bg-slate-700 hover:text-slate-200 transition-colors ${
              sidebarCollapsed ? 'justify-center' : ''
            }`}
          >
            <Plus className="w-4 h-4 shrink-0" />
            {!sidebarCollapsed && <span className="text-sm">New Memory</span>}
          </button>
          <button
            onClick={toggleSidebar}
            title={sidebarCollapsed ? 'Expand sidebar (Ctrl+B)' : 'Collapse sidebar (Ctrl+B)'}
            className={`w-full flex items-center gap-3 px-4 py-2.5 rounded-lg text-slate-400 hover:bg-slate-700 hover:text-slate-200 transition-colors ${
              sidebarCollapsed ? 'justify-center' : ''
            }`}
          >
            {sidebarCollapsed ? (
              <PanelLeftOpen className="w-4 h-4 shrink-0" />
            ) : (
              <PanelLeftClose className="w-4 h-4 shrink-0" />
            )}
            {!sidebarCollapsed && <span className="text-sm">Collapse</span>}
          </button>
        </div>

        {/* Status */}
        <div className="p-3 border-t border-slate-700">
          <div className={`flex items-center gap-2 text-sm ${sidebarCollapsed ? 'justify-center' : ''}`}>
            <div
              className={`w-2 h-2 rounded-full shrink-0 ${
                isConnected ? 'bg-green-500 animate-pulse' : 'bg-red-500'
              }`}
            />
            {!sidebarCollapsed && (
              <span className="text-slate-400 text-xs">
                {isConnected ? 'Connected' : 'Disconnected'}
              </span>
            )}
          </div>
          {!sidebarCollapsed && health && (
            <div className="mt-2 flex gap-1.5">
              <span
                className={`text-xs px-1.5 py-0.5 rounded ${
                  health.ollama ? 'bg-green-500/20 text-green-400' : 'bg-slate-700 text-slate-500'
                }`}
              >
                Ollama
              </span>
              <span
                className={`text-xs px-1.5 py-0.5 rounded ${
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
        <ErrorBoundary>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/memories" element={<MemoryBrowser />} />
            <Route path="/sessions" element={<ClaudeSessions />} />
            <Route path="/graph" element={<KnowledgeGraph />} />
            <Route path="/sources" element={<DataSources />} />
            <Route path="/settings" element={<SettingsPage />} />
          </Routes>
        </ErrorBoundary>
      </main>

      {/* Global Overlays */}
      <ToastProvider />
      <CommandPalette open={commandPaletteOpen} onClose={() => setCommandPaletteOpen(false)} />
      <CreateMemoryModal
        open={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
      />
    </div>
  );
}

export default App;
