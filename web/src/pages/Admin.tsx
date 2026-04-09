import { useState } from 'react';
import { ChevronRight, Database, Mail, Server, Shield, ShieldOff, Trash2, Users } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { EmailTemplate, SMTPAccount, User, VerificationRecord } from '../types';

type ModelType = 'users' | 'verifications' | 'smtp_accounts' | 'templates' | null;

const statusBadge = (status: string) => {
  if (status === 'valid') return 'bg-emerald-50 text-emerald-700';
  if (status === 'invalid' || status === 'bounced') return 'bg-red-50 text-red-700';
  if (status === 'pending_bounce_check' || status === 'greylisted') return 'bg-amber-50 text-amber-700';
  return 'bg-slate-100 text-slate-700';
};

export function AdminPanel() {
  const { config, user } = useAuth();
  const [selectedModel, setSelectedModel] = useState<ModelType>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [users, setUsers] = useState<User[]>([]);
  const [verifications, setVerifications] = useState<VerificationRecord[]>([]);
  const [smtpAccounts, setSmtpAccounts] = useState<SMTPAccount[]>([]);
  const [templates, setTemplates] = useState<EmailTemplate[]>([]);

  if (!user?.is_superuser) {
    return (
      <div className="mx-auto flex min-h-[60vh] max-w-xl flex-col items-center justify-center text-center">
        <ShieldOff className="mb-4 h-14 w-14 text-slate-300" />
        <h1 className="text-2xl font-bold text-slate-950">Access Denied</h1>
        <p className="mt-2 text-sm leading-6 text-slate-500">Only superusers can view the admin panel.</p>
      </div>
    );
  }

  const loadModel = async (model: ModelType) => {
    if (!model) return;
    setLoading(true);
    setError('');
    setSelectedModel(model);

    try {
      if (model === 'users') {
        const response = await api.adminListUsers(config);
        setUsers(response.items || []);
      }
      if (model === 'verifications') {
        const response = await api.adminListVerifications(config);
        setVerifications(response.items || []);
      }
      if (model === 'smtp_accounts') {
        const response = await api.adminListSmtpAccounts(config);
        setSmtpAccounts(response.items || []);
      }
      if (model === 'templates') {
        const response = await api.adminListTemplates(config);
        setTemplates(response.items || []);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setLoading(false);
    }
  };

  const handleToggleSuperuser = async (userId: string, currentValue: boolean) => {
    if (!confirm(`Are you sure you want to ${currentValue ? 'remove' : 'grant'} superuser access?`)) return;

    try {
      await api.adminUpdateUser(config, userId, { is_superuser: !currentValue });
      await loadModel('users');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update user');
    }
  };

  const handleDeleteUser = async (userId: string) => {
    if (!confirm('Are you sure you want to delete this user? This action cannot be undone.')) return;

    try {
      await api.adminDeleteUser(config, userId);
      await loadModel('users');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete user');
    }
  };

  const handleDeleteVerification = async (id: string) => {
    if (!confirm('Delete this verification record?')) return;

    try {
      await api.adminDeleteVerification(config, id);
      await loadModel('verifications');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete');
    }
  };

  const renderTable = () => {
    if (loading) {
      return (
        <div className="p-10 text-center">
          <div className="mx-auto h-6 w-6 animate-spin rounded-full border-2 border-slate-200 border-t-slate-950" />
        </div>
      );
    }

    if (selectedModel === 'users') {
      return (
        <DataTable headers={['Email', 'API Key', 'Role', 'Created', 'Actions']}>
          {users.map((u) => (
            <tr className="hover:bg-slate-50" key={u.id}>
              <td className="px-4 py-3 text-sm font-semibold text-slate-950">{u.email}</td>
              <td className="hidden px-4 py-3 font-mono text-sm text-slate-500 sm:table-cell">{u.api_key?.slice(0, 8)}...</td>
              <td className="px-4 py-3">
                <span className={`px-2.5 py-1 text-xs font-semibold ${u.is_superuser ? 'bg-sky-50 text-sky-700' : 'bg-slate-100 text-slate-700'}`}>
                  {u.is_superuser ? 'Superuser' : 'User'}
                </span>
              </td>
              <td className="hidden px-4 py-3 text-sm text-slate-500 md:table-cell">{new Date(u.created_at).toLocaleDateString()}</td>
              <td className="px-4 py-3 text-right">
                <button className="inline-flex h-8 w-8 items-center justify-center text-slate-500 hover:bg-slate-100 hover:text-slate-950" onClick={() => handleToggleSuperuser(u.id, u.is_superuser)} type="button">
                  <Shield className="h-4 w-4" />
                </button>
                <button className="inline-flex h-8 w-8 items-center justify-center text-slate-500 hover:bg-red-50 hover:text-red-600" onClick={() => handleDeleteUser(u.id)} type="button">
                  <Trash2 className="h-4 w-4" />
                </button>
              </td>
            </tr>
          ))}
        </DataTable>
      );
    }

    if (selectedModel === 'verifications') {
      return (
        <DataTable headers={['Email', 'Status', 'Source', 'Date', 'Actions']}>
          {verifications.map((v) => (
            <tr className="hover:bg-slate-50" key={v.id}>
              <td className="px-4 py-3 text-sm font-semibold text-slate-950">{v.email}</td>
              <td className="px-4 py-3">
                <span className={`px-2.5 py-1 text-xs font-semibold capitalize ${statusBadge(v.status)}`}>{v.status.replace(/_/g, ' ')}</span>
                <div className="mt-1 text-xs capitalize text-slate-500">{v.confidence || 'low'} confidence</div>
              </td>
              <td className="hidden px-4 py-3 text-sm text-slate-500 sm:table-cell">{v.source || '-'}</td>
              <td className="hidden px-4 py-3 text-sm text-slate-500 md:table-cell">{new Date(v.created_at * 1000).toLocaleString()}</td>
              <td className="px-4 py-3 text-right">
                <button className="inline-flex h-8 w-8 items-center justify-center text-slate-500 hover:bg-red-50 hover:text-red-600" onClick={() => handleDeleteVerification(v.id)} type="button">
                  <Trash2 className="h-4 w-4" />
                </button>
              </td>
            </tr>
          ))}
        </DataTable>
      );
    }

    if (selectedModel === 'smtp_accounts') {
      return (
        <DataTable headers={['Host', 'Username', 'Owner', 'Usage', 'Status']}>
          {smtpAccounts.map((a) => (
            <tr className="hover:bg-slate-50" key={a.id}>
              <td className="px-4 py-3 text-sm font-semibold text-slate-950">{a.host}:{a.port}</td>
              <td className="hidden px-4 py-3 text-sm text-slate-500 sm:table-cell">{a.username}</td>
              <td className="px-4 py-3 text-sm text-slate-500">{a.user_id?.slice(0, 8)}...</td>
              <td className="hidden px-4 py-3 text-sm text-slate-500 md:table-cell">{a.sent_today} / {a.daily_limit}</td>
              <td className="px-4 py-3">
                <span className={`px-2.5 py-1 text-xs font-semibold ${a.active ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-700'}`}>
                  {a.active ? 'Active' : 'Inactive'}
                </span>
              </td>
            </tr>
          ))}
        </DataTable>
      );
    }

    if (selectedModel === 'templates') {
      return (
        <DataTable headers={['Name', 'Subject', 'Owner', 'Status']}>
          {templates.map((t) => (
            <tr className="hover:bg-slate-50" key={t.id}>
              <td className="px-4 py-3 text-sm font-semibold text-slate-950">{t.name}</td>
              <td className="hidden max-w-[360px] px-4 py-3 text-sm text-slate-500 sm:table-cell">
                <p className="truncate">{t.subject_template}</p>
              </td>
              <td className="px-4 py-3 text-sm text-slate-500">{t.user_id?.slice(0, 8)}...</td>
              <td className="px-4 py-3">
                <span className={`px-2.5 py-1 text-xs font-semibold ${t.active ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-700'}`}>
                  {t.active ? 'Active' : 'Inactive'}
                </span>
              </td>
            </tr>
          ))}
        </DataTable>
      );
    }

    return (
      <div className="p-10 text-center text-sm text-slate-500">
        <Database className="mx-auto mb-4 h-12 w-12 text-slate-300" />
        Select a model to view its data.
      </div>
    );
  };

  const models = [
    { key: 'users' as ModelType, name: 'Users', icon: Users, count: users.length },
    { key: 'verifications' as ModelType, name: 'Email Verifications', icon: Mail, count: verifications.length },
    { key: 'smtp_accounts' as ModelType, name: 'SMTP Accounts', icon: Server, count: smtpAccounts.length },
    { key: 'templates' as ModelType, name: 'Email Templates', icon: Database, count: templates.length },
  ];

  return (
    <div className="mx-auto max-w-[1280px] space-y-6">
      <div>
        <p className="text-xs font-semibold uppercase tracking-widest text-slate-500">Superuser controls</p>
        <h1 className="mt-2 text-2xl font-bold tracking-tight text-slate-950 sm:text-3xl">Site Administration</h1>
        <p className="mt-1 text-sm text-slate-500">Inspect users, verifications, SMTP accounts, and templates.</p>
      </div>

      {error && (
        <div className="border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[300px_1fr]">
        <aside className="border border-slate-200 bg-white p-3">
          <p className="px-2 pb-3 text-xs font-semibold uppercase tracking-widest text-slate-500">Models</p>
          <nav className="space-y-1">
            {models.map((model) => (
              <button
                className={`flex w-full items-center justify-between px-3 py-3 text-left text-sm transition-colors ${
                  selectedModel === model.key ? 'bg-slate-950 text-white' : 'text-slate-700 hover:bg-slate-100 hover:text-slate-950'
                }`}
                key={model.key}
                onClick={() => loadModel(model.key)}
                type="button"
              >
                <span className="flex min-w-0 items-center gap-3">
                  <model.icon className="h-4 w-4 shrink-0" />
                  <span className="truncate">{model.name}</span>
                </span>
                <span className="flex items-center gap-2">
                  <span className="text-xs opacity-70">{model.count}</span>
                  <ChevronRight className="h-4 w-4 opacity-60" />
                </span>
              </button>
            ))}
          </nav>
        </aside>

        <section className="border border-slate-200 bg-white">
          {selectedModel && (
            <div className="border-b border-slate-200 bg-slate-50 px-4 py-3">
              <h2 className="text-sm font-bold capitalize text-slate-950">{selectedModel.replace('_', ' ')}</h2>
            </div>
          )}
          {renderTable()}
        </section>
      </div>
    </div>
  );
}

function DataTable({ headers, children }: { headers: string[]; children: React.ReactNode }) {
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full border-collapse text-left">
        <thead className="bg-slate-50 text-xs font-semibold uppercase tracking-widest text-slate-500">
          <tr>
            {headers.map((header, index) => (
              <th className={`border-b border-slate-200 px-4 py-3 ${index === headers.length - 1 ? 'text-right' : ''}`} key={header}>
                {header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-200">{children}</tbody>
      </table>
    </div>
  );
}
