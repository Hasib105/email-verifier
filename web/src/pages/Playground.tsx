import { useEffect, useState } from 'react';
import { AlertCircle, CheckCircle2, Clock, Loader2, Mail, Server } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { VerificationRecord, VerifyResponse } from '../types';

const statusBadge = (status: string) => {
  const map: Record<string, string> = {
    valid: 'bg-emerald-50 text-emerald-700',
    invalid: 'bg-red-50 text-red-700',
    bounced: 'bg-red-50 text-red-700',
    pending_bounce_check: 'bg-amber-50 text-amber-700',
    greylisted: 'bg-orange-50 text-orange-700',
    error: 'bg-slate-100 text-slate-700',
  };
  return map[status] || 'bg-slate-100 text-slate-700';
};

const resultFromRecord = (record: VerificationRecord, previous: VerifyResponse): VerifyResponse => ({
  id: record.id,
  email: record.email,
  status: record.status,
  message: record.message,
  source: record.source,
  cached: previous.cached,
  finalized: record.finalized,
  next_check_at: record.next_check_at || undefined,
  confidence: record.confidence,
  deterministic: record.deterministic,
  reason_code: record.reason_code,
  verification_path: record.verification_path,
  signal_summary: record.signal_summary,
  expires_at: record.expires_at,
});

export function Playground() {
  const { config } = useAuth();
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<VerifyResponse | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!result?.id || result.finalized) {
      return undefined;
    }

    const refreshTimer = window.setInterval(async () => {
      try {
        const record = await api.getVerification(config, result.id);
        setResult((current) => (current ? resultFromRecord(record, current) : current));
      } catch {
        // Keep the current result visible if a background refresh fails.
      }
    }, 15000);

    return () => window.clearInterval(refreshTimer);
  }, [config, result?.finalized, result?.id]);

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email) return;

    setLoading(true);
    setError('');
    setResult(null);

    try {
      const response = await api.verifyEmail(config, email);
      setResult(response);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred during verification.');
    } finally {
      setLoading(false);
    }
  };

  const StatusIcon = () => {
    switch (result?.status) {
      case 'valid':
        return <CheckCircle2 className="h-5 w-5 text-emerald-600" />;
      case 'invalid':
      case 'bounced':
      case 'error':
        return <AlertCircle className="h-5 w-5 text-red-600" />;
      case 'pending_bounce_check':
      case 'greylisted':
        return <Clock className="h-5 w-5 text-amber-600" />;
      default:
        return <Server className="h-5 w-5 text-slate-500" />;
    }
  };

  return (
    <div className="mx-auto max-w-[1120px] space-y-6">
      <div>
        <p className="text-xs font-semibold uppercase tracking-widest text-slate-500">Verification tool</p>
        <h1 className="mt-2 text-2xl font-bold tracking-tight text-slate-950 sm:text-3xl">Playground</h1>
        <p className="mt-1 text-sm text-slate-500">Run a single email check and inspect the returned signal detail. Pending results refresh every 15 seconds.</p>
      </div>

      <section className="border border-slate-200 bg-white p-5">
        <form className="flex flex-col gap-3 sm:flex-row" onSubmit={handleVerify}>
          <label className="relative flex-1">
            <Mail className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
            <input
              className="h-11 w-full border border-slate-300 pl-10 pr-3 text-sm text-slate-950 outline-none transition-colors placeholder:text-slate-400 focus:border-slate-950 focus:ring-1 focus:ring-slate-950"
              disabled={loading}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="test@example.com"
              required
              type="email"
              value={email}
            />
          </label>
          <button
            className="inline-flex h-11 items-center justify-center gap-2 bg-slate-950 px-5 text-sm font-semibold text-white transition-colors hover:bg-slate-800 disabled:opacity-50"
            disabled={loading || !email}
            type="submit"
          >
            {loading && <Loader2 className="h-4 w-4 animate-spin" />}
            {loading ? 'Verifying...' : 'Verify Email'}
          </button>
        </form>
      </section>

      {error && (
        <div className="border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      )}

      {result && (
        <section className="border border-slate-200 bg-white">
          <div className="flex flex-col gap-3 border-b border-slate-200 bg-slate-50 px-5 py-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-3">
              <span className="flex h-9 w-9 items-center justify-center border border-slate-200 bg-white">
                <StatusIcon />
              </span>
              <div>
                <h2 className="text-base font-bold text-slate-950">Verification Result</h2>
                <p className="text-sm text-slate-500">{result.email}</p>
              </div>
            </div>
            <span className={`inline-flex w-fit px-2.5 py-1 text-xs font-semibold uppercase tracking-wider ${statusBadge(result.status)}`}>
              {result.status.replace(/_/g, ' ')}
            </span>
          </div>

          <div className="grid gap-0 lg:grid-cols-[1fr_340px]">
            <div className="border-b border-slate-200 p-5 lg:border-b-0 lg:border-r">
              <p className="text-xs font-semibold uppercase tracking-widest text-slate-500">Message Detail</p>
              <p className="mt-3 text-sm leading-6 text-slate-700">{result.message}</p>
              {!result.finalized && result.next_check_at && (
                <p className="mt-3 text-sm text-amber-700">
                  Next backend bounce check: {new Date(result.next_check_at * 1000).toLocaleString()}
                </p>
              )}
              {result.signal_summary && (
                <div className="mt-5 border border-slate-200 bg-slate-50 p-4">
                  <p className="text-xs font-semibold uppercase tracking-widest text-slate-500">Signal Summary</p>
                  <p className="mt-2 text-sm leading-6 text-slate-700">{result.signal_summary}</p>
                </div>
              )}
            </div>

            <dl className="grid grid-cols-2 border-slate-200 text-sm lg:grid-cols-1">
              {[
                ['Source', result.source],
                ['Confidence', result.confidence],
                ['Cached', result.cached ? 'Yes' : 'No'],
                ['Finalized', result.finalized ? 'Yes' : 'No'],
                ['Deterministic', result.deterministic ? 'Yes' : 'No'],
                ['Path', result.verification_path.replace(/_/g, ' ')],
              ].map(([label, value]) => (
                <div className="border-b border-r border-slate-200 p-4 last:border-b-0 lg:border-r-0" key={label}>
                  <dt className="text-xs font-semibold uppercase tracking-widest text-slate-500">{label}</dt>
                  <dd className="mt-1 capitalize text-slate-950">{value}</dd>
                </div>
              ))}
            </dl>
          </div>
        </section>
      )}
    </div>
  );
}
