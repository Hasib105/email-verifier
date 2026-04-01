import { useState, useEffect, useCallback } from 'react';
import { Plus, Pencil, Trash2, X } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { SMTPAccount, SMTPAccountCreateRequest } from '../types';

const hostnamePattern = /^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)(?:\.(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?))*$/i;

function isValidHost(value: string): boolean {
  const host = value.trim();
  if (!host) return false;
  if (host.includes('@') || host.includes('://') || /[\\/\s]/.test(host)) return false;
  if (/^\d{1,3}(?:\.\d{1,3}){3}$/.test(host)) return true;
  return hostnamePattern.test(host);
}

export function EmailConfig() {
  const { config } = useAuth();
  const [accounts, setAccounts] = useState<SMTPAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const [formData, setFormData] = useState<SMTPAccountCreateRequest>({
    host: '',
    port: 587,
    username: '',
    password: '',
    sender: '',
    imap_host: '',
    imap_port: 993,
    imap_mailbox: 'INBOX',
    daily_limit: 100,
    active: true,
  });

  const loadAccounts = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.listSmtpAccounts(config);
      setAccounts(response.items || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load accounts');
    } finally {
      setLoading(false);
    }
  }, [config]);

  useEffect(() => {
    void loadAccounts();
  }, [loadAccounts]);

  const resetForm = () => {
    setFormData({
      host: '',
      port: 587,
      username: '',
      password: '',
      sender: '',
      imap_host: '',
      imap_port: 993,
      imap_mailbox: 'INBOX',
      daily_limit: 100,
      active: true,
    });
    setEditingId(null);
    setShowForm(false);
  };

  const handleEdit = (account: SMTPAccount) => {
    setFormData({
      host: account.host,
      port: account.port,
      username: account.username,
      password: '',
      sender: account.sender,
      imap_host: account.imap_host,
      imap_port: account.imap_port,
      imap_mailbox: account.imap_mailbox,
      daily_limit: account.daily_limit,
      active: account.active,
    });
    setEditingId(account.id);
    setShowForm(true);
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this SMTP account?')) return;
    
    try {
      await api.deleteSmtpAccount(config, id);
      await loadAccounts();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete account');
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const smtpHost = formData.host.trim();
    const imapHost = (formData.imap_host.trim() || smtpHost);
    if (!isValidHost(smtpHost)) {
      setError('SMTP host must be a valid hostname like smtp.gmail.com (not an email address).');
      return;
    }
    if (!isValidHost(imapHost)) {
      setError('IMAP host must be a valid hostname like imap.gmail.com.');
      return;
    }
    if (formData.port < 1 || formData.port > 65535) {
      setError('SMTP port must be between 1 and 65535.');
      return;
    }
    if (formData.imap_port < 1 || formData.imap_port > 65535) {
      setError('IMAP port must be between 1 and 65535.');
      return;
    }

    setSaving(true);
    setError('');

    const payload: SMTPAccountCreateRequest = {
      ...formData,
      host: smtpHost,
      imap_host: imapHost,
      username: formData.username.trim(),
      sender: formData.sender.trim(),
      imap_mailbox: formData.imap_mailbox.trim() || 'INBOX',
    };

    try {
      if (editingId) {
        await api.updateSmtpAccount(config, editingId, payload);
      } else {
        await api.createSmtpAccount(config, payload);
      }
      resetForm();
      await loadAccounts();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save account');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4">
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold text-gray-900 tracking-tight">Email Configurations</h2>
          <p className="text-sm text-gray-500 mt-1">Configure SMTP accounts for probe-based verification.</p>
        </div>
        {!showForm && (
          <button
            onClick={() => setShowForm(true)}
            className="flex items-center justify-center gap-2 bg-yellow-400 text-black px-4 py-2 rounded-md text-sm font-bold hover:bg-yellow-500 shadow-sm"
          >
            <Plus className="w-4 h-4" />
            Add Account
          </button>
        )}
      </div>

      {error && (
        <div className="p-4 bg-red-50 border border-red-200 text-red-700 rounded-xl text-sm">
          {error}
        </div>
      )}

      {showForm && (
        <div className="bg-white border rounded-xl shadow-sm p-6">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-lg font-bold">{editingId ? 'Edit' : 'Add'} SMTP Account</h3>
            <button onClick={resetForm} className="text-gray-400 hover:text-gray-600">
              <X className="w-5 h-5" />
            </button>
          </div>
          
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700">SMTP Host</label>
                <input
                  type="text"
                  value={formData.host}
                  onChange={(e) => setFormData({ ...formData, host: e.target.value })}
                  placeholder="smtp.gmail.com"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 border"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">SMTP Port</label>
                <input
                  type="number"
                  value={formData.port}
                  onChange={(e) => setFormData({ ...formData, port: parseInt(e.target.value) || 587 })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 border"
                  required
                />
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700">Username</label>
                <input
                  type="email"
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                  placeholder="user@example.com"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 border"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Password {editingId && <span className="text-gray-400">(leave blank to keep)</span>}
                </label>
                <input
                  type="password"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  placeholder="••••••••"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 border"
                  required={!editingId}
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700">Sender Email</label>
              <input
                type="email"
                value={formData.sender}
                onChange={(e) => setFormData({ ...formData, sender: e.target.value })}
                placeholder="noreply@yourdomain.com"
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 border"
                required
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700">IMAP Host</label>
                <input
                  type="text"
                  value={formData.imap_host}
                  onChange={(e) => setFormData({ ...formData, imap_host: e.target.value })}
                  placeholder="imap.gmail.com"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 border"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">IMAP Port</label>
                <input
                  type="number"
                  value={formData.imap_port}
                  onChange={(e) => setFormData({ ...formData, imap_port: parseInt(e.target.value) || 993 })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 border"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700">Daily Limit</label>
                <input
                  type="number"
                  value={formData.daily_limit}
                  onChange={(e) => setFormData({ ...formData, daily_limit: parseInt(e.target.value) || 100 })}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm p-2 border"
                />
              </div>
            </div>

            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="active"
                checked={formData.active}
                onChange={(e) => setFormData({ ...formData, active: e.target.checked })}
                className="rounded border-gray-300"
              />
              <label htmlFor="active" className="text-sm text-gray-700">Active</label>
            </div>

            <div className="flex gap-3 pt-4">
              <button
                type="submit"
                disabled={saving}
                className="bg-black text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-gray-800 disabled:opacity-50"
              >
                {saving ? 'Saving...' : editingId ? 'Update' : 'Create'}
              </button>
              <button
                type="button"
                onClick={resetForm}
                className="px-4 py-2 rounded-md text-sm font-medium border hover:bg-gray-50"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      <div className="bg-white border rounded-xl shadow-sm overflow-hidden">
        {loading ? (
          <div className="p-8 text-center">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-yellow-500 mx-auto"></div>
          </div>
        ) : accounts.length === 0 ? (
          <div className="p-8 text-center text-gray-500">
            No SMTP accounts configured yet
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Host</th>
                  <th className="hidden sm:table-cell px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Username</th>
                  <th className="hidden md:table-cell px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Usage</th>
                  <th className="px-4 sm:px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                  <th className="px-4 sm:px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {accounts.map((account) => (
                  <tr key={account.id} className="hover:bg-gray-50">
                    <td className="px-4 sm:px-6 py-4 text-sm font-medium text-gray-900">
                      {account.host}:{account.port}
                    </td>
                    <td className="hidden sm:table-cell px-6 py-4 text-sm text-gray-500">{account.username}</td>
                    <td className="hidden md:table-cell px-6 py-4 text-sm text-gray-500">
                      {account.sent_today} / {account.daily_limit}
                    </td>
                    <td className="px-4 sm:px-6 py-4 text-sm">
                      <span className={`px-2 py-1 text-xs font-semibold rounded-full ${
                        account.active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                      }`}>
                        {account.active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="px-4 sm:px-6 py-4 text-right">
                      <div className="flex justify-end gap-2">
                        <button
                          onClick={() => handleEdit(account)}
                          className="p-1 text-gray-400 hover:text-blue-600"
                        >
                          <Pencil className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(account.id)}
                          className="p-1 text-gray-400 hover:text-red-600"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
