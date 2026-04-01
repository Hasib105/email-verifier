import { useState } from 'react';
import { Database, Shield, ShieldOff, Trash2, Users } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { User, VerificationRecord } from '../types';

type ModelType = 'users' | 'verifications' | null;

export function AdminPanel() {
  const { config, user } = useAuth();
  const [selectedModel, setSelectedModel] = useState<ModelType>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [users, setUsers] = useState<User[]>([]);
  const [verifications, setVerifications] = useState<VerificationRecord[]>([]);

  if (!user?.is_superuser) {
    return (
      <div className="flex flex-col items-center justify-center py-16">
        <ShieldOff className="w-16 h-16 text-gray-400 mb-4" />
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Access Denied</h2>
        <p className="text-gray-500 text-center">Only superusers can access the administration panel.</p>
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
      } else {
        const response = await api.adminListVerifications(config);
        setVerifications(response.items || []);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load admin data');
    } finally {
      setLoading(false);
    }
  };

  const handleToggleSuperuser = async (target: User) => {
    if (!confirm(`Change superuser access for ${target.email}?`)) return;
    try {
      await api.adminUpdateUser(config, target.id, { is_superuser: !target.is_superuser });
      await loadModel('users');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update user');
    }
  };

  const handleDeleteUser = async (id: string) => {
    if (!confirm('Delete this user?')) return;
    try {
      await api.adminDeleteUser(config, id);
      await loadModel('users');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete user');
    }
  };

  const handleDeleteVerification = async (id: string) => {
    if (!confirm('Delete this verification?')) return;
    try {
      await api.adminDeleteVerification(config, id);
      await loadModel('verifications');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete verification');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Shield className="w-8 h-8 text-purple-600" />
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold text-gray-900 tracking-tight">Administration</h2>
          <p className="text-sm text-gray-500">Users and verification history only. Legacy probe-email operations were removed in V2.</p>
        </div>
      </div>

      {error && <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</div>}

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        <div className="bg-white border rounded-xl shadow-sm p-4">
          <h3 className="font-bold text-sm uppercase text-gray-500 mb-4">Models</h3>
          <nav className="space-y-1">
            <button
              onClick={() => void loadModel('users')}
              className={`w-full rounded-lg px-3 py-3 text-left text-sm ${selectedModel === 'users' ? 'bg-purple-100 text-purple-900' : 'hover:bg-gray-50'}`}
            >
              <div className="flex items-center gap-3">
                <Users className="w-4 h-4" />
                <span>Users</span>
              </div>
            </button>
            <button
              onClick={() => void loadModel('verifications')}
              className={`w-full rounded-lg px-3 py-3 text-left text-sm ${selectedModel === 'verifications' ? 'bg-purple-100 text-purple-900' : 'hover:bg-gray-50'}`}
            >
              <div className="flex items-center gap-3">
                <Database className="w-4 h-4" />
                <span>Verifications</span>
              </div>
            </button>
          </nav>
        </div>

        <div className="lg:col-span-3 bg-white border rounded-xl shadow-sm overflow-hidden">
          {!selectedModel ? (
            <div className="p-10 text-center text-gray-500">Select a model to inspect.</div>
          ) : loading ? (
            <div className="p-10 text-center text-gray-500">Loading…</div>
          ) : selectedModel === 'users' ? (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Email</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Role</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Created</th>
                    <th className="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {users.map((item) => (
                    <tr key={item.id} className="hover:bg-gray-50">
                      <td className="px-4 py-4 text-sm">{item.email}</td>
                      <td className="px-4 py-4 text-sm">{item.is_superuser ? 'Superuser' : 'User'}</td>
                      <td className="px-4 py-4 text-sm text-gray-500">{new Date(item.created_at * 1000).toLocaleString()}</td>
                      <td className="px-4 py-4 text-right">
                        <div className="flex items-center justify-end gap-3">
                          <button onClick={() => void handleToggleSuperuser(item)} className="text-sm text-purple-700 hover:underline">
                            {item.is_superuser ? 'Demote' : 'Promote'}
                          </button>
                          <button onClick={() => void handleDeleteUser(item.id)} className="text-gray-400 hover:text-red-600">
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Email</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Classification</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Confidence</th>
                    <th className="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {verifications.map((item) => (
                    <tr key={item.id} className="hover:bg-gray-50">
                      <td className="px-4 py-4 text-sm">{item.email}</td>
                      <td className="px-4 py-4 text-sm capitalize">{item.classification.replace(/_/g, ' ')}</td>
                      <td className="px-4 py-4 text-sm">{item.confidence_score}</td>
                      <td className="px-4 py-4 text-right">
                        <button onClick={() => void handleDeleteVerification(item.id)} className="text-gray-400 hover:text-red-600">
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
