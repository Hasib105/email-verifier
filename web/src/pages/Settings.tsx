import { useState, useEffect } from 'react';
import { Save, TestTube, CheckCircle, AlertCircle, Key, RefreshCw, Eye, EyeOff, Copy } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';

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

  useEffect(() => {
    loadSettings();
  }, [config]);

  const loadSettings = async () => {
    setLoading(true);
    try {
      const settings = await api.getSettings(config);
      setWebhookUrl(settings.webhook_url || '');
      // If the API allows passing back API key on /users/me, we can use it.
      // Otherwise, we take it from AuthContext config.
      setApiKey(config.apiKey);
    } catch (err) {
      setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Failed to load settings' });
    } finally {
      setLoading(false);
    }
  };

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
      setMessage({ type: 'success', text: 'Test webhook sent! Check your endpoint.' });
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
      setMessage({ type: 'success', text: 'API Key regenerated successfully.' });
    } catch (err) {
      setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Failed to regenerate API key' });
    } finally {
      setRegenerating(false);
    }
  };

  const copyApiKey = () => {
    navigator.clipboard.writeText(apiKey);
    setMessage({ type: 'success', text: 'API Key copied to clipboard!' });
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-yellow-500"></div>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-2xl sm:text-3xl font-bold text-gray-900 tracking-tight">Settings</h2>
        <p className="text-sm text-gray-500 mt-1">Manage global configurations.</p>
      </div>

      {message && (
        <div className={`flex items-center gap-2 p-4 rounded-xl text-sm ${
          message.type === 'success'
            ? 'bg-green-50 border border-green-200 text-green-700'
            : 'bg-red-50 border border-red-200 text-red-700'
        }`}>
          {message.type === 'success' ? <CheckCircle className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}
          {message.text}
        </div>
      )}

      <div className="bg-white border rounded-xl shadow-sm p-6">
        <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Key className="w-5 h-5 text-gray-500" />
          API Access Configuration
        </h3>
        <p className="text-sm text-gray-500 mb-4">
          Your API Key is used to authenticate requests to the verifier API. Keep it secure and do not share it.
        </p>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Your API Key</label>
            <div className="flex bg-gray-50 border border-gray-300 rounded-md overflow-hidden">
              <input
                type={showApiKey ? "text" : "password"}
                readOnly
                value={apiKey}
                className="flex-1 bg-transparent border-none focus:ring-0 px-3 py-2 text-sm text-gray-800 font-mono"
              />
              <button
                type="button"
                onClick={() => setShowApiKey(!showApiKey)}
                className="px-3 text-gray-500 hover:text-gray-700 hover:bg-gray-100 flex items-center justify-center transition-colors"
                title={showApiKey ? "Hide key" : "Show key"}
              >
                {showApiKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
              <button
                type="button"
                onClick={copyApiKey}
                className="px-3 border-l border-gray-300 text-gray-500 hover:text-gray-700 hover:bg-gray-100 flex items-center justify-center transition-colors"
                title="Copy to clipboard"
              >
                <Copy className="w-4 h-4" />
              </button>
            </div>
          </div>
          
          <div className="pt-2">
            <button
              type="button"
              onClick={handleRegenerateApiKey}
              disabled={regenerating}
              className="flex items-center gap-2 bg-white border border-gray-300 text-red-600 px-4 py-2 rounded-md text-sm font-medium hover:bg-red-50 disabled:opacity-50 transition-colors"
            >
              <RefreshCw className={`w-4 h-4 ${regenerating ? 'animate-spin' : ''}`} />
              {regenerating ? 'Regenerating...' : 'Regenerate API Key'}
            </button>
          </div>
        </div>
      </div>

      <div className="bg-white border rounded-xl shadow-sm p-6">
        <h3 className="text-lg font-semibold mb-4">Webhook Configuration</h3>
        <p className="text-sm text-gray-500 mb-4">
          Configure a webhook URL to receive notifications when email verification status changes 
          (e.g., when a bounce is detected after the 6-hour check).
        </p>
        
        <form onSubmit={handleSave} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">Webhook URL</label>
            <input
              type="url"
              value={webhookUrl}
              onChange={(e) => setWebhookUrl(e.target.value)}
              placeholder="https://yourdomain.com/webhook"
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 border"
            />
            <p className="mt-1 text-xs text-gray-500">
              The webhook will receive a POST request with JSON payload containing verification results.
            </p>
          </div>

          <div className="flex flex-wrap gap-3">
            <button
              type="submit"
              disabled={saving}
              className="flex items-center gap-2 bg-black text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-gray-800 disabled:opacity-50"
            >
              <Save className="w-4 h-4" />
              {saving ? 'Saving...' : 'Save Webhook'}
            </button>
            <button
              type="button"
              onClick={handleTestWebhook}
              disabled={testing || !webhookUrl}
              className="flex items-center gap-2 bg-gray-100 text-gray-700 px-4 py-2 rounded-md text-sm font-medium hover:bg-gray-200 disabled:opacity-50"
            >
              <TestTube className="w-4 h-4" />
              {testing ? 'Sending...' : 'Test Webhook'}
            </button>
          </div>
        </form>
      </div>

      <div className="bg-white border rounded-xl shadow-sm p-6">
        <h3 className="text-lg font-semibold mb-4">Webhook Payload Example</h3>
        <pre className="bg-gray-900 text-gray-100 p-4 rounded-lg text-sm overflow-x-auto">
{`{
  "event": "verification_complete",
  "email": "user@example.com",
  "status": "bounced",
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
      </div>
    </div>
  );
}
