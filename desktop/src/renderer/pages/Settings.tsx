import { useState, useEffect, useCallback } from 'react';
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
  Plug,
  Copy,
  Check,
  Info,
  ChevronDown,
} from 'lucide-react';
import type { AppSettings, HealthStatus, ServiceStatus } from '../../shared/types';

// ── Shared UI Components ─────────────────────────────────────────

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

// ── CodeBlock with Copy Button ───────────────────────────────────

function CodeBlock({ children, language }: { children: string; language?: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(children);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback for environments without clipboard API
      const textarea = document.createElement('textarea');
      textarea.value = children;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }, [children]);

  return (
    <div className="relative group">
      <pre className="bg-slate-900 border border-slate-600 rounded-lg p-3 text-xs text-slate-300 overflow-x-auto font-mono whitespace-pre">
        {language && (
          <span className="text-slate-500 text-[10px] absolute top-1 right-10">{language}</span>
        )}
        {children}
      </pre>
      <button
        onClick={handleCopy}
        className="absolute top-2 right-2 p-1.5 bg-slate-700 hover:bg-slate-600 rounded transition-colors opacity-0 group-hover:opacity-100"
        title="Copy to clipboard"
      >
        {copied ? (
          <Check className="w-3.5 h-3.5 text-green-400" />
        ) : (
          <Copy className="w-3.5 h-3.5 text-slate-400" />
        )}
      </button>
    </div>
  );
}

// ── Platform Toggle ──────────────────────────────────────────────

type Platform = 'win32' | 'darwin' | 'linux';

function PlatformToggle({
  value,
  onChange,
}: {
  value: Platform;
  onChange: (p: Platform) => void;
}) {
  const platforms: { id: Platform; label: string }[] = [
    { id: 'win32', label: 'Windows' },
    { id: 'darwin', label: 'macOS' },
    { id: 'linux', label: 'Linux' },
  ];

  return (
    <div className="flex gap-1 bg-slate-900 rounded-lg p-1 w-fit">
      {platforms.map((p) => (
        <button
          key={p.id}
          onClick={() => onChange(p.id)}
          className={`px-3 py-1 text-xs rounded-md transition-colors ${
            value === p.id
              ? 'bg-primary-500 text-white'
              : 'text-slate-400 hover:text-slate-300'
          }`}
        >
          {p.label}
        </button>
      ))}
    </div>
  );
}

// ── MCP Setup Guide Section ──────────────────────────────────────

function MCPSetupGuide() {
  const detectedPlatform = (typeof window !== 'undefined' && window.mycelicMemory?.app?.getPlatform?.() || 'win32') as Platform;
  const [platform, setPlatform] = useState<Platform>(detectedPlatform);

  const binaryName = platform === 'win32' ? 'mycelicmemory.exe' : 'mycelicmemory';

  const findBinaryCommand: Record<Platform, string> = {
    win32: `where ${binaryName}\n# or check: %LOCALAPPDATA%\\mycelicmemory\\${binaryName}`,
    darwin: `which ${binaryName}\n# or check: ~/.local/bin/${binaryName}`,
    linux: `which ${binaryName}\n# or check: ~/.local/bin/${binaryName}`,
  };

  const mcpJson = JSON.stringify(
    {
      mcpServers: {
        mycelicmemory: {
          command: platform === 'win32'
            ? `C:\\path\\to\\${binaryName}`
            : `/path/to/${binaryName}`,
          args: ['mcp'],
        },
      },
    },
    null,
    2,
  );

  const configFileLocation: Record<Platform, string> = {
    win32: '%APPDATA%\\Claude\\claude_desktop_config.json',
    darwin: '~/Library/Application Support/Claude/claude_desktop_config.json',
    linux: '~/.config/Claude/claude_desktop_config.json',
  };

  return (
    <SettingsSection title="Connect Claude (MCP)" icon={Plug}>
      <p className="text-sm text-slate-400">
        MCP (Model Context Protocol) lets Claude communicate with MycelicMemory directly.
        Add this to your MCP config to enable it.
      </p>

      <PlatformToggle value={platform} onChange={setPlatform} />

      {/* Step 1 */}
      <div>
        <h3 className="text-sm font-medium mb-1">1. Locate your binary</h3>
        <CodeBlock language="shell">{findBinaryCommand[platform]}</CodeBlock>
      </div>

      {/* Step 2 */}
      <div>
        <h3 className="text-sm font-medium mb-1">2. Claude Code config</h3>
        <p className="text-xs text-slate-500 mb-2">
          Add to <code className="bg-slate-700 px-1 rounded">~/.claude/mcp.json</code>
          {' '}(create if it doesn't exist):
        </p>
        <CodeBlock language="json">{mcpJson}</CodeBlock>
      </div>

      {/* Step 3 */}
      <div>
        <h3 className="text-sm font-medium mb-1">3. Claude Desktop config</h3>
        <p className="text-xs text-slate-500 mb-2">
          Config file location:
          <code className="bg-slate-700 px-1 rounded ml-1">{configFileLocation[platform]}</code>
        </p>
        <p className="text-xs text-slate-500 mb-2">Add the same JSON block above to your Claude Desktop config file.</p>
      </div>

      {/* Step 4 */}
      <div>
        <h3 className="text-sm font-medium mb-1">4. Verify</h3>
        <p className="text-xs text-slate-500">
          Run <code className="bg-slate-700 px-1 rounded">/mcp</code> in Claude Code to confirm the
          connection, or restart Claude Desktop and look for MycelicMemory in the MCP tools list.
        </p>
      </div>
    </SettingsSection>
  );
}

// ── Model Selector ───────────────────────────────────────────────

interface ModelSelectorProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  models: string[];
  loading: boolean;
  hint?: string;
  placeholder?: string;
}

function ModelSelector({ label, value, onChange, models, loading, hint, placeholder }: ModelSelectorProps) {
  const [showCustom, setShowCustom] = useState(false);

  // If current value isn't in the model list, show custom input
  const valueInList = models.some(
    (m) => m === value || m.split(':')[0] === value,
  );
  const isCustomMode = showCustom || (!valueInList && value !== '' && models.length > 0);

  return (
    <div>
      <label className="text-sm text-slate-400">{label}</label>
      {loading ? (
        <div className="w-full mt-1 p-2 bg-slate-700 rounded-lg text-sm text-slate-500 flex items-center gap-2">
          <RefreshCw className="w-3 h-3 animate-spin" />
          Loading models...
        </div>
      ) : models.length === 0 && !isCustomMode ? (
        <div>
          <input
            type="text"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            placeholder={placeholder}
            className="w-full mt-1 p-2 bg-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
          <p className="text-xs text-amber-400 mt-1">
            No models found — install Ollama and pull models
          </p>
        </div>
      ) : isCustomMode ? (
        <div className="flex gap-2 mt-1">
          <input
            type="text"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            placeholder={placeholder}
            className="flex-1 p-2 bg-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
          {models.length > 0 && (
            <button
              onClick={() => setShowCustom(false)}
              className="px-2 py-1 text-xs bg-slate-600 hover:bg-slate-500 rounded-lg transition-colors"
              title="Switch to dropdown"
            >
              <ChevronDown className="w-4 h-4" />
            </button>
          )}
        </div>
      ) : (
        <div className="flex gap-2 mt-1">
          <select
            value={value}
            onChange={(e) => {
              if (e.target.value === '__custom__') {
                setShowCustom(true);
              } else {
                onChange(e.target.value);
              }
            }}
            className="flex-1 p-2 bg-slate-700 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            {!valueInList && <option value={value}>{value} (not installed)</option>}
            {models.map((m) => (
              <option key={m} value={m}>{m}</option>
            ))}
            <option value="__custom__">Type custom name...</option>
          </select>
        </div>
      )}
      {hint && <p className="text-xs text-slate-500 mt-1">{hint}</p>}
    </div>
  );
}

// ── Main Settings Page ───────────────────────────────────────────

export default function SettingsPage() {
  const [settings, setSettings] = useState<AppSettings | null>(null);
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [testing, setTesting] = useState(false);

  // Model list from Ollama
  const [ollamaModels, setOllamaModels] = useState<string[]>([]);
  const [modelsLoading, setModelsLoading] = useState(true);

  useEffect(() => {
    fetchSettings();
    testConnections();
    fetchOllamaModels();
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

  async function fetchOllamaModels() {
    try {
      setModelsLoading(true);
      const status: ServiceStatus = await window.mycelicMemory.services.status();
      setOllamaModels(status.ollama?.models ?? []);
    } catch (err) {
      console.error('Failed to fetch Ollama models:', err);
    } finally {
      setModelsLoading(false);
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
        {/* MCP Setup Guide */}
        <MCPSetupGuide />

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
            <ModelSelector
              label="Embedding Model"
              value={settings.ollama_embedding_model}
              onChange={(v) => updateSetting('ollama_embedding_model', v)}
              models={ollamaModels}
              loading={modelsLoading}
              placeholder="nomic-embed-text"
              hint="Model for generating embeddings"
            />
            <ModelSelector
              label="Chat Model"
              value={settings.ollama_chat_model}
              onChange={(v) => updateSetting('ollama_chat_model', v)}
              models={ollamaModels}
              loading={modelsLoading}
              placeholder="llama3.2"
              hint="Model for AI analysis"
            />
          </div>
          <div className="flex items-start gap-2 p-3 bg-slate-900 rounded-lg border border-slate-700">
            <Info className="w-4 h-4 text-primary-400 mt-0.5 shrink-0" />
            <p className="text-xs text-slate-400">
              Cloud models (GPT-4, Claude, etc.) can be used via OpenAI-compatible proxies like{' '}
              <span className="text-primary-400">LiteLLM</span> or Ollama's OpenAI compatibility layer.
              Point the Base URL to your proxy and use the appropriate model name.
            </p>
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
