import { useState, useEffect } from 'react';
import { Activity, Mail, CheckCircle, AlertTriangle, XCircle, Clock } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { VerificationStats } from '../types';

export function Dashboard() {
  const { config } = useAuth();
  const [stats, setStats] = useState<VerificationStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [torStatus, setTorStatus] = useState<'checking' | 'online' | 'offline'>('checking');

  useEffect(() => {
    const loadData = async () => {
      try {
        const [statsData, torData] = await Promise.all([
          api.getVerificationStats(config),
          api.getTorStatus(config).catch(() => ({ is_tor: false })),
        ]);
        setStats(statsData);
        setTorStatus(torData.is_tor ? 'online' : 'offline');
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load data');
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, [config]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-yellow-500"></div>
      </div>
    );
  }

  const validCount = stats?.by_status?.valid || 0;
  const invalidCount = (stats?.by_status?.invalid || 0) + (stats?.by_status?.bounced || 0);
  const pendingCount = (stats?.by_status?.pending_bounce_check || 0) + (stats?.by_status?.greylisted || 0);
  const totalCount = stats?.total || 0;

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-2xl sm:text-3xl font-bold text-gray-900 tracking-tight">Overview Dashboard</h2>
        <p className="text-sm text-gray-500 mt-1">Metrics and statistics about your verified emails.</p>
      </div>

      {error && (
        <div className="p-4 bg-red-50 border border-red-200 text-red-700 rounded-xl text-sm">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-6">
        <div className="bg-gradient-to-br from-yellow-50 to-yellow-100 p-4 sm:p-6 rounded-xl border border-yellow-200 shadow-sm">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-sm font-semibold tracking-wider text-yellow-800">Total Verified</h3>
            <Mail className="text-yellow-600 w-5 h-5"/>
          </div>
          <p className="text-3xl sm:text-4xl font-black text-yellow-900">{totalCount.toLocaleString()}</p>
        </div>

        <div className="bg-white p-4 sm:p-6 rounded-xl border shadow-sm">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-sm font-semibold tracking-wider text-gray-500">Valid Emails</h3>
            <CheckCircle className="text-green-500 w-5 h-5"/>
          </div>
          <p className="text-3xl sm:text-4xl font-black text-gray-900">{validCount.toLocaleString()}</p>
        </div>
        
        <div className="bg-white p-4 sm:p-6 rounded-xl border shadow-sm">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-sm font-semibold tracking-wider text-gray-500">Invalid/Bounced</h3>
            <XCircle className="text-red-500 w-5 h-5"/>
          </div>
          <p className="text-3xl sm:text-4xl font-black text-gray-900">{invalidCount.toLocaleString()}</p>
        </div>

        <div className="bg-white p-4 sm:p-6 rounded-xl border shadow-sm">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-sm font-semibold tracking-wider text-gray-500">Pending</h3>
            <Clock className="text-orange-500 w-5 h-5"/>
          </div>
          <p className="text-3xl sm:text-4xl font-black text-gray-900">{pendingCount.toLocaleString()}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white border rounded-xl shadow-sm p-6">
          <h3 className="text-lg font-bold mb-4">System Status</h3>
          <div className="space-y-4">
            <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
              <div className="flex items-center gap-3">
                <Activity className="text-blue-500 w-5 h-5"/>
                <span className="text-sm font-medium">API Status</span>
              </div>
              <span className="px-2 py-1 text-xs font-semibold rounded-full bg-green-100 text-green-800">
                Online
              </span>
            </div>
            <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
              <div className="flex items-center gap-3">
                <AlertTriangle className="text-purple-500 w-5 h-5"/>
                <span className="text-sm font-medium">Tor Proxy</span>
              </div>
              <span className={`px-2 py-1 text-xs font-semibold rounded-full ${
                torStatus === 'online' ? 'bg-green-100 text-green-800' :
                torStatus === 'offline' ? 'bg-red-100 text-red-800' :
                'bg-yellow-100 text-yellow-800'
              }`}>
                {torStatus === 'checking' ? 'Checking...' : torStatus === 'online' ? 'Connected' : 'Disconnected'}
              </span>
            </div>
          </div>
        </div>

        <div className="bg-white border rounded-xl shadow-sm p-6">
          <h3 className="text-lg font-bold mb-4">Status Breakdown</h3>
          {stats?.by_status && Object.keys(stats.by_status).length > 0 ? (
            <div className="space-y-3">
              {Object.entries(stats.by_status).map(([status, count]) => (
                <div key={status} className="flex items-center justify-between">
                  <span className="text-sm text-gray-600 capitalize">{status.replace(/_/g, ' ')}</span>
                  <div className="flex items-center gap-2">
                    <div className="w-24 sm:w-32 bg-gray-200 rounded-full h-2">
                      <div 
                        className="bg-yellow-500 h-2 rounded-full" 
                        style={{ width: `${totalCount > 0 ? (count / totalCount) * 100 : 0}%` }}
                      ></div>
                    </div>
                    <span className="text-sm font-medium w-12 text-right">{count}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-gray-400 text-sm">No verifications yet</p>
          )}
        </div>
      </div>
    </div>
  );
}
