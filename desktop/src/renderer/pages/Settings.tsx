import { useState, useEffect } from 'react';
import {
  Settings,
  Server,
  Database,
  Brain,
  Save,
  RefreshCw,
  CheckCircle,
  AlertCircle,
  Folder,
} from 'lucide-react';
import type { AppSettings, HealthStatus } from '../../shared/types';

interface SettingsSectionProps {
  title: string;
  icon: React.ComponentType<{ className?: string }>;
  children: React.ReactNode;
}

function SettingsSection({ title, icon: Icon, children }: SettingsSectionProps) {
  return (
    <div className="bg-slate-800 rounded-xl border border-slate-700">
      <div className="p-4 border-b border-slate-700 flex items-center gap-3">
        <Icon className="w-5 h-5 text-primary-400" />
        <h2 className="font-semibold">{title}</h2>
      </div>
      <div className="p-4 space-y-4">{children}</div>
    </div>
  );
}

interface InputFieldProps {
  label: string;
  value: string | number;
  onChange: (value: string) => void;
  type?: 'text' | 'number' | 'url';
  placeholder?: string;
  hint?: string;
}

function InputField({ label, value, onChange, type = 'text', placeholder, hint }: InputFieldProps) {
  return (
    <div>
      <label className="text-sm text-slate-400">{label}</label>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full mt-1 p-2 bg-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
      />
      {hint && <p className="text-xs text-slate-500 mt-1">{hint}</p>}
    </div>
  );
}

interface ToggleFieldProps {
  label: string;
  value: boolean;
  onChange: (value: boolean) => void;
  description?: string;
}

function ToggleField({ label, value, onChange, description }: ToggleFieldProps) {
  return (
    <div className="flex items-center justify-between">
      <div>
        <label className="text-sm">{label}</label>
        {description && <p className="text-xs text-slate-500 mt-0.5">{description}</p>}
      </div>
      <button
        onClick={() => onChange(!value)}
        className={`w-12 h-6 rounded-full transition-colors relative ${
          value ? 'bg-primary-500' : 'bg-slate-600'
        }`}
      >
        <div
          className={`absolute top-1 w-4 h-4 bg-white rounded-full transition-transform ${
            value ? 'translate-x-7' : 'translate-x-1'
          }`}
        />
      </button>
    </div>
  );
}

export default function SettingsPage() {
  const [settings, setSettings] = useState<AppSettings | null>(null);
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [testing, setTesting] = useState(false);

  useEffect(() => {
    fetchSettings();
    testConnections();
  }, []);

  async function fetchSettings() {
    try {
      setLoading(true);
      const response = await window.mycelicMemory.settings.get();
      setSettings(response);
    } catch (err) {
      console.error('Failed to fetch settings:', err);
    } finally {
      setLoading(false);
    }
  }

  async function testConnections() {
    try {
      setTesting(true);
      const response = await window.mycelicMemory.stats.health();
      setHealth(response);
    } catch (err) {
      console.error('Failed to test connections:', err);
    } finally {
      setTesting(false);
    }
  }

  async function handleSave() {
    if (!settings) return;
    try {
      setSaving(true);
      await window.mycelicMemory.settings.update(settings);
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch (err) {
      console.error('Failed to save settings:', err);
    } finally {
      setSaving(false);
    }
  }

  function updateSetting<K extends keyof AppSettings>(key: K, value: AppSettings[K]) {
    if (!settings) return;
    setSettings({ ...settings, [key]: value });
  }

  if (loading || !settings) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="animate-spin w-8 h-8 border-4 border-primary-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  return (
    <div className="p-8 animate-fade-in max-w-4xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Settings</h1>
        <div className="flex items-center gap-2">
          <button
            onClick={testConnections}
            disabled={testing}
            className="px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg transition-colors flex items-center gap-2 disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${testing ? 'animate-spin' : ''}`} />
            Test Connections
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-4 py-2 bg-primary-500 hover:bg-primary-600 rounded-lg transition-colors flex items-center gap-2 disabled:opacity-50"
          >
            {saved ? (
              <>
                <CheckCircle className="w-4 h-4" />
                Saved
              </>
            ) : (
              <>
                <Save className="w-4 h-4" />
                Save Changes
              </>
            )}
          </button>
        </div>
      </div>

      {/* Connection Status */}
      <div className="bg-slate-800 rounded-xl border border-slate-700 p-4 mb-6">
        <h3 className="font-semibold mb-3">Connection Status</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="flex items-center gap-2">
            {health?.api ? (
              <CheckCircle className="w-4 h-4 text-green-400" />
            ) : (
              <AlertCircle className="w-4 h-4 text-red-400" />
            )}
            <span className="text-sm">MycelicMemory API</span>
          </div>
          <div className="flex items-center gap-2">
            {health?.ollama ? (
              <CheckCircle className="w-4 h-4 text-green-400" />
            ) : (
              <AlertCircle className="w-4 h-4 text-amber-400" />
            )}
            <span className="text-sm">Ollama</span>
          </div>
          <div className="flex items-center gap-2">
            {health?.qdrant ? (
              <CheckCircle className="w-4 h-4 text-green-400" />
            ) : (
              <AlertCircle className="w-4 h-4 text-amber-400" />
            )}
            <span className="text-sm">Qdrant</span>
          </div>
          <div className="flex items-center gap-2">
            {health?.database ? (
              <CheckCircle className="w-4 h-4 text-green-400" />
            ) : (
              <AlertCircle className="w-4 h-4 text-red-400" />
            )}
            <span className="text-sm">Database</span>
          </div>
        </div>
      </div>

      <div className="space-y-6">
        {/* MycelicMemory API */}
        <SettingsSection title="MycelicMemory API" icon={Server}>
          <div className="grid grid-cols-2 gap-4">
            <InputField
              label="API URL"
              value={settings.api_url}
              onChange={(v) => updateSetting('api_url', v)}
              type="url"
              placeholder="http://localhost"
            />
            <InputField
              label="Port"
              value={settings.api_port}
              onChange={(v) => updateSetting('api_port', parseInt(v) || 3099)}
              type="number"
              placeholder="3099"
            />
          </div>
        </SettingsSection>

        {/* Ollama */}
        <SettingsSection title="Ollama Configuration" icon={Brain}>
          <InputField
            label="Base URL"
            value={settings.ollama_base_url}
            onChange={(v) => updateSetting('ollama_base_url', v)}
            type="url"
            placeholder="http://localhost:11434"
          />
          <div className="grid grid-cols-2 gap-4">
            <InputField
              label="Embedding Model"
              value={settings.ollama_embedding_model}
              onChange={(v) => updateSetting('ollama_embedding_model', v)}
              placeholder="nomic-embed-text"
              hint="Model for generating embeddings"
            />
            <InputField
              label="Chat Model"
              value={settings.ollama_chat_model}
              onChange={(v) => updateSetting('ollama_chat_model', v)}
              placeholder="llama3.2"
              hint="Model for AI analysis"
            />
          </div>
        </SettingsSection>

        {/* Qdrant */}
        <SettingsSection title="Qdrant Vector Database" icon={Database}>
          <ToggleField
            label="Enable Qdrant"
            value={settings.qdrant_enabled}
            onChange={(v) => updateSetting('qdrant_enabled', v)}
            description="Use Qdrant for semantic search (requires running Qdrant server)"
          />
          <InputField
            label="Qdrant URL"
            value={settings.qdrant_url}
            onChange={(v) => updateSetting('qdrant_url', v)}
            type="url"
            placeholder="http://localhost:6333"
          />
        </SettingsSection>

        {/* Claude Chat Stream */}
        <SettingsSection title="Claude Chat Stream" icon={Folder}>
          <InputField
            label="Database Path"
            value={settings.claude_stream_db_path}
            onChange={(v) => updateSetting('claude_stream_db_path', v)}
            placeholder="Path to chats.db"
            hint="Location of the claude-chat-stream SQLite database"
          />
        </SettingsSection>

        {/* UI */}
        <SettingsSection title="Interface" icon={Settings}>
          <div>
            <label className="text-sm text-slate-400">Theme</label>
            <select
              value={settings.theme}
              onChange={(e) => updateSetting('theme', e.target.value as 'dark' | 'light' | 'system')}
              className="w-full mt-1 p-2 bg-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="dark">Dark</option>
              <option value="light">Light</option>
              <option value="system">System</option>
            </select>
          </div>
          <ToggleField
            label="Collapse Sidebar"
            value={settings.sidebar_collapsed}
            onChange={(v) => updateSetting('sidebar_collapsed', v)}
            description="Start with sidebar collapsed"
          />
        </SettingsSection>
      </div>

      {/* App Info */}
      <div className="mt-8 text-center text-sm text-slate-500">
        <p>MycelicMemory Desktop</p>
        <p className="mt-1">Platform: {window.mycelicMemory.app.getPlatform()}</p>
      </div>
    </div>
  );
}
