import { useState } from 'react';
import { Activity, Loader2, Mail, ShieldCheck, TriangleAlert } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { VerifyResponse } from '../types';

export function Playground() {
  const { config } = useAuth();
  const [email, setEmail] = useState('');
  const [result, setResult] = useState<VerifyResponse | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email) return;

    setLoading(true);
    setError('');
    setResult(null);
    try {
      const response = await api.createVerification(config, email);
      setResult(response);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Verification failed');
    } finally {
      setLoading(false);
    }
  };

  const accent = (classification: string) => {
    switch (classification) {
      case 'deliverable':
        return 'bg-green-100 text-green-800 border-green-200';
      case 'undeliverable':
        return 'bg-red-100 text-red-800 border-red-200';
      case 'accept_all':
        return 'bg-amber-100 text-amber-800 border-amber-200';
      default:
        return 'bg-gray-100 text-gray-800 border-gray-200';
    }
  };

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-2xl sm:text-3xl font-bold text-gray-900 tracking-tight">Verification Playground</h2>
        <p className="text-sm text-gray-500 mt-1">Run the deterministic SMTP classifier and inspect the first-pass confidence before enrichment completes.</p>
      </div>

      <div className="bg-white border rounded-xl shadow-sm p-6">
        <form onSubmit={handleVerify} className="flex flex-col md:flex-row gap-4">
          <div className="flex-1 relative">
            <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-lg border px-10 py-3 text-sm"
              placeholder="name@company.com"
              required
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="inline-flex items-center justify-center gap-2 rounded-lg bg-black px-5 py-3 text-sm font-medium text-white hover:bg-gray-800 disabled:opacity-50"
          >
            {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Activity className="w-4 h-4" />}
            {loading ? 'Verifying...' : 'Verify'}
          </button>
        </form>
      </div>

      {error && (
        <div className="rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</div>
      )}

      {result && (
        <div className="bg-white border rounded-xl shadow-sm overflow-hidden">
          <div className="flex items-center justify-between gap-3 border-b bg-gray-50 px-6 py-4">
            <div>
              <p className="text-xs uppercase tracking-wider text-gray-500">Classification</p>
              <h3 className="text-xl font-bold text-gray-900">{result.email}</h3>
            </div>
            <span className={`rounded-full border px-3 py-1 text-xs font-semibold uppercase tracking-wider ${accent(result.classification)}`}>
              {result.classification.replace(/_/g, ' ')}
            </span>
          </div>
          <div className="space-y-6 px-6 py-6">
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <Stat label="Confidence" value={`${result.confidence_score}/100`} />
              <Stat label="Risk" value={result.risk_level} />
              <Stat label="State" value={result.state} />
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <Section title="Protocol Summary" body={result.protocol_summary} />
              <Section title="Reason Codes" body={result.reason_codes.join(', ') || 'None'} />
            </div>

            <Section title="Enrichment Summary" body={result.enrichment_summary || 'Pending or not needed.'} />

            <div className="rounded-xl border border-gray-200 bg-gray-50 p-4 text-sm text-gray-600">
              {result.classification === 'deliverable' ? (
                <div className="flex items-start gap-3">
                  <ShieldCheck className="w-5 h-5 text-green-600 mt-0.5" />
                  <p>The target recipient was accepted while the domain baseline rejected an obviously fake control address.</p>
                </div>
              ) : (
                <div className="flex items-start gap-3">
                  <TriangleAlert className="w-5 h-5 text-amber-600 mt-0.5" />
                  <p>Ambiguous results are preserved as evidence-backed risk, rather than forced into a binary valid/invalid answer.</p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-gray-200 bg-gray-50 p-4">
      <p className="text-xs uppercase tracking-wider text-gray-500">{label}</p>
      <p className="mt-2 text-lg font-semibold text-gray-900 capitalize">{value}</p>
    </div>
  );
}

function Section({ title, body }: { title: string; body: string }) {
  return (
    <div className="rounded-xl border border-gray-200 p-4">
      <p className="text-xs uppercase tracking-wider text-gray-500">{title}</p>
      <p className="mt-2 text-sm text-gray-700 whitespace-pre-wrap">{body}</p>
    </div>
  );
}
