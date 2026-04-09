import { Outlet } from 'react-router-dom';

export function AuthLayout() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-[#f7f8fa] px-4 py-10">
      <div className="grid w-full max-w-5xl overflow-hidden border border-slate-200 bg-white shadow-sm lg:grid-cols-[1fr_440px]">
        <section className="hidden border-r border-slate-200 bg-slate-950 p-10 text-white lg:flex lg:flex-col lg:justify-between">
          <div>
            <div className="flex items-center gap-3">
              <span className="flex h-9 w-9 items-center justify-center rounded-sm bg-white text-sm font-black text-slate-950">
                EV
              </span>
              <span className="text-lg font-bold">Verifier</span>
            </div>
            <div className="mt-16 max-w-md">
              <p className="text-xs font-semibold uppercase tracking-widest text-slate-400">Email verification</p>
              <h1 className="mt-4 text-4xl font-bold leading-tight tracking-tight">
                Keep your sender reputation clean.
              </h1>
              <p className="mt-4 text-sm leading-6 text-slate-300">
                Verify inboxes, monitor bounce signals, and manage webhook delivery from one focused console.
              </p>
            </div>
          </div>
          <div className="grid grid-cols-3 gap-3 text-sm">
            <div className="border border-white/15 p-4">
              <p className="text-2xl font-bold">SMTP</p>
              <p className="mt-1 text-slate-400">Probe checks</p>
            </div>
            <div className="border border-white/15 p-4">
              <p className="text-2xl font-bold">API</p>
              <p className="mt-1 text-slate-400">Key access</p>
            </div>
            <div className="border border-white/15 p-4">
              <p className="text-2xl font-bold">Hooks</p>
              <p className="mt-1 text-slate-400">Live events</p>
            </div>
          </div>
        </section>
        <section className="p-6 sm:p-10">
          <Outlet />
        </section>
      </div>
    </div>
  );
}
