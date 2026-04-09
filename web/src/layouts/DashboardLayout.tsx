import { useState } from 'react';
import { Outlet, NavLink, useLocation, useNavigate } from 'react-router-dom';
import {
  Activity,
  Bell,
  ChevronDown,
  FileText,
  HelpCircle,
  LayoutDashboard,
  LogOut,
  MailCheck,
  Menu,
  PlayCircle,
  Search,
  Settings,
  ShieldAlert,
  X,
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';

export function DashboardLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { logout, user } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const navigation = [
    { name: 'Overview', href: '/dashboard', icon: LayoutDashboard },
    { name: 'Playground', href: '/dashboard/playground', icon: PlayCircle },
    { name: 'Email Config', href: '/dashboard/email-config', icon: MailCheck },
    { name: 'Templates', href: '/dashboard/templates', icon: FileText },
    { name: 'Email Status', href: '/dashboard/status', icon: Activity },
    ...(user?.is_superuser ? [{ name: 'Admin', href: '/dashboard/admin', icon: ShieldAlert }] : []),
  ];

  const displayName = user?.name?.trim() || user?.email?.split('@')[0] || 'Account';
  const initials = displayName
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join('');

  const closeSidebar = () => setSidebarOpen(false);

  const handleLogout = () => {
    logout();
    navigate('/auth/login');
  };

  const isActiveRoute = (href: string) => (
    href === '/dashboard' ? location.pathname === href : location.pathname.startsWith(href)
  );

  return (
    <div className="min-h-screen bg-[#f7f8fa] text-slate-950">
      {sidebarOpen && (
        <button
          aria-label="Close navigation overlay"
          className="fixed inset-0 z-40 bg-slate-950/30 lg:hidden"
          onClick={closeSidebar}
          type="button"
        />
      )}

      <div className="flex min-h-screen">
        <aside
          className={`fixed inset-y-0 left-0 z-50 flex w-[216px] shrink-0 flex-col border-r border-slate-200 bg-white transition-transform duration-200 lg:static lg:translate-x-0 ${
            sidebarOpen ? 'translate-x-0' : '-translate-x-full'
          }`}
        >
          <div className="flex h-16 items-center justify-between border-b border-slate-200 px-4">
            <NavLink className="flex items-center gap-3" onClick={closeSidebar} to="/dashboard">
              <span className="flex h-8 w-8 items-center justify-center bg-slate-950 text-white">
                <LayoutDashboard className="h-4 w-4" />
              </span>
              <span className="text-base font-bold tracking-tight">Verifier</span>
            </NavLink>
            <button
              aria-label="Close navigation"
              className="rounded-md p-2 text-slate-500 hover:bg-slate-100 lg:hidden"
              onClick={closeSidebar}
              type="button"
            >
              <X className="h-5 w-5" />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto px-3 py-5">
            <p className="px-3 text-xs font-semibold uppercase tracking-widest text-slate-500">Workspace</p>
            <nav className="mt-3 space-y-1">
              {navigation.map((item) => {
                const active = isActiveRoute(item.href);
                return (
                  <NavLink
                    className={`flex items-center gap-3 px-3 py-2.5 text-sm font-medium transition-colors ${
                      active
                        ? 'bg-slate-950 text-white'
                        : 'text-slate-600 hover:bg-slate-100 hover:text-slate-950'
                    }`}
                    key={item.name}
                    onClick={closeSidebar}
                    to={item.href}
                  >
                    <item.icon className="h-4 w-4" />
                    <span>{item.name}</span>
                  </NavLink>
                );
              })}
            </nav>
          </div>

          <div className="border-t border-slate-200 p-3">
            <NavLink
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 text-sm font-medium transition-colors ${
                  isActive ? 'bg-slate-950 text-white' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-950'
                }`
              }
              onClick={closeSidebar}
              to="/dashboard/settings"
            >
              <Settings className="h-4 w-4" />
              <span>Settings</span>
            </NavLink>
            <button
              className="mt-1 flex w-full items-center gap-3 px-3 py-2.5 text-left text-sm font-medium text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-950"
              type="button"
            >
              <HelpCircle className="h-4 w-4" />
              <span>Help & Support</span>
            </button>
            <button
              className="mt-1 flex w-full items-center gap-3 px-3 py-2.5 text-left text-sm font-medium text-red-600 transition-colors hover:bg-red-50"
              onClick={handleLogout}
              type="button"
            >
              <LogOut className="h-4 w-4" />
              <span>Log out</span>
            </button>
          </div>
        </aside>

        <main className="flex min-w-0 flex-1 flex-col">
          <header className="sticky top-0 z-30 flex h-12 items-center justify-between border-b border-slate-200 bg-white px-4 lg:px-6">
            <div className="flex min-w-0 flex-1 items-center gap-3">
              <button
                aria-label="Open navigation"
                className="rounded-md p-2 text-slate-600 hover:bg-slate-100 lg:hidden"
                onClick={() => setSidebarOpen(true)}
                type="button"
              >
                <Menu className="h-5 w-5" />
              </button>
              <label className="relative hidden w-full max-w-sm sm:block">
                <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                <input
                  className="h-9 w-full rounded-none border-0 bg-slate-100 pl-9 pr-3 text-sm text-slate-900 outline-none transition-colors placeholder:text-slate-500 focus:bg-slate-50 focus:ring-1 focus:ring-slate-300"
                  placeholder="Search emails, statuses..."
                  type="search"
                />
              </label>
            </div>

            <div className="flex items-center gap-3">
              <button
                aria-label="Notifications"
                className="relative rounded-md p-2 text-slate-600 hover:bg-slate-100"
                type="button"
              >
                <Bell className="h-4 w-4" />
                <span className="absolute right-1.5 top-1.5 h-1.5 w-1.5 rounded-full bg-red-500" />
              </button>
              <button
                className="flex items-center gap-2 rounded-md px-1.5 py-1 text-left hover:bg-slate-100"
                type="button"
              >
                <span className="flex h-8 w-8 items-center justify-center rounded-full bg-emerald-100 text-xs font-bold text-emerald-900">
                  {initials || 'A'}
                </span>
                <span className="hidden leading-tight sm:block">
                  <span className="block max-w-[140px] truncate text-sm font-semibold text-slate-950">{displayName}</span>
                  <span className="block text-xs text-slate-500">{user?.is_superuser ? 'Admin' : 'Member'}</span>
                </span>
                <ChevronDown className="hidden h-4 w-4 text-slate-400 sm:block" />
              </button>
            </div>
          </header>

          <div className="flex-1 overflow-auto px-4 py-6 sm:px-6 lg:px-8">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
