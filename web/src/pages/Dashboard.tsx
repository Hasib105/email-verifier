import { useEffect, useState, type ReactNode } from 'react';
import { Activity, Globe, ShieldCheck, TriangleAlert } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { HealthResponse, VerificationStats } from '../types';

export function Dashboard() {
  const { config } = useAuth();
  const [stats, setStats] = useState<VerificationStats | null>(null);
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const load = async () => {
      try {
        const [statsData, healthData] = await Promise.all([
          api.getVerificationStats(config),
          api.getHealth(config),
        ]);
        setStats(statsData);
        setHealth(healthData);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load dashboard');
      } finally {
        setLoading(false);
      }
    };

    void load();
  }, [config]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-yellow-500"></div>
      </div>
    );
  }

  const total = stats?.total || 0;
  const deliverable = stats?.by_classification?.deliverable || 0;
  const undeliverable = stats?.by_classification?.undeliverable || 0;
  const risky = (stats?.by_classification?.accept_all || 0) + (stats?.by_classification?.unknown || 0);

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-2xl sm:text-3xl font-bold text-gray-900 tracking-tight">Verifier V2</h2>
        <p className="text-sm text-gray-500 mt-1">Direct SMTP callout classifications, domain baselines, and evidence-driven risk scoring.</p>
      </div>

      {error && (
        <div className="p-4 bg-red-50 border border-red-200 text-red-700 rounded-xl text-sm">{error}</div>
      )}

      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4 sm:gap-6">
        <MetricCard title="Total Checks" value={total} icon={<Activity className="w-5 h-5 text-blue-600" />} tone="blue" />
        <MetricCard title="Deliverable" value={deliverable} icon={<ShieldCheck className="w-5 h-5 text-green-600" />} tone="green" />
        <MetricCard title="Undeliverable" value={undeliverable} icon={<TriangleAlert className="w-5 h-5 text-red-600" />} tone="red" />
        <MetricCard title="Risky / Ambiguous" value={risky} icon={<Globe className="w-5 h-5 text-amber-600" />} tone="amber" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white border rounded-xl shadow-sm p-6">
          <h3 className="text-lg font-bold mb-4">Classifier Health</h3>
          {health ? (
            <div className="space-y-3 text-sm text-gray-700">
              <div className="flex items-center justify-between rounded-lg bg-gray-50 px-4 py-3">
                <span>Mode</span>
                <span className="font-semibold text-gray-900">{health.mode}</span>
              </div>
              <div className="flex items-center justify-between rounded-lg bg-gray-50 px-4 py-3">
                <span>MAIL FROM</span>
                <span className="font-mono text-xs text-gray-900">{health.mail_from}</span>
              </div>
              <div className="flex items-center justify-between rounded-lg bg-gray-50 px-4 py-3">
                <span>EHLO Domain</span>
                <span className="font-mono text-xs text-gray-900">{health.ehlo_domain}</span>
              </div>
              <div className="flex items-center justify-between rounded-lg bg-gray-50 px-4 py-3">
                <span>Per-domain Baseline TTL</span>
                <span className="font-semibold text-gray-900">{health.baseline_ttl}</span>
              </div>
            </div>
          ) : (
            <p className="text-sm text-gray-500">Health information is unavailable.</p>
          )}
        </div>

        <div className="bg-white border rounded-xl shadow-sm p-6">
          <h3 className="text-lg font-bold mb-4">Classification Mix</h3>
          {stats?.by_classification && Object.keys(stats.by_classification).length > 0 ? (
            <div className="space-y-4">
              {Object.entries(stats.by_classification).map(([classification, count]) => (
                <div key={classification}>
                  <div className="flex items-center justify-between text-sm mb-1">
                    <span className="capitalize text-gray-700">{classification.replace(/_/g, ' ')}</span>
                    <span className="font-semibold text-gray-900">{count}</span>
                  </div>
                  <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-yellow-500 rounded-full"
                      style={{ width: `${total > 0 ? (count / total) * 100 : 0}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-500">No verification history yet.</p>
          )}
        </div>
      </div>
    </div>
  );
}

function MetricCard({
  title,
  value,
  icon,
  tone,
}: {
  title: string
  value: number
  icon: ReactNode
  tone: 'blue' | 'green' | 'red' | 'amber'
}) {
  const classes: Record<typeof tone, string> = {
    blue: 'from-blue-50 to-blue-100 border-blue-200',
    green: 'from-green-50 to-green-100 border-green-200',
    red: 'from-red-50 to-red-100 border-red-200',
    amber: 'from-amber-50 to-amber-100 border-amber-200',
  };

  return (
    <div className={`bg-gradient-to-br ${classes[tone]} p-4 sm:p-6 rounded-xl border shadow-sm`}>
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold tracking-wider text-gray-700 uppercase">{title}</h3>
        {icon}
      </div>
      <p className="text-3xl sm:text-4xl font-black text-gray-900">{value.toLocaleString()}</p>
    </div>
  );
}
