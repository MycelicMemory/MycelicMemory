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
  ChevronsUpDown,
  Loader2,
  Sun,
  Moon,
} from 'lucide-react';
import { useEffect, useState, useRef, useCallback } from 'react';
import toast from 'react-hot-toast';
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
import type { HealthStatus, DatabaseInfo } from '../shared/types';

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
  const [theme, setTheme] = useState<'dark' | 'light'>('dark');
  const [commandPaletteOpen, setCommandPaletteOpen] = useState(false);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const location = useLocation();
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Database state
  const [databases, setDatabases] = useState<DatabaseInfo[]>([]);
  const [activeDbName, setActiveDbName] = useState<string>('default');
  const [switching, setSwitching] = useState(false);
  const [dbSwitchKey, setDbSwitchKey] = useState(0);

  // Load sidebar collapsed state and theme from settings
  useEffect(() => {
    window.mycelicMemory.settings?.get?.()
      .then((s: { sidebar_collapsed?: boolean; theme?: string }) => {
        if (s?.sidebar_collapsed) setSidebarCollapsed(true);
        if (s?.theme === 'light') {
          setTheme('light');
          document.documentElement.classList.add('light-mode');
        }
      })
      .catch(() => {});
  }, []);

  // Fetch database list on mount and when API becomes available
  const fetchDatabases = useCallback(async () => {
    try {
      const dbs = await window.mycelicMemory.databases.list();
      if (Array.isArray(dbs)) {
        setDatabases(dbs);
        const active = dbs.find((d) => d.is_active);
        if (active) setActiveDbName(active.name);
      }
    } catch {
      // API not available yet — will retry on next health check
    }
  }, []);

  useEffect(() => {
    fetchDatabases();
  }, [fetchDatabases]);

  // Listen for db-switched events from Settings page (or anywhere else)
  useEffect(() => {
    const handler = async (e: Event) => {
      const detail = (e as CustomEvent).detail;
      if (detail?.name) {
        // Refresh DB list to get authoritative is_active state
        try {
          const dbs = await window.mycelicMemory.databases.list();
          if (Array.isArray(dbs)) {
            setDatabases(dbs);
            const active = dbs.find((d: DatabaseInfo) => d.is_active);
            setActiveDbName(active?.name || detail.name);
          } else {
            setActiveDbName(detail.name);
          }
        } catch {
          setActiveDbName(detail.name);
        }
        // Force re-mount of all route components to fetch from the new database
        setDbSwitchKey((prev) => prev + 1);
      }
    };
    window.addEventListener('db-switched', handler);
    return () => window.removeEventListener('db-switched', handler);
  }, []);

  const handleSwitchDatabase = useCallback(async (name: string) => {
    if (name === activeDbName || switching) return;
    setSwitching(true);
    try {
      await window.mycelicMemory.databases.switch(name);

      // Refresh database list and verify the switch took effect
      const dbs = await window.mycelicMemory.databases.list();
      if (Array.isArray(dbs)) {
        setDatabases(dbs);
        const active = dbs.find((d: DatabaseInfo) => d.is_active);
        if (active && active.name !== activeDbName) {
          setActiveDbName(active.name);
          setDbSwitchKey((prev) => prev + 1);
          toast.success(`Switched to "${active.name}"`);
        } else {
          // Backend didn't actually change the active database
          toast.error(`Switch to "${name}" failed — database may not exist`);
        }
      } else {
        // Couldn't verify — apply optimistically
        setActiveDbName(name);
        setDbSwitchKey((prev) => prev + 1);
        toast.success(`Switched to "${name}"`);
      }
    } catch (err: any) {
      toast.error(err?.message || 'Failed to switch database');
    } finally {
      setSwitching(false);
    }
  }, [activeDbName, switching]);

  const toggleSidebar = () => {
    const next = !sidebarCollapsed;
    setSidebarCollapsed(next);
    window.mycelicMemory.settings?.update?.({ sidebar_collapsed: next }).catch(() => {});
  };

  const toggleTheme = () => {
    const next = theme === 'dark' ? 'light' : 'dark';
    setTheme(next);
    if (next === 'light') {
      document.documentElement.classList.add('light-mode');
    } else {
      document.documentElement.classList.remove('light-mode');
    }
    window.mycelicMemory.settings?.update?.({ theme: next }).catch(() => {});
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

      // Refresh DB list when we first connect
      if (connected) fetchDatabases();

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
              fetchDatabases();
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
  }, [checkHealth, fetchDatabases]);

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

        {/* Database Selector */}
        <div className={`border-b border-slate-700 ${sidebarCollapsed ? 'p-2' : 'px-3 py-2'}`}>
          {sidebarCollapsed ? (
            <button
              onClick={toggleSidebar}
              title={`Database: ${activeDbName}`}
              className="w-full flex justify-center p-2 rounded-lg text-slate-400 hover:bg-slate-700 hover:text-slate-200 transition-colors"
            >
              <Database className="w-4 h-4" />
            </button>
          ) : (
            <div>
              <label className="text-[10px] text-slate-500 uppercase tracking-wider font-medium px-1">
                Database
              </label>
              <div className="relative mt-1">
                <select
                  value={activeDbName}
                  onChange={(e) => handleSwitchDatabase(e.target.value)}
                  disabled={switching || databases.length === 0}
                  className="w-full appearance-none px-3 py-2 pr-8 bg-slate-900 border border-slate-600 rounded-lg text-sm text-slate-200 focus:outline-none focus:ring-1 focus:ring-primary-500 focus:border-primary-500 disabled:opacity-50 cursor-pointer hover:border-slate-500 transition-colors"
                >
                  {databases.length === 0 ? (
                    <option value="default">default</option>
                  ) : (
                    databases.map((db) => (
                      <option key={db.name} value={db.name}>
                        {db.name}
                      </option>
                    ))
                  )}
                </select>
                <div className="absolute right-2 top-1/2 -translate-y-1/2 pointer-events-none text-slate-400">
                  {switching ? (
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  ) : (
                    <ChevronsUpDown className="w-3.5 h-3.5" />
                  )}
                </div>
              </div>
            </div>
          )}
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
            onClick={toggleTheme}
            title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
            className={`w-full flex items-center gap-3 px-4 py-2.5 rounded-lg text-slate-400 hover:bg-slate-700 hover:text-slate-200 transition-colors ${
              sidebarCollapsed ? 'justify-center' : ''
            }`}
          >
            {theme === 'dark' ? (
              <Sun className="w-4 h-4 shrink-0" />
            ) : (
              <Moon className="w-4 h-4 shrink-0" />
            )}
            {!sidebarCollapsed && <span className="text-sm">{theme === 'dark' ? 'Light Mode' : 'Dark Mode'}</span>}
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

      {/* Main Content — key forces re-mount of all routes on DB switch */}
      <main className="flex-1 overflow-auto">
        <ErrorBoundary key={dbSwitchKey}>
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
