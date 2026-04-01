import { useEffect, useState } from 'react';
import { Copy, Eye, EyeOff, Key, RefreshCw, Server } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { HealthResponse } from '../types';

export function Settings() {
  const { config, updateApiKey } = useAuth();
  const [apiKey, setApiKey] = useState(config.apiKey);
  const [showApiKey, setShowApiKey] = useState(false);
  const [regenerating, setRegenerating] = useState(false);
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  useEffect(() => {
    const load = async () => {
      try {
        const healthData = await api.getHealth(config);
        setHealth(healthData);
      } catch (err) {
        setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Failed to load health information' });
      }
    };

    void load();
  }, [config]);

  const handleRegenerateApiKey = async () => {
    if (!window.confirm('Regenerate the API key? Existing integrations will stop working until updated.')) {
      return;
    }

    setRegenerating(true);
    setMessage(null);
    try {
      const response = await api.regenerateAPIKey(config);
      setApiKey(response.api_key);
      updateApiKey(response.api_key);
      setMessage({ type: 'success', text: 'API key regenerated successfully.' });
    } catch (err) {
      setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Failed to regenerate API key' });
    } finally {
      setRegenerating(false);
    }
  };

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-2xl sm:text-3xl font-bold text-gray-900 tracking-tight">Settings</h2>
        <p className="text-sm text-gray-500 mt-1">Direct-callout verifier configuration and API access.</p>
      </div>

      {message && (
        <div className={`rounded-xl border p-4 text-sm ${message.type === 'success' ? 'border-green-200 bg-green-50 text-green-700' : 'border-red-200 bg-red-50 text-red-700'}`}>
          {message.text}
        </div>
      )}

      <div className="bg-white border rounded-xl shadow-sm p-6">
        <div className="flex items-center gap-2 mb-4">
          <Key className="w-5 h-5 text-gray-500" />
          <h3 className="text-lg font-semibold">API Access</h3>
        </div>

        <div className="flex rounded-lg border overflow-hidden">
          <input
            type={showApiKey ? 'text' : 'password'}
            value={apiKey}
            readOnly
            className="flex-1 px-3 py-3 text-sm font-mono bg-gray-50"
          />
          <button
            type="button"
            onClick={() => setShowApiKey((current) => !current)}
            className="px-3 border-l bg-white hover:bg-gray-50"
            title={showApiKey ? 'Hide API key' : 'Show API key'}
          >
            {showApiKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
          </button>
          <button
            type="button"
            onClick={() => navigator.clipboard.writeText(apiKey)}
            className="px-3 border-l bg-white hover:bg-gray-50"
            title="Copy API key"
          >
            <Copy className="w-4 h-4" />
          </button>
        </div>

        <button
          type="button"
          onClick={() => void handleRegenerateApiKey()}
          disabled={regenerating}
          className="mt-4 inline-flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 disabled:opacity-50"
        >
          <RefreshCw className={`w-4 h-4 ${regenerating ? 'animate-spin' : ''}`} />
          {regenerating ? 'Regenerating...' : 'Regenerate API Key'}
        </button>
      </div>

      <div className="bg-white border rounded-xl shadow-sm p-6">
        <div className="flex items-center gap-2 mb-4">
          <Server className="w-5 h-5 text-gray-500" />
          <h3 className="text-lg font-semibold">Verifier Runtime</h3>
        </div>

        {health ? (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <SettingRow label="Mode" value={health.mode} />
            <SettingRow label="MAIL FROM" value={health.mail_from} />
            <SettingRow label="EHLO Domain" value={health.ehlo_domain} />
            <SettingRow label="Max Parallel Callouts" value={String(health.max_parallel)} />
            <SettingRow label="Deliverable Cache TTL" value={health.deliverable_ttl} />
            <SettingRow label="Baseline TTL" value={health.baseline_ttl} />
          </div>
        ) : (
          <p className="text-sm text-gray-500">Health information is unavailable.</p>
        )}
      </div>
    </div>
  );
}

function SettingRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-gray-200 bg-gray-50 p-4">
      <p className="text-xs uppercase tracking-wider text-gray-500">{label}</p>
      <p className="mt-2 text-sm font-semibold text-gray-900 break-all">{value}</p>
    </div>
  );
}
