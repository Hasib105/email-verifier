import { useState, useEffect } from 'react';
import { Save, TestTube, CheckCircle, AlertCircle } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';

export function Settings() {
  const { config } = useAuth();
  const [webhookUrl, setWebhookUrl] = useState('');
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
  "verified_at": "2024-01-15T10:30:00Z",
  "mx_record": "gmail-smtp-in.l.google.com",
  "details": "Bounce detected after probe verification"
}`}
        </pre>
      </div>
    </div>
  );
}
