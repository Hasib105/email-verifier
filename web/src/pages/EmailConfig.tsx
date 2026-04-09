import { useCallback, useEffect, useState } from 'react';
import { Pencil, Plus, Trash2, X } from 'lucide-react';
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

const inputClass = 'mt-1 block h-10 w-full border border-slate-300 px-3 text-sm text-slate-950 outline-none focus:border-slate-950 focus:ring-1 focus:ring-slate-950';

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
      setError('');
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
    const imapHost = formData.imap_host.trim() || smtpHost;
    if (!isValidHost(smtpHost)) {
      setError('SMTP host must be a valid hostname like smtp.gmail.com.');
      return;
    }
    if (!isValidHost(imapHost)) {
      setError('IMAP host must be a valid hostname like imap.gmail.com.');
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
    <div className="mx-auto max-w-[1280px] space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-slate-500">Probe delivery</p>
          <h1 className="mt-2 text-2xl font-bold tracking-tight text-slate-950 sm:text-3xl">Email Configurations</h1>
          <p className="mt-1 text-sm text-slate-500">Configure SMTP and IMAP accounts for probe-based verification.</p>
        </div>
        {!showForm && (
          <button
            className="inline-flex h-9 items-center justify-center gap-2 bg-slate-950 px-4 text-sm font-semibold text-white hover:bg-slate-800"
            onClick={() => setShowForm(true)}
            type="button"
          >
            <Plus className="h-4 w-4" />
            Add Account
          </button>
        )}
      </div>

      {error && (
        <div className="border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      )}

      {showForm && (
        <section className="border border-slate-200 bg-white">
          <div className="flex items-center justify-between border-b border-slate-200 px-5 py-4">
            <h2 className="text-base font-bold text-slate-950">{editingId ? 'Edit SMTP Account' : 'Add SMTP Account'}</h2>
            <button className="p-2 text-slate-500 hover:bg-slate-100" onClick={resetForm} type="button">
              <X className="h-4 w-4" />
            </button>
          </div>

          <form className="space-y-5 p-5" onSubmit={handleSubmit}>
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="block text-sm font-semibold text-slate-700">SMTP Host</label>
                <input className={inputClass} onChange={(e) => setFormData({ ...formData, host: e.target.value })} placeholder="smtp.gmail.com" required type="text" value={formData.host} />
              </div>
              <div>
                <label className="block text-sm font-semibold text-slate-700">SMTP Port</label>
                <input className={inputClass} onChange={(e) => setFormData({ ...formData, port: parseInt(e.target.value) || 587 })} required type="number" value={formData.port} />
              </div>
              <div>
                <label className="block text-sm font-semibold text-slate-700">Username</label>
                <input className={inputClass} onChange={(e) => setFormData({ ...formData, username: e.target.value })} placeholder="user@example.com" required type="email" value={formData.username} />
              </div>
              <div>
                <label className="block text-sm font-semibold text-slate-700">
                  Password {editingId && <span className="font-normal text-slate-400">(leave blank to keep)</span>}
                </label>
                <input className={inputClass} onChange={(e) => setFormData({ ...formData, password: e.target.value })} placeholder="Password" required={!editingId} type="password" value={formData.password} />
              </div>
            </div>

            <div>
              <label className="block text-sm font-semibold text-slate-700">Sender Email</label>
              <input className={inputClass} onChange={(e) => setFormData({ ...formData, sender: e.target.value })} placeholder="noreply@yourdomain.com" required type="email" value={formData.sender} />
            </div>

            <div className="grid gap-4 sm:grid-cols-3">
              <div>
                <label className="block text-sm font-semibold text-slate-700">IMAP Host</label>
                <input className={inputClass} onChange={(e) => setFormData({ ...formData, imap_host: e.target.value })} placeholder="imap.gmail.com" type="text" value={formData.imap_host} />
              </div>
              <div>
                <label className="block text-sm font-semibold text-slate-700">IMAP Port</label>
                <input className={inputClass} onChange={(e) => setFormData({ ...formData, imap_port: parseInt(e.target.value) || 993 })} type="number" value={formData.imap_port} />
              </div>
              <div>
                <label className="block text-sm font-semibold text-slate-700">Daily Limit</label>
                <input className={inputClass} onChange={(e) => setFormData({ ...formData, daily_limit: parseInt(e.target.value) || 100 })} type="number" value={formData.daily_limit} />
              </div>
            </div>

            <label className="inline-flex items-center gap-2 text-sm font-medium text-slate-700">
              <input checked={formData.active} className="h-4 w-4 border-slate-300" onChange={(e) => setFormData({ ...formData, active: e.target.checked })} type="checkbox" />
              Active
            </label>

            <div className="flex gap-3 border-t border-slate-200 pt-5">
              <button className="h-9 bg-slate-950 px-4 text-sm font-semibold text-white hover:bg-slate-800 disabled:opacity-50" disabled={saving} type="submit">
                {saving ? 'Saving...' : editingId ? 'Update' : 'Create'}
              </button>
              <button className="h-9 border border-slate-200 bg-white px-4 text-sm font-semibold text-slate-700 hover:bg-slate-50" onClick={resetForm} type="button">
                Cancel
              </button>
            </div>
          </form>
        </section>
      )}

      <section className="border border-slate-200 bg-white">
        {loading ? (
          <div className="p-10 text-center">
            <div className="mx-auto h-6 w-6 animate-spin rounded-full border-2 border-slate-200 border-t-slate-950" />
          </div>
        ) : accounts.length === 0 ? (
          <div className="p-10 text-center text-sm text-slate-500">No SMTP accounts configured yet.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full border-collapse text-left">
              <thead className="bg-slate-50 text-xs font-semibold uppercase tracking-widest text-slate-500">
                <tr>
                  <th className="border-b border-slate-200 px-4 py-3">Host</th>
                  <th className="hidden border-b border-slate-200 px-4 py-3 sm:table-cell">Username</th>
                  <th className="hidden border-b border-slate-200 px-4 py-3 md:table-cell">Usage</th>
                  <th className="border-b border-slate-200 px-4 py-3">Status</th>
                  <th className="border-b border-slate-200 px-4 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200">
                {accounts.map((account) => (
                  <tr className="hover:bg-slate-50" key={account.id}>
                    <td className="px-4 py-3 text-sm font-semibold text-slate-950">{account.host}:{account.port}</td>
                    <td className="hidden px-4 py-3 text-sm text-slate-600 sm:table-cell">{account.username}</td>
                    <td className="hidden px-4 py-3 text-sm text-slate-600 md:table-cell">{account.sent_today} / {account.daily_limit}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2.5 py-1 text-xs font-semibold ${account.active ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-700'}`}>
                        {account.active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button className="inline-flex h-8 w-8 items-center justify-center text-slate-500 hover:bg-slate-100 hover:text-slate-950" onClick={() => handleEdit(account)} type="button">
                        <Pencil className="h-4 w-4" />
                      </button>
                      <button className="inline-flex h-8 w-8 items-center justify-center text-slate-500 hover:bg-red-50 hover:text-red-600" onClick={() => handleDelete(account.id)} type="button">
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
