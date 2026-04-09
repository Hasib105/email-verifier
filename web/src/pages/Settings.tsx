import { useCallback, useEffect, useState } from 'react';
import { AlertCircle, CheckCircle, Copy, Eye, EyeOff, Key, RefreshCw, Save, TestTube } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';

const inputClass = 'block h-10 w-full border border-slate-300 bg-white px-3 text-sm text-slate-950 outline-none focus:border-slate-950 focus:ring-1 focus:ring-slate-950';

export function Settings() {
  const { config, updateApiKey } = useAuth();
  const [webhookUrl, setWebhookUrl] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [showApiKey, setShowApiKey] = useState(false);
  const [regenerating, setRegenerating] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const loadSettings = useCallback(async () => {
    setLoading(true);
    try {
      const settings = await api.getSettings(config);
      setWebhookUrl(settings.webhook_url || '');
      setApiKey(config.apiKey);
    } catch (err) {
      setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Failed to load settings' });
    } finally {
      setLoading(false);
    }
  }, [config]);

  useEffect(() => {
    void loadSettings();
  }, [loadSettings]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setMessage(null);

    try {
      await api.updateSettings(config, { webhook_url: webhookUrl });
      setMessage({ type: 'success', text: 'Webhook URL saved successfully' });
    } catch (err) {
      setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Failed to save settings' });
    } finally {
      setSaving(false);
    }
  };

  const handleTestWebhook = async () => {
    if (!webhookUrl) {
      setMessage({ type: 'error', text: 'Please enter a webhook URL first' });
      return;
    }

    setTesting(true);
    setMessage(null);

    try {
      await api.testWebhook(config, webhookUrl);
      setMessage({ type: 'success', text: 'Test webhook sent. Check your endpoint.' });
    } catch (err) {
      setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Failed to send test webhook' });
    } finally {
      setTesting(false);
    }
  };

  const handleRegenerateApiKey = async () => {
    if (!window.confirm('Are you sure? Any integrations using your current API key will break.')) return;

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

  const copyApiKey = () => {
    void navigator.clipboard.writeText(apiKey);
    setMessage({ type: 'success', text: 'API key copied to clipboard.' });
  };

  if (loading) {
    return (
      <div className="flex min-h-[50vh] items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-slate-200 border-t-slate-950" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-[1120px] space-y-6">
      <div>
        <p className="text-xs font-semibold uppercase tracking-widest text-slate-500">Workspace controls</p>
        <h1 className="mt-2 text-2xl font-bold tracking-tight text-slate-950 sm:text-3xl">Settings</h1>
        <p className="mt-1 text-sm text-slate-500">Manage API access and webhook delivery.</p>
      </div>

      {message && (
        <div className={`flex items-center gap-2 border p-4 text-sm ${
          message.type === 'success'
            ? 'border-emerald-200 bg-emerald-50 text-emerald-700'
            : 'border-red-200 bg-red-50 text-red-700'
        }`}>
          {message.type === 'success' ? <CheckCircle className="h-4 w-4" /> : <AlertCircle className="h-4 w-4" />}
          {message.text}
        </div>
      )}

      <section className="border border-slate-200 bg-white p-5">
        <div className="flex items-center gap-3">
          <span className="flex h-9 w-9 items-center justify-center bg-slate-100 text-slate-700">
            <Key className="h-4 w-4" />
          </span>
          <div>
            <h2 className="text-base font-bold text-slate-950">API Access</h2>
            <p className="text-sm text-slate-500">Keep this key private and rotate it when needed.</p>
          </div>
        </div>

        <div className="mt-5">
          <label className="block text-sm font-semibold text-slate-700">Your API Key</label>
          <div className="mt-1 flex border border-slate-300 bg-slate-50">
            <input
              className="min-w-0 flex-1 bg-transparent px-3 py-2 font-mono text-sm text-slate-800 outline-none"
              readOnly
              type={showApiKey ? 'text' : 'password'}
              value={apiKey}
            />
            <button className="border-l border-slate-300 px-3 text-slate-500 hover:bg-white hover:text-slate-950" onClick={() => setShowApiKey(!showApiKey)} type="button">
              {showApiKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
            <button className="border-l border-slate-300 px-3 text-slate-500 hover:bg-white hover:text-slate-950" onClick={copyApiKey} type="button">
              <Copy className="h-4 w-4" />
            </button>
          </div>
        </div>

        <button
          className="mt-4 inline-flex h-9 items-center gap-2 border border-red-200 bg-white px-3 text-sm font-semibold text-red-600 hover:bg-red-50 disabled:opacity-50"
          disabled={regenerating}
          onClick={handleRegenerateApiKey}
          type="button"
        >
          <RefreshCw className={`h-4 w-4 ${regenerating ? 'animate-spin' : ''}`} />
          {regenerating ? 'Regenerating...' : 'Regenerate API Key'}
        </button>
      </section>

      <section className="border border-slate-200 bg-white p-5">
        <h2 className="text-base font-bold text-slate-950">Webhook Configuration</h2>
        <p className="mt-1 text-sm text-slate-500">Receive a POST request when verification status changes.</p>

        <form className="mt-5 space-y-4" onSubmit={handleSave}>
          <div>
            <label className="block text-sm font-semibold text-slate-700">Webhook URL</label>
            <input className={`${inputClass} mt-1`} onChange={(e) => setWebhookUrl(e.target.value)} placeholder="https://yourdomain.com/webhook" type="url" value={webhookUrl} />
          </div>

          <div className="flex flex-wrap gap-3">
            <button className="inline-flex h-9 items-center gap-2 bg-slate-950 px-4 text-sm font-semibold text-white hover:bg-slate-800 disabled:opacity-50" disabled={saving} type="submit">
              <Save className="h-4 w-4" />
              {saving ? 'Saving...' : 'Save Webhook'}
            </button>
            <button className="inline-flex h-9 items-center gap-2 border border-slate-200 bg-white px-4 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:opacity-50" disabled={testing || !webhookUrl} onClick={handleTestWebhook} type="button">
              <TestTube className="h-4 w-4" />
              {testing ? 'Sending...' : 'Test Webhook'}
            </button>
          </div>
        </form>
      </section>

      <section className="border border-slate-200 bg-white p-5">
        <h2 className="text-base font-bold text-slate-950">Webhook Payload Example</h2>
        <pre className="mt-4 overflow-x-auto bg-slate-950 p-4 text-sm leading-6 text-slate-100">
{`{
  "event": "verification_complete",
  "email": "user@example.com",
  "status": "invalid",
  "confidence": "high",
  "deterministic": true,
  "reason_code": "bounce_token_match",
  "verification_path": "hybrid",
  "signal_summary": "Bounce evidence matched the unique probe token.",
  "expires_at": 1736937000,
  "verified_at": "2024-01-15T10:30:00Z",
  "details": "Bounce detected after probe verification"
}`}
        </pre>
      </section>
    </div>
  );
}
