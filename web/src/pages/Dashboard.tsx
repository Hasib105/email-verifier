import { useState, useEffect } from 'react';
import {
  CalendarDays,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Filter,
  LayoutList,
  Search,
  Table2,
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { HealthResponse, VerificationStats } from '../types';

type LeadRow = {
  name: string;
  email: string;
  phone: string;
  purpose: string;
  amount: number;
  progress: number;
  stage: 'New' | 'In progress' | 'Loan Granted';
  avatarTint: string;
  owners: string[];
};

const leadRows: LeadRow[] = [
  {
    name: 'Jenny Wilson',
    email: 'jenny.wilson@gmail.com',
    phone: '(603) 555-0123',
    purpose: 'Home Loan',
    amount: 978878,
    progress: 70,
    stage: 'New',
    avatarTint: 'bg-gradient-to-br from-[#b36f4c] to-[#f6d3b8]',
    owners: ['JW', 'AL'],
  },
  {
    name: 'Eleanor Pena',
    email: 'eleanor.pena@gmail.com',
    phone: '(208) 555-0112',
    purpose: 'Gold Loan',
    amount: 9878,
    progress: 20,
    stage: 'In progress',
    avatarTint: 'bg-gradient-to-br from-[#2f3645] to-[#c4cad3]',
    owners: ['EP', 'SF', 'RN'],
  },
  {
    name: 'Jane Cooper',
    email: 'jane.cooper@gmail.com',
    phone: '(205) 555-0100',
    purpose: 'Business Loan',
    amount: 43532,
    progress: 45,
    stage: 'Loan Granted',
    avatarTint: 'bg-gradient-to-br from-[#0f172a] to-[#444f63]',
    owners: ['JC', 'LR'],
  },
  {
    name: 'Imalia Jones',
    email: 'imalia.jones@gmail.com',
    phone: '(201) 555-0124',
    purpose: 'Property Loan',
    amount: 978878,
    progress: 96,
    stage: 'In progress',
    avatarTint: 'bg-gradient-to-br from-[#93a2ae] to-[#f5c8b0]',
    owners: ['IJ'],
  },
  {
    name: 'Linda Miles',
    email: 'linda.miles@gmail.com',
    phone: '(307) 555-0133',
    purpose: 'Education Loan',
    amount: 9878,
    progress: 50,
    stage: 'New',
    avatarTint: 'bg-gradient-to-br from-[#a05f32] to-[#ffcc9f]',
    owners: ['LM', 'TR'],
  },
  {
    name: 'Bella Sanders',
    email: 'bella.sanders@gmail.com',
    phone: '(907) 555-0101',
    purpose: 'Gold Loan',
    amount: 13324,
    progress: 42,
    stage: 'Loan Granted',
    avatarTint: 'bg-gradient-to-br from-[#6f4f39] to-[#f5ddb2]',
    owners: ['BS'],
  },
  {
    name: 'Jacob Jones',
    email: 'jacob.jones@gmail.com',
    phone: '(907) 555-0101',
    purpose: 'Home Loan',
    amount: 13324,
    progress: 56,
    stage: 'New',
    avatarTint: 'bg-gradient-to-br from-[#5b726d] to-[#cfdfd8]',
    owners: ['JJ', 'DM'],
  },
];

const stageTone: Record<LeadRow['stage'], string> = {
  New: 'bg-[#f0e5ff] text-[#7c3aed]',
  'In progress': 'bg-[#dcfce7] text-[#1f7a43]',
  'Loan Granted': 'bg-[#fef3c7] text-[#a16207]',
};

const ownerTone = [
  'bg-[#e2e8f0] text-[#334155]',
  'bg-[#fee2e2] text-[#9f1239]',
  'bg-[#ede9fe] text-[#5b21b6]',
  'bg-[#dcfce7] text-[#166534]',
];

const currency = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
  maximumFractionDigits: 0,
});

export function Dashboard() {
  const { config } = useAuth();
  const [stats, setStats] = useState<VerificationStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [health, setHealth] = useState<HealthResponse | null>(null);

  useEffect(() => {
    const loadData = async () => {
      try {
        const [statsData, healthData] = await Promise.all([
          api.getVerificationStats(config),
          api.getHealth(config).catch(() => null),
        ]);
        setStats(statsData);
        setHealth(healthData);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load data');
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, [config]);

  const validCount = stats?.by_status?.valid || 0;
  const invalidCount = (stats?.by_status?.invalid || 0) + (stats?.by_status?.bounced || 0);
  const pendingCount = (stats?.by_status?.pending_bounce_check || 0) + (stats?.by_status?.greylisted || 0);
  const totalCount = stats?.total || 0;
  const directSmtpStatus = health?.direct_smtp_status || 'unknown';
  const healthLabel =
    directSmtpStatus === 'available'
      ? 'SMTP Available'
      : directSmtpStatus === 'degraded'
        ? 'SMTP Degraded'
        : 'SMTP Unknown';
  const healthPillTone =
    directSmtpStatus === 'available'
      ? 'bg-[#dcfce7] text-[#166534]'
      : directSmtpStatus === 'degraded'
        ? 'bg-[#fee2e2] text-[#b91c1c]'
        : 'bg-[#fef3c7] text-[#92400e]';

  return (
    <div className="space-y-4">
      <section className="animate-fade-up rounded-[24px] border border-[#d8dde6] bg-white/95 p-4 shadow-[0_12px_35px_rgba(15,23,42,0.08)] sm:p-5 lg:p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
          <div>
            <h1 className="text-2xl font-extrabold tracking-tight text-[#111827] sm:text-3xl">Leads</h1>
            <p className="mt-1 text-sm text-[#667085]">Pipeline view inspired by your shared CRM layout.</p>
          </div>

          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <div className="rounded-xl border border-[#dde3ec] bg-[#f8fafd] px-3 py-2">
              <p className="text-[11px] font-bold uppercase tracking-[0.16em] text-[#8a94a6]">Total</p>
              <p className="mt-1 text-base font-extrabold text-[#0f172a]">{loading ? '...' : totalCount.toLocaleString()}</p>
            </div>
            <div className="rounded-xl border border-[#dde3ec] bg-[#f8fafd] px-3 py-2">
              <p className="text-[11px] font-bold uppercase tracking-[0.16em] text-[#8a94a6]">Valid</p>
              <p className="mt-1 text-base font-extrabold text-[#0f172a]">{loading ? '...' : validCount.toLocaleString()}</p>
            </div>
            <div className="rounded-xl border border-[#dde3ec] bg-[#f8fafd] px-3 py-2">
              <p className="text-[11px] font-bold uppercase tracking-[0.16em] text-[#8a94a6]">Invalid</p>
              <p className="mt-1 text-base font-extrabold text-[#0f172a]">{loading ? '...' : invalidCount.toLocaleString()}</p>
            </div>
            <div className="rounded-xl border border-[#dde3ec] bg-[#f8fafd] px-3 py-2">
              <p className="text-[11px] font-bold uppercase tracking-[0.16em] text-[#8a94a6]">SMTP</p>
              <p className={`mt-1 inline-flex rounded-md px-2 py-0.5 text-xs font-bold ${healthPillTone}`}>{healthLabel}</p>
            </div>
          </div>
        </div>

        {error && (
          <div className="mt-4 rounded-xl border border-[#fecaca] bg-[#fef2f2] p-3 text-sm font-medium text-[#991b1b]">
            {error}
          </div>
        )}

        <div className="mt-5 overflow-hidden rounded-[18px] border border-[#dde3ec]">
          <div className="flex flex-col gap-3 border-b border-[#e5eaf1] bg-[#f7f9fc] p-4 md:flex-row md:items-center md:justify-between">
            <div className="relative w-full md:max-w-sm">
              <Search className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-[#94a3b8]" />
              <input
                type="text"
                placeholder="Search..."
                className="h-10 w-full rounded-xl border border-[#d8dde6] bg-white pl-10 pr-3 text-sm text-[#111827] outline-none transition focus:border-[#bac6d8]"
              />
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <button className="inline-flex h-10 items-center gap-2 rounded-xl border border-[#d8dde6] bg-white px-3 text-sm font-semibold text-[#475467] transition hover:text-[#111827]">
                <Filter className="h-4 w-4" />
                <span>1</span>
              </button>
              <button className="grid h-10 w-10 place-items-center rounded-xl border border-[#d8dde6] bg-white text-[#64748b] transition hover:text-[#111827]">
                <LayoutList className="h-4 w-4" />
              </button>
              <button className="grid h-10 w-10 place-items-center rounded-xl border border-[#d8dde6] bg-white text-[#64748b] transition hover:text-[#111827]">
                <Table2 className="h-4 w-4" />
              </button>
              <button className="grid h-10 w-10 place-items-center rounded-xl border border-[#d8dde6] bg-white text-[#64748b] transition hover:text-[#111827]">
                <CalendarDays className="h-4 w-4" />
              </button>
              <button className="inline-flex h-10 items-center gap-2 rounded-xl border border-[#d8dde6] bg-white px-3 text-sm font-semibold text-[#111827]">
                Options
                <ChevronDown className="h-4 w-4" />
              </button>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="min-w-[980px] w-full border-collapse text-left">
              <thead className="bg-[#f4f7fb]">
                <tr className="text-[11px] uppercase tracking-[0.16em] text-[#8b97ab]">
                  <th className="w-10 px-4 py-3">
                    <input type="checkbox" className="h-4 w-4 rounded border-[#c8d1df] text-[#111827]" />
                  </th>
                  <th className="px-4 py-3 font-bold">Name</th>
                  <th className="px-4 py-3 font-bold">Contacts</th>
                  <th className="px-4 py-3 font-bold">Purpose</th>
                  <th className="px-4 py-3 font-bold">Amount</th>
                  <th className="px-4 py-3 font-bold">Lead Owner</th>
                  <th className="px-4 py-3 font-bold">Progress</th>
                  <th className="px-4 py-3 font-bold">Stages</th>
                </tr>
              </thead>
              <tbody>
                {leadRows.map((lead, index) => (
                  <tr
                    key={lead.email}
                    className="animate-rise border-t border-[#e6ebf2] bg-white text-sm text-[#1f2937] transition-colors hover:bg-[#fafcff]"
                    style={{ animationDelay: `${index * 70}ms` }}
                  >
                    <td className="px-4 py-4">
                      <input type="checkbox" className="h-4 w-4 rounded border-[#c8d1df] text-[#111827]" />
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-3">
                        <div className={`grid h-10 w-10 place-items-center rounded-full text-xs font-extrabold text-white ${lead.avatarTint}`}>
                          {lead.name.split(' ').map((value) => value[0]).join('').slice(0, 2)}
                        </div>
                        <p className="font-bold text-[#0f172a]">{lead.name}</p>
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      <p className="font-semibold text-[#1e293b]">{lead.email}</p>
                      <p className="mt-0.5 text-xs text-[#8091a7]">{lead.phone}</p>
                    </td>
                    <td className="px-4 py-4 font-semibold text-[#334155]">{lead.purpose}</td>
                    <td className="px-4 py-4 text-base font-extrabold text-[#0f172a]">{currency.format(lead.amount)}</td>
                    <td className="px-4 py-4">
                      <div className="flex -space-x-2">
                        {lead.owners.map((owner, ownerIndex) => (
                          <div
                            key={`${lead.email}-${owner}`}
                            className={`grid h-8 w-8 place-items-center rounded-full border-2 border-white text-[10px] font-extrabold ${ownerTone[(index + ownerIndex) % ownerTone.length]}`}
                          >
                            {owner}
                          </div>
                        ))}
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-3">
                        <div className="h-2.5 w-28 overflow-hidden rounded-full bg-[#e9edf3]">
                          <div className="h-full rounded-full bg-[#5f6d82]" style={{ width: `${lead.progress}%` }} />
                        </div>
                        <span className="text-xs font-bold text-[#6b7280]">{lead.progress}%</span>
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      <span className={`inline-flex rounded-md px-3 py-1 text-xs font-bold ${stageTone[lead.stage]}`}>{lead.stage}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3 border-t border-[#e6ebf2] bg-[#f9fbfd] px-4 py-4">
            <button className="inline-flex h-9 items-center gap-2 rounded-lg border border-[#d6dce6] bg-white px-3 text-sm font-semibold text-[#374151] transition hover:text-[#111827]">
              <ChevronLeft className="h-4 w-4" />
              Previous
            </button>

            <div className="flex items-center gap-1 text-sm font-semibold text-[#6b7280]">
              <button className="grid h-9 w-9 place-items-center rounded-md">1</button>
              <button className="grid h-9 w-9 place-items-center rounded-md bg-[#0f172a] text-white">2</button>
              <span className="px-1">...</span>
              <button className="grid h-9 w-9 place-items-center rounded-md">8</button>
              <button className="grid h-9 w-9 place-items-center rounded-md">9</button>
            </div>

            <button className="inline-flex h-9 items-center gap-2 rounded-lg border border-[#d6dce6] bg-white px-3 text-sm font-semibold text-[#374151] transition hover:text-[#111827]">
              Next
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      </section>

      {pendingCount > 0 && (
        <p className="px-1 text-xs font-semibold uppercase tracking-[0.2em] text-[#8b97ab]">
          Pending queue: {pendingCount.toLocaleString()} verifications
        </p>
      )}
    </div>
  );
}
