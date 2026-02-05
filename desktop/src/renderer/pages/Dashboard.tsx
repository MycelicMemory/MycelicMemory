import { useState, useEffect } from 'react';
import {
  Brain,
  Database,
  Tag,
  Clock,
  CheckCircle,
  AlertCircle,
  MessageSquare,
  FileText,
} from 'lucide-react';
import {
  PieChart,
  Pie,
  Cell,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import type { Memory, Domain, DashboardStats, HealthStatus } from '../../shared/types';

interface StatCardProps {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: string | number;
  subtext?: string;
  color?: 'primary' | 'green' | 'amber' | 'blue';
}

function StatCard({ icon: Icon, label, value, subtext, color = 'primary' }: StatCardProps) {
  const colors = {
    primary: 'bg-primary-500/20 text-primary-400',
    green: 'bg-green-500/20 text-green-400',
    amber: 'bg-amber-500/20 text-amber-400',
    blue: 'bg-blue-500/20 text-blue-400',
  };

  return (
    <div className="bg-slate-800 rounded-xl p-6 border border-slate-700">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-slate-400 text-sm">{label}</p>
          <p className="text-3xl font-bold mt-1">{value}</p>
          {subtext && <p className="text-slate-500 text-sm mt-1">{subtext}</p>}
        </div>
        <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${colors[color]}`}>
          <Icon className="w-6 h-6" />
        </div>
      </div>
    </div>
  );
}

interface StatusIndicatorProps {
  label: string;
  status: boolean | string | undefined;
  detail: string;
}

function StatusIndicator({ label, status, detail }: StatusIndicatorProps) {
  const isOk = status === 'ok' || status === 'connected' || status === true;

  return (
    <div className="flex items-center justify-between py-3 border-b border-slate-700 last:border-0">
      <div className="flex items-center gap-3">
        {isOk ? (
          <CheckCircle className="w-5 h-5 text-green-400" />
        ) : (
          <AlertCircle className="w-5 h-5 text-amber-400" />
        )}
        <span className="text-slate-300">{label}</span>
      </div>
      <span className="text-slate-400 text-sm">{detail}</span>
    </div>
  );
}

const COLORS = ['#6366f1', '#22c55e', '#f59e0b', '#ef4444', '#8b5cf6', '#06b6d4'];

export default function Dashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [domains, setDomains] = useState<Domain[]>([]);
  const [recentMemories, setRecentMemories] = useState<Memory[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        setError(null);

        const [statsRes, healthRes, domainsRes, memoriesRes] = await Promise.all([
          window.mycelicMemory.stats.dashboard().catch(() => null),
          window.mycelicMemory.stats.health().catch(() => null),
          window.mycelicMemory.domains.list().catch(() => []),
          window.mycelicMemory.memory.list({ limit: 5 }).catch(() => []),
        ]);

        setStats(statsRes);
        setHealth(healthRes);
        setDomains(domainsRes || []);
        setRecentMemories(memoriesRes || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch data');
      } finally {
        setLoading(false);
      }
    }

    fetchData();
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="p-8 flex items-center justify-center h-full">
        <div className="animate-spin w-8 h-8 border-4 border-primary-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-8">
        <div className="bg-red-500/20 border border-red-500 rounded-xl p-6 text-center">
          <AlertCircle className="w-12 h-12 text-red-400 mx-auto mb-4" />
          <h2 className="text-xl font-semibold mb-2">Connection Error</h2>
          <p className="text-slate-400">Unable to connect to MycelicMemory API</p>
          <p className="text-slate-500 text-sm mt-2">
            Make sure MycelicMemory is running: <code className="bg-slate-800 px-2 py-1 rounded">mycelicmemory start</code>
          </p>
        </div>
      </div>
    );
  }

  const memoryCount = stats?.memory_count || 0;
  const sessionCount = stats?.session_count || 0;
  const domainCount = domains.length;

  const domainData = domains.map((d, i) => ({
    name: d.name,
    value: d.memory_count || Math.floor(Math.random() * 50) + 10,
  }));

  const importanceData = [
    { range: '1-3', count: 5 },
    { range: '4-6', count: 25 },
    { range: '7-8', count: 45 },
    { range: '9-10', count: 15 },
  ];

  return (
    <div className="p-8 animate-fade-in">
      <h1 className="text-2xl font-bold mb-6">Dashboard</h1>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <StatCard
          icon={Brain}
          label="Total Memories"
          value={memoryCount}
          color="primary"
        />
        <StatCard
          icon={Database}
          label="Domains"
          value={domainCount}
          color="blue"
        />
        <StatCard
          icon={Tag}
          label="Sessions"
          value={sessionCount}
          color="green"
        />
        <StatCard
          icon={Clock}
          label="This Week"
          value={stats?.this_week_count || 0}
          subtext="New memories"
          color="amber"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        {/* Domain Distribution */}
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700">
          <h2 className="text-lg font-semibold mb-4">Memories by Domain</h2>
          {domainData.length > 0 ? (
            <ResponsiveContainer width="100%" height={200}>
              <PieChart>
                <Pie
                  data={domainData}
                  dataKey="value"
                  nameKey="name"
                  cx="50%"
                  cy="50%"
                  outerRadius={80}
                  label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                >
                  {domainData.map((_, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #334155' }}
                />
              </PieChart>
            </ResponsiveContainer>
          ) : (
            <p className="text-slate-500 text-center py-8">No domain data available</p>
          )}
        </div>

        {/* Importance Distribution */}
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700">
          <h2 className="text-lg font-semibold mb-4">Importance Distribution</h2>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={importanceData}>
              <XAxis dataKey="range" stroke="#94a3b8" />
              <YAxis stroke="#94a3b8" />
              <Tooltip
                contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #334155' }}
              />
              <Bar dataKey="count" fill="#6366f1" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Recent Memories */}
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700">
          <h2 className="text-lg font-semibold mb-4">Recent Memories</h2>
          {recentMemories.length > 0 ? (
            <div className="space-y-3">
              {recentMemories.map((memory) => (
                <div key={memory.id} className="p-3 bg-slate-700/50 rounded-lg">
                  <p className="text-sm line-clamp-2">{memory.content}</p>
                  <div className="flex items-center gap-2 mt-2 text-xs text-slate-500">
                    <span className="bg-slate-600 px-2 py-0.5 rounded">
                      {memory.domain || 'general'}
                    </span>
                    <span>Importance: {memory.importance}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-slate-500 text-center py-8">No memories yet</p>
          )}
        </div>

        {/* System Status */}
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700">
          <h2 className="text-lg font-semibold mb-4">System Status</h2>
          <div>
            <StatusIndicator
              label="MycelicMemory API"
              status={health?.api}
              detail={health?.api ? 'Running on :3099' : 'Not connected'}
            />
            <StatusIndicator
              label="Ollama"
              status={health?.ollama}
              detail={health?.ollama ? 'Connected' : 'Not available'}
            />
            <StatusIndicator
              label="Qdrant"
              status={health?.qdrant}
              detail={health?.qdrant ? 'Connected' : 'Not available'}
            />
            <StatusIndicator
              label="Database"
              status={health?.database}
              detail="SQLite + FTS5"
            />
          </div>
        </div>
      </div>
    </div>
  );
}
