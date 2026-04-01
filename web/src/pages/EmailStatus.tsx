import { useCallback, useEffect, useState } from 'react';
import { ChevronRight, Loader2, RefreshCw, Trash2 } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { VerificationRecord, VerifyResponse } from '../types';

export function EmailStatus() {
  const { config } = useAuth();
  const [items, setItems] = useState<VerificationRecord[]>([]);
  const [selected, setSelected] = useState<VerifyResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState('');
  const [offset, setOffset] = useState(0);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const limit = 20;

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.listVerifications(config, limit, offset);
      setItems(response.items || []);
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load verifications');
    } finally {
      setLoading(false);
    }
  }, [config, limit, offset]);

  useEffect(() => {
    void load();
  }, [load]);

  const loadDetail = async (id: string) => {
    setDetailLoading(true);
    try {
      const response = await api.getVerification(config, id);
      setSelected(response);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load verification detail');
    } finally {
      setDetailLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this verification?')) return;
    setDeletingId(id);
    try {
      await api.deleteVerification(config, id);
      if (selected?.id === id) {
        setSelected(null);
      }
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete verification');
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold text-gray-900 tracking-tight">Verification History</h2>
          <p className="text-sm text-gray-500 mt-1">Inspect classifications, callout traces, and enrichment evidence.</p>
        </div>
        <button
          onClick={() => void load()}
          disabled={loading}
          className="inline-flex items-center gap-2 rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium hover:bg-gray-200 disabled:opacity-50"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</div>
      )}

      <div className="grid grid-cols-1 xl:grid-cols-[1.5fr,1fr] gap-6">
        <div className="bg-white border rounded-xl shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Email</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Classification</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Confidence</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">State</th>
                  <th className="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 bg-white">
                {loading ? (
                  <tr>
                    <td colSpan={5} className="px-6 py-10 text-center">
                      <Loader2 className="w-5 h-5 animate-spin mx-auto text-yellow-500" />
                    </td>
                  </tr>
                ) : items.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-6 py-10 text-center text-sm text-gray-500">No verification history yet.</td>
                  </tr>
                ) : (
                  items.map((item) => (
                    <tr key={item.id} className="hover:bg-gray-50">
                      <td className="px-4 py-4 text-sm font-medium text-gray-900">
                        <button onClick={() => void loadDetail(item.id)} className="text-left">
                          <div>{item.email}</div>
                          <div className="text-xs text-gray-500">{item.domain}</div>
                        </button>
                      </td>
                      <td className="px-4 py-4 text-sm">
                        <span className={`rounded-full px-2.5 py-1 text-xs font-semibold ${classificationBadge(item.classification)}`}>
                          {item.classification.replace(/_/g, ' ')}
                        </span>
                      </td>
                      <td className="px-4 py-4 text-sm text-gray-700">{item.confidence_score}</td>
                      <td className="px-4 py-4 text-sm text-gray-700 capitalize">{item.state}</td>
                      <td className="px-4 py-4 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button onClick={() => void loadDetail(item.id)} className="p-1 text-gray-400 hover:text-blue-600" title="View detail">
                            <ChevronRight className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => void handleDelete(item.id)}
                            disabled={deletingId === item.id}
                            className="p-1 text-gray-400 hover:text-red-600 disabled:opacity-50"
                            title="Delete"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {items.length > 0 && (
            <div className="flex items-center justify-between border-t bg-gray-50 px-4 py-3 text-sm text-gray-600">
              <span>Showing {offset + 1} - {offset + items.length}</span>
              <div className="flex gap-2">
                <button
                  onClick={() => setOffset(Math.max(0, offset - limit))}
                  disabled={offset === 0}
                  className="rounded-md border px-3 py-1 disabled:opacity-50"
                >
                  Previous
                </button>
                <button
                  onClick={() => setOffset(offset + limit)}
                  disabled={items.length < limit}
                  className="rounded-md border px-3 py-1 disabled:opacity-50"
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </div>

        <div className="bg-white border rounded-xl shadow-sm p-6">
          {detailLoading ? (
            <div className="flex items-center justify-center py-20">
              <Loader2 className="w-5 h-5 animate-spin text-yellow-500" />
            </div>
          ) : selected ? (
            <div className="space-y-5">
              <div>
                <p className="text-xs uppercase tracking-wider text-gray-500">Selected Verification</p>
                <h3 className="mt-1 text-lg font-bold text-gray-900 break-all">{selected.email}</h3>
              </div>

              <div className="grid grid-cols-2 gap-3 text-sm">
                <DetailCard label="Classification" value={selected.classification} />
                <DetailCard label="Confidence" value={`${selected.confidence_score}/100`} />
                <DetailCard label="Risk" value={selected.risk_level} />
                <DetailCard label="State" value={selected.state} />
              </div>

              <TextBlock title="Protocol Summary" body={selected.protocol_summary} />
              <TextBlock title="Reason Codes" body={selected.reason_codes.join(', ') || 'None'} />
              <TextBlock title="Enrichment Summary" body={selected.enrichment_summary || 'Pending or not required.'} />

              <div>
                <h4 className="text-sm font-semibold text-gray-900 mb-2">Evidence</h4>
                <div className="space-y-2">
                  {(selected.evidence || []).length === 0 ? (
                    <p className="text-sm text-gray-500">No evidence stored.</p>
                  ) : (
                    selected.evidence?.map((item) => (
                      <div key={item.id} className="rounded-lg border border-gray-200 p-3 text-sm">
                        <div className="flex items-center justify-between gap-3">
                          <span className="font-medium text-gray-900">{item.signal.replace(/_/g, ' ')}</span>
                          <span className={`text-xs font-semibold ${item.weight >= 0 ? 'text-green-700' : 'text-red-700'}`}>
                            {item.weight >= 0 ? '+' : ''}{item.weight}
                          </span>
                        </div>
                        <p className="mt-1 text-gray-600">{item.summary}</p>
                      </div>
                    ))
                  )}
                </div>
              </div>

              <div>
                <h4 className="text-sm font-semibold text-gray-900 mb-2">Callouts</h4>
                <div className="space-y-2">
                  {(selected.callouts || []).length === 0 ? (
                    <p className="text-sm text-gray-500">No callout trace stored.</p>
                  ) : (
                    selected.callouts?.map((item, index) => (
                      <div key={`${item.id}-${index}`} className="rounded-lg border border-gray-200 p-3 text-sm text-gray-700">
                        <div className="flex items-center justify-between gap-3">
                          <span className="font-medium text-gray-900">{item.stage} via {item.smtp_host}</span>
                          <span className="text-xs uppercase tracking-wider text-gray-500">{item.outcome}</span>
                        </div>
                        <p className="mt-1">{item.smtp_code > 0 ? `${item.smtp_code} ` : ''}{item.smtp_message}</p>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>
          ) : (
            <div className="py-20 text-center text-sm text-gray-500">Select a verification to inspect evidence and callout traces.</div>
          )}
        </div>
      </div>
    </div>
  );
}

function classificationBadge(classification: VerificationRecord['classification']) {
  switch (classification) {
    case 'deliverable':
      return 'bg-green-100 text-green-800';
    case 'undeliverable':
      return 'bg-red-100 text-red-800';
    case 'accept_all':
      return 'bg-amber-100 text-amber-800';
    default:
      return 'bg-gray-100 text-gray-800';
  }
}

function DetailCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-gray-200 bg-gray-50 p-3">
      <p className="text-xs uppercase tracking-wider text-gray-500">{label}</p>
      <p className="mt-1 text-sm font-semibold text-gray-900 capitalize">{value}</p>
    </div>
  );
}

function TextBlock({ title, body }: { title: string; body: string }) {
  return (
    <div>
      <h4 className="text-sm font-semibold text-gray-900 mb-2">{title}</h4>
      <div className="rounded-xl border border-gray-200 bg-gray-50 p-4 text-sm text-gray-700 whitespace-pre-wrap">{body}</div>
    </div>
  );
}
