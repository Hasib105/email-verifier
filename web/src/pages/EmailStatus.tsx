import { useCallback, useEffect, useState } from 'react';
import { RefreshCw, Trash2 } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { VerificationRecord } from '../types';

const statusBadge = (status: string) => {
  const statusMap: Record<string, string> = {
    valid: 'bg-emerald-50 text-emerald-700',
    invalid: 'bg-red-50 text-red-700',
    bounced: 'bg-red-50 text-red-700',
    pending_bounce_check: 'bg-amber-50 text-amber-700',
    greylisted: 'bg-orange-50 text-orange-700',
    error: 'bg-slate-100 text-slate-700',
    unknown: 'bg-slate-100 text-slate-700',
    disposable: 'bg-sky-50 text-sky-700',
  };
  return statusMap[status] || statusMap.unknown;
};

const formatDate = (timestamp: number) => {
  if (!timestamp) return '-';
  return new Date(timestamp * 1000).toLocaleString();
};

export function EmailStatus() {
  const { config } = useAuth();
  const [verifications, setVerifications] = useState<VerificationRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [error, setError] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = 20;

  const loadVerifications = useCallback(async (silent = false) => {
    if (!silent) {
      setLoading(true);
    }
    try {
      const response = await api.listVerifications(config, limit, offset);
      setVerifications(response.items || []);
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load verifications');
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  }, [config, offset]);

  useEffect(() => {
    void loadVerifications();
    const refreshTimer = window.setInterval(() => {
      void loadVerifications(true);
    }, 15000);

    return () => window.clearInterval(refreshTimer);
  }, [loadVerifications]);

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this email status record?')) {
      return;
    }

    setDeletingId(id);
    try {
      await api.deleteVerification(config, id);
      await loadVerifications();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete verification');
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className="mx-auto max-w-[1280px] space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-slate-500">Verification history</p>
          <h1 className="mt-2 text-2xl font-bold tracking-tight text-slate-950 sm:text-3xl">Email Status</h1>
          <p className="mt-1 text-sm text-slate-500">Review every checked address and its latest signal state. Auto-refreshes every 15 seconds.</p>
        </div>
        <button
          className="inline-flex h-9 items-center justify-center gap-2 border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-950 hover:bg-slate-50 disabled:opacity-50"
          disabled={loading}
          onClick={() => loadVerifications()}
          type="button"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {error && (
        <div className="border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      )}

      <section className="border border-slate-200 bg-white">
        <div className="overflow-x-auto">
          <table className="min-w-full border-collapse text-left">
            <thead className="bg-slate-50 text-xs font-semibold uppercase tracking-widest text-slate-500">
              <tr>
                <th className="border-b border-slate-200 px-4 py-3">Email</th>
                <th className="border-b border-slate-200 px-4 py-3">Status</th>
                <th className="hidden border-b border-slate-200 px-4 py-3 md:table-cell">Confidence</th>
                <th className="hidden border-b border-slate-200 px-4 py-3 sm:table-cell">Source</th>
                <th className="hidden border-b border-slate-200 px-4 py-3 lg:table-cell">Message</th>
                <th className="hidden border-b border-slate-200 px-4 py-3 lg:table-cell">Checked</th>
                <th className="border-b border-slate-200 px-4 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200">
              {loading ? (
                <tr>
                  <td className="px-4 py-10 text-center" colSpan={7}>
                    <div className="mx-auto h-6 w-6 animate-spin rounded-full border-2 border-slate-200 border-t-slate-950" />
                  </td>
                </tr>
              ) : verifications.length === 0 ? (
                <tr>
                  <td className="px-4 py-10 text-center text-sm text-slate-500" colSpan={7}>
                    No verifications found.
                  </td>
                </tr>
              ) : (
                verifications.map((item) => {
                  const status = item.status || 'unknown';
                  return (
                    <tr className="hover:bg-slate-50" key={item.id}>
                      <td className="max-w-[260px] px-4 py-3">
                        <p className="truncate text-sm font-semibold text-slate-950">{item.email}</p>
                        <p className="truncate text-xs text-slate-500">{item.reason_code || 'verification record'}</p>
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex px-2.5 py-1 text-xs font-semibold capitalize ${statusBadge(status)}`}>
                          {status.replace(/_/g, ' ')}
                        </span>
                        {!item.finalized && item.next_check_at > 0 && (
                          <p className="mt-1 text-xs text-amber-700">
                            Next check {new Date(item.next_check_at * 1000).toLocaleTimeString()}
                          </p>
                        )}
                      </td>
                      <td className="hidden px-4 py-3 text-sm capitalize text-slate-600 md:table-cell">
                        {item.confidence || 'low'}
                      </td>
                      <td className="hidden px-4 py-3 text-sm text-slate-600 sm:table-cell">
                        {item.source || '-'}
                      </td>
                      <td className="hidden max-w-[280px] px-4 py-3 text-sm text-slate-500 lg:table-cell">
                        <p className="truncate">{item.signal_summary || item.message || '-'}</p>
                      </td>
                      <td className="hidden whitespace-nowrap px-4 py-3 text-sm text-slate-500 lg:table-cell">
                        {formatDate(item.last_checked_at)}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <button
                          className="inline-flex h-8 w-8 items-center justify-center text-slate-500 hover:bg-red-50 hover:text-red-600 disabled:opacity-50"
                          disabled={deletingId === item.id}
                          onClick={() => handleDelete(item.id)}
                          title="Delete record"
                          type="button"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>

        {verifications.length > 0 && (
          <div className="flex flex-col gap-3 border-t border-slate-200 bg-slate-50 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
            <span className="text-sm text-slate-500">
              Showing {offset + 1} - {offset + verifications.length}
            </span>
            <div className="flex gap-2">
              <button
                className="h-8 border border-slate-200 bg-white px-3 text-sm font-medium text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                disabled={offset === 0}
                onClick={() => setOffset(Math.max(0, offset - limit))}
                type="button"
              >
                Previous
              </button>
              <button
                className="h-8 border border-slate-200 bg-white px-3 text-sm font-medium text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                disabled={verifications.length < limit}
                onClick={() => setOffset(offset + limit)}
                type="button"
              >
                Next
              </button>
            </div>
          </div>
        )}
      </section>
    </div>
  );
}
