import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Activity,
  AlertTriangle,
  CheckCircle,
  Clock,
  Download,
  Eye,
  Filter,
  LayoutGrid,
  List,
  Mail,
  Plus,
  Settings,
  SlidersHorizontal,
  XCircle,
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { HealthResponse, VerificationRecord, VerificationStats } from '../types';

const statusStyles: Record<string, string> = {
  valid: 'bg-emerald-50 text-emerald-700',
  invalid: 'bg-red-50 text-red-700',
  bounced: 'bg-red-50 text-red-700',
  pending_bounce_check: 'bg-amber-50 text-amber-700',
  greylisted: 'bg-orange-50 text-orange-700',
  disposable: 'bg-sky-50 text-sky-700',
  error: 'bg-slate-100 text-slate-700',
  unknown: 'bg-slate-100 text-slate-700',
};

const formatStatus = (status: string) => status.replace(/_/g, ' ');

const formatDate = (timestamp: number) => {
  if (!timestamp) return '-';
  return new Date(timestamp * 1000).toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
};

const confidencePercent = (confidence: VerificationRecord['confidence'] | string) => {
  if (confidence === 'high') return 92;
  if (confidence === 'medium') return 58;
  return 28;
};

export function Dashboard() {
  const { config } = useAuth();
  const [stats, setStats] = useState<VerificationStats | null>(null);
  const [recentVerifications, setRecentVerifications] = useState<VerificationRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [health, setHealth] = useState<HealthResponse | null>(null);

  const loadData = useCallback(async (silent = false) => {
    if (!silent) {
      setLoading(true);
    }
    try {
      const [statsData, healthData, verificationData] = await Promise.all([
        api.getVerificationStats(config),
        api.getHealth(config).catch(() => null),
        api.listVerifications(config, 8, 0).catch(() => ({ items: [] as VerificationRecord[] })),
      ]);

      setStats(statsData);
      setHealth(healthData);
      setRecentVerifications(verificationData.items || []);
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load dashboard data');
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  }, [config]);

  useEffect(() => {
    void loadData();
    const refreshTimer = window.setInterval(() => {
      void loadData(true);
    }, 15000);

    return () => window.clearInterval(refreshTimer);
  }, [loadData]);

  const validCount = stats?.by_status?.valid || 0;
  const invalidCount = (stats?.by_status?.invalid || 0) + (stats?.by_status?.bounced || 0);
  const pendingCount = (stats?.by_status?.pending_bounce_check || 0) + (stats?.by_status?.greylisted || 0);
  const totalCount = stats?.total || 0;
  const validRate = totalCount > 0 ? Math.round((validCount / totalCount) * 100) : 0;
  const riskRate = totalCount > 0 ? Math.round((invalidCount / totalCount) * 100) : 0;
  const directSmtpStatus = health?.direct_smtp_status || 'unknown';
  const statusEntries = Object.entries(stats?.by_status || {}).sort(([, a], [, b]) => b - a);

  if (loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-slate-200 border-t-slate-950" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-[1280px] space-y-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-slate-500">Email verification</p>
          <div className="mt-2 flex flex-wrap items-center gap-3">
            <h1 className="text-2xl font-bold tracking-tight text-slate-950 sm:text-3xl">Verification Console</h1>
            <span className="border border-slate-200 bg-white px-2.5 py-1 text-sm text-slate-500">
              {totalCount.toLocaleString()} records
            </span>
          </div>
          <p className="mt-1 text-sm text-slate-500">Auto-refreshes every 15 seconds.</p>
        </div>

        <div className="flex flex-wrap gap-2">
          <Link
            className="inline-flex h-9 items-center gap-2 border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-950 transition-colors hover:bg-slate-50"
            to="/dashboard/settings"
          >
            <Settings className="h-4 w-4" />
            Settings
          </Link>
          <Link
            className="inline-flex h-9 items-center gap-2 border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-950 transition-colors hover:bg-slate-50"
            to="/dashboard/status"
          >
            <Download className="h-4 w-4" />
            All Records
          </Link>
          <Link
            className="inline-flex h-9 items-center gap-2 bg-slate-950 px-4 text-sm font-semibold text-white transition-colors hover:bg-slate-800"
            to="/dashboard/playground"
          >
            <Plus className="h-4 w-4" />
            Verify Email
          </Link>
        </div>
      </div>

      {error && (
        <div className="border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <section className="border border-slate-200 bg-white p-5">
          <div className="flex items-start justify-between">
            <p className="text-sm text-slate-500">Total Verified</p>
            <span className="flex h-8 w-8 items-center justify-center bg-slate-100 text-slate-700">
              <Mail className="h-4 w-4" />
            </span>
          </div>
          <p className="mt-5 text-3xl font-bold tracking-tight text-slate-950">{totalCount.toLocaleString()}</p>
          <p className="mt-2 text-sm text-emerald-700">Live verification history</p>
        </section>

        <section className="border border-slate-200 bg-white p-5">
          <div className="flex items-start justify-between">
            <p className="text-sm text-slate-500">Valid Emails</p>
            <span className="flex h-8 w-8 items-center justify-center bg-emerald-50 text-emerald-700">
              <CheckCircle className="h-4 w-4" />
            </span>
          </div>
          <p className="mt-5 text-3xl font-bold tracking-tight text-slate-950">{validCount.toLocaleString()}</p>
          <p className="mt-2 text-sm text-emerald-700">{validRate}% acceptance rate</p>
        </section>

        <section className="border border-slate-200 bg-white p-5">
          <div className="flex items-start justify-between">
            <p className="text-sm text-slate-500">Risky or Bounced</p>
            <span className="flex h-8 w-8 items-center justify-center bg-red-50 text-red-700">
              <XCircle className="h-4 w-4" />
            </span>
          </div>
          <p className="mt-5 text-3xl font-bold tracking-tight text-slate-950">{invalidCount.toLocaleString()}</p>
          <p className="mt-2 text-sm text-red-600">{riskRate}% needs attention</p>
        </section>

        <section className="border border-slate-200 bg-white p-5">
          <div className="flex items-start justify-between">
            <p className="text-sm text-slate-500">Pending Checks</p>
            <span className="flex h-8 w-8 items-center justify-center bg-amber-50 text-amber-700">
              <Clock className="h-4 w-4" />
            </span>
          </div>
          <p className="mt-5 text-3xl font-bold tracking-tight text-slate-950">{pendingCount.toLocaleString()}</p>
          <p className="mt-2 text-sm text-amber-700">Awaiting bounce signals</p>
        </section>
      </div>

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-[1fr_340px]">
        <section className="border border-slate-200 bg-white">
          <div className="flex flex-col gap-3 border-b border-slate-200 p-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-4">
              <button className="border-b-2 border-slate-950 pb-3 text-sm font-semibold text-slate-950" type="button">
                Recent Checks
              </button>
              <Link className="pb-3 text-sm font-semibold text-slate-500 hover:text-slate-950" to="/dashboard/status">
                Full History
              </Link>
            </div>
            <div className="flex flex-wrap gap-2">
              <button
                className="inline-flex h-9 items-center gap-2 border border-slate-200 bg-white px-3 text-sm font-medium text-slate-800 hover:bg-slate-50"
                type="button"
              >
                <Filter className="h-4 w-4" />
                Filter
              </button>
              <button
                className="inline-flex h-9 items-center gap-2 border border-slate-200 bg-white px-3 text-sm font-medium text-slate-800 hover:bg-slate-50"
                type="button"
              >
                <SlidersHorizontal className="h-4 w-4" />
                Columns
              </button>
              <span className="hidden items-center gap-2 px-2 text-slate-500 sm:inline-flex">
                <LayoutGrid className="h-4 w-4" />
                <List className="h-4 w-4" />
              </span>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="min-w-full border-collapse text-left">
              <thead className="bg-slate-50 text-xs font-semibold uppercase tracking-widest text-slate-500">
                <tr>
                  <th className="w-10 border-b border-slate-200 px-4 py-3">
                    <input className="h-4 w-4 border-slate-300" type="checkbox" />
                  </th>
                  <th className="border-b border-slate-200 px-4 py-3">Email</th>
                  <th className="border-b border-slate-200 px-4 py-3">Source</th>
                  <th className="hidden border-b border-slate-200 px-4 py-3 md:table-cell">Path</th>
                  <th className="border-b border-slate-200 px-4 py-3">Confidence</th>
                  <th className="border-b border-slate-200 px-4 py-3">Status</th>
                  <th className="hidden border-b border-slate-200 px-4 py-3 lg:table-cell">Checked</th>
                  <th className="border-b border-slate-200 px-4 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200">
                {recentVerifications.length === 0 ? (
                  <tr>
                    <td className="px-4 py-10 text-center text-sm text-slate-500" colSpan={8}>
                      No verification records yet.
                    </td>
                  </tr>
                ) : (
                  recentVerifications.map((item) => {
                    const percent = confidencePercent(item.confidence);
                    const status = item.status || 'unknown';

                    return (
                      <tr className="hover:bg-slate-50" key={item.id}>
                        <td className="px-4 py-3">
                          <input className="h-4 w-4 border-slate-300" type="checkbox" />
                        </td>
                        <td className="max-w-[260px] px-4 py-3">
                          <p className="truncate text-sm font-semibold text-slate-950">{item.email}</p>
                          <p className="truncate text-xs text-slate-500">{item.reason_code || item.message || 'Verification result'}</p>
                        </td>
                        <td className="px-4 py-3 text-sm text-slate-700">{item.source || '-'}</td>
                        <td className="hidden px-4 py-3 text-sm capitalize text-slate-700 md:table-cell">
                          {(item.verification_path || 'unknown').replace(/_/g, ' ')}
                        </td>
                        <td className="min-w-[150px] px-4 py-3">
                          <div className="flex items-center gap-3">
                            <div className="h-1.5 w-20 bg-slate-100">
                              <div className="h-1.5 bg-slate-950" style={{ width: `${percent}%` }} />
                            </div>
                            <span className="text-xs font-medium capitalize text-slate-600">{item.confidence || 'low'}</span>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <span className={`inline-flex px-2.5 py-1 text-xs font-semibold capitalize ${statusStyles[status] || statusStyles.unknown}`}>
                            {formatStatus(status)}
                          </span>
                          {!item.finalized && item.next_check_at > 0 && (
                            <p className="mt-1 text-xs text-amber-700">
                              Next {new Date(item.next_check_at * 1000).toLocaleTimeString()}
                            </p>
                          )}
                        </td>
                        <td className="hidden whitespace-nowrap px-4 py-3 text-sm text-slate-500 lg:table-cell">
                          {formatDate(item.last_checked_at || item.created_at)}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <Link
                            className="inline-flex h-8 w-8 items-center justify-center text-slate-600 hover:bg-slate-100 hover:text-slate-950"
                            title="View status"
                            to="/dashboard/status"
                          >
                            <Eye className="h-4 w-4" />
                          </Link>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </section>

        <aside className="space-y-6">
          <section className="border border-slate-200 bg-white p-5">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-bold text-slate-950">System Status</h2>
              <Activity className="h-4 w-4 text-slate-500" />
            </div>
            <div className="mt-5 space-y-4">
              <div className="flex items-center justify-between border border-slate-200 p-3">
                <div>
                  <p className="text-sm font-semibold text-slate-950">API Status</p>
                  <p className="text-xs text-slate-500">Request endpoint</p>
                </div>
                <span className="bg-emerald-50 px-2 py-1 text-xs font-semibold text-emerald-700">Online</span>
              </div>
              <div className="flex items-center justify-between border border-slate-200 p-3">
                <div>
                  <p className="text-sm font-semibold text-slate-950">Direct SMTP</p>
                  <p className="text-xs text-slate-500">Verification path</p>
                </div>
                <span
                  className={`px-2 py-1 text-xs font-semibold ${
                    directSmtpStatus === 'available'
                      ? 'bg-emerald-50 text-emerald-700'
                      : directSmtpStatus === 'degraded'
                        ? 'bg-red-50 text-red-700'
                        : 'bg-amber-50 text-amber-700'
                  }`}
                >
                  {formatStatus(directSmtpStatus)}
                </span>
              </div>
              {health?.message && (
                <p className="flex gap-2 text-sm leading-6 text-slate-500">
                  <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-600" />
                  {health.message}
                </p>
              )}
            </div>
          </section>

          <section className="border border-slate-200 bg-white p-5">
            <h2 className="text-base font-bold text-slate-950">Status Breakdown</h2>
            <div className="mt-5 space-y-4">
              {statusEntries.length === 0 ? (
                <p className="text-sm text-slate-500">No statuses available yet.</p>
              ) : (
                statusEntries.map(([status, count]) => {
                  const width = totalCount > 0 ? (count / totalCount) * 100 : 0;
                  return (
                    <div key={status}>
                      <div className="mb-2 flex items-center justify-between gap-3">
                        <span className="truncate text-sm capitalize text-slate-600">{formatStatus(status)}</span>
                        <span className="text-sm font-semibold text-slate-950">{count}</span>
                      </div>
                      <div className="h-1.5 bg-slate-100">
                        <div className="h-1.5 bg-slate-950" style={{ width: `${width}%` }} />
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </section>
        </aside>
      </div>
    </div>
  );
}
