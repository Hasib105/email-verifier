import { useState } from 'react';
import { Users, Mail, Database, Server, ChevronRight, Trash2, Shield, ShieldOff } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { User, SMTPAccount, EmailTemplate, VerificationRecord } from '../types';

type ModelType = 'users' | 'verifications' | 'smtp_accounts' | 'templates' | null;

export function AdminPanel() {
  const { config, user } = useAuth();
  const [selectedModel, setSelectedModel] = useState<ModelType>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // Data states
  const [users, setUsers] = useState<User[]>([]);
  const [verifications, setVerifications] = useState<VerificationRecord[]>([]);
  const [smtpAccounts, setSmtpAccounts] = useState<SMTPAccount[]>([]);
  const [templates, setTemplates] = useState<EmailTemplate[]>([]);

  // Check if current user is superuser
  if (!user?.is_superuser) {
    return (
      <div className="flex flex-col items-center justify-center py-16">
        <ShieldOff className="w-16 h-16 text-gray-400 mb-4" />
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Access Denied</h2>
        <p className="text-gray-500 text-center">
          You don't have permission to access the admin panel.<br />
          Only superusers can view this page.
        </p>
      </div>
    );
  }

  const loadModel = async (model: ModelType) => {
    if (!model) return;
    setLoading(true);
    setError('');
    setSelectedModel(model);

    try {
      switch (model) {
        case 'users':
          const usersRes = await api.adminListUsers(config);
          setUsers(usersRes.items || []);
          break;
        case 'verifications':
          const verificationsRes = await api.adminListVerifications(config);
          setVerifications(verificationsRes.items || []);
          break;
        case 'smtp_accounts':
          const smtpRes = await api.adminListSmtpAccounts(config);
          setSmtpAccounts(smtpRes.items || []);
          break;
        case 'templates':
          const templatesRes = await api.adminListTemplates(config);
          setTemplates(templatesRes.items || []);
          break;
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

  const renderModelList = () => {
    if (loading) {
      return (
        <div className="flex justify-center py-8">
          <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-yellow-500"></div>
        </div>
      );
    }

    switch (selectedModel) {
      case 'users':
        return (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Email</th>
                  <th className="hidden sm:table-cell px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">API Key</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Role</th>
                  <th className="hidden md:table-cell px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Created</th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {users.map((u) => (
                  <tr key={u.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3 text-sm">{u.email}</td>
                    <td className="hidden sm:table-cell px-4 py-3 text-sm font-mono text-gray-500">{u.api_key?.slice(0, 8)}...</td>
                    <td className="px-4 py-3 text-sm">
                      <span className={`px-2 py-1 text-xs font-semibold rounded-full ${
                        u.is_superuser ? 'bg-purple-100 text-purple-800' : 'bg-gray-100 text-gray-800'
                      }`}>
                        {u.is_superuser ? 'Superuser' : 'User'}
                      </span>
                    </td>
                    <td className="hidden md:table-cell px-4 py-3 text-sm text-gray-500">
                      {new Date(u.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex justify-end gap-2">
                        <button
                          onClick={() => handleToggleSuperuser(u.id, u.is_superuser)}
                          className={`p-1 ${u.is_superuser ? 'text-purple-500 hover:text-purple-700' : 'text-gray-400 hover:text-purple-600'}`}
                          title={u.is_superuser ? 'Remove superuser' : 'Make superuser'}
                        >
                          <Shield className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleDeleteUser(u.id)}
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
        );

      case 'verifications':
        return (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Email</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                  <th className="hidden sm:table-cell px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Source</th>
                  <th className="hidden md:table-cell px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Date</th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {verifications.map((v) => (
                  <tr key={v.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3 text-sm font-medium">{v.email}</td>
                    <td className="px-4 py-3 text-sm">
                      <span className={`px-2 py-1 text-xs font-semibold rounded-full ${
                        v.status === 'valid' ? 'bg-green-100 text-green-800' :
                        v.status === 'invalid' ? 'bg-red-100 text-red-800' :
                        v.status === 'pending_bounce_check' ? 'bg-yellow-100 text-yellow-800' :
                        'bg-gray-100 text-gray-800'
                      }`}>
                        {v.status}
                      </span>
                      <div className="text-xs text-gray-500 mt-1 capitalize">
                        {v.confidence || 'low'} confidence
                      </div>
                    </td>
                    <td className="hidden sm:table-cell px-4 py-3 text-sm text-gray-500">{v.source || '-'}</td>
                    <td className="hidden md:table-cell px-4 py-3 text-sm text-gray-500">
                      {new Date(v.created_at * 1000).toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        onClick={() => handleDeleteVerification(v.id)}
                        className="p-1 text-gray-400 hover:text-red-600"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        );

      case 'smtp_accounts':
        return (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Host</th>
                  <th className="hidden sm:table-cell px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Username</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Owner</th>
                  <th className="hidden md:table-cell px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Usage</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {smtpAccounts.map((a) => (
                  <tr key={a.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3 text-sm font-medium">{a.host}:{a.port}</td>
                    <td className="hidden sm:table-cell px-4 py-3 text-sm text-gray-500">{a.username}</td>
                    <td className="px-4 py-3 text-sm text-gray-500">{a.user_id?.slice(0, 8)}...</td>
                    <td className="hidden md:table-cell px-4 py-3 text-sm text-gray-500">
                      {a.sent_today} / {a.daily_limit}
                    </td>
                    <td className="px-4 py-3 text-sm">
                      <span className={`px-2 py-1 text-xs font-semibold rounded-full ${
                        a.active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                      }`}>
                        {a.active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        );

      case 'templates':
        return (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                  <th className="hidden sm:table-cell px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Subject</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Owner</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {templates.map((t) => (
                  <tr key={t.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3 text-sm font-medium">{t.name}</td>
                    <td className="hidden sm:table-cell px-4 py-3 text-sm text-gray-500">{t.subject_template}</td>
                    <td className="px-4 py-3 text-sm text-gray-500">{t.user_id?.slice(0, 8)}...</td>
                    <td className="px-4 py-3 text-sm">
                      <span className={`px-2 py-1 text-xs font-semibold rounded-full ${
                        t.active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                      }`}>
                        {t.active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        );

      default:
        return null;
    }
  };

  const models = [
    { key: 'users' as ModelType, name: 'Users', icon: Users, count: users.length },
    { key: 'verifications' as ModelType, name: 'Email Verifications', icon: Mail, count: verifications.length },
    { key: 'smtp_accounts' as ModelType, name: 'SMTP Accounts', icon: Server, count: smtpAccounts.length },
    { key: 'templates' as ModelType, name: 'Email Templates', icon: Database, count: templates.length },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Shield className="w-8 h-8 text-purple-600" />
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold text-gray-900 tracking-tight">Site Administration</h2>
          <p className="text-sm text-gray-500">Superuser access to all system models.</p>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-red-50 border border-red-200 text-red-700 rounded-xl text-sm">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Model Selector */}
        <div className="bg-white border rounded-xl shadow-sm p-4">
          <h3 className="font-bold text-sm uppercase text-gray-500 mb-4">Models</h3>
          <nav className="space-y-1">
            {models.map((model) => (
              <button
                key={model.key}
                onClick={() => loadModel(model.key)}
                className={`w-full flex items-center justify-between p-3 rounded-lg text-sm transition-colors ${
                  selectedModel === model.key
                    ? 'bg-purple-100 text-purple-900'
                    : 'hover:bg-gray-50 text-gray-700'
                }`}
              >
                <div className="flex items-center gap-3">
                  <model.icon className="w-4 h-4" />
                  <span>{model.name}</span>
                </div>
                <ChevronRight className="w-4 h-4 text-gray-400" />
              </button>
            ))}
          </nav>
        </div>

        {/* Model Data */}
        <div className="lg:col-span-3 bg-white border rounded-xl shadow-sm overflow-hidden">
          {selectedModel ? (
            <>
              <div className="px-4 py-3 bg-gray-50 border-b">
                <h3 className="font-semibold capitalize">{selectedModel.replace('_', ' ')}</h3>
              </div>
              {renderModelList()}
            </>
          ) : (
            <div className="p-8 text-center text-gray-500">
              <Database className="w-12 h-12 mx-auto mb-4 text-gray-400" />
              <p>Select a model from the left panel to view its data.</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
