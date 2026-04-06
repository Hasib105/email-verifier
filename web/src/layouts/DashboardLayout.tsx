import { useState } from 'react';
import { Outlet, NavLink, useLocation, useNavigate } from 'react-router-dom';
import {
  Bell,
  BriefcaseBusiness,
  ChevronDown,
  CircleHelp,
  ContactRound,
  LayoutGrid,
  LogOut,
  Menu,
  Search,
  Settings,
  Shapes,
  ShieldAlert,
  User,
  Users,
  Workflow,
  X,
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';

export function DashboardLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { logout, user } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const primaryNavigation = [
    { name: 'Overview', href: '/dashboard/status', icon: LayoutGrid },
  ];

  const contactsNavigation = [
    { name: 'Leads', href: '/dashboard', icon: Users },
    { name: 'Referral Partners', href: '/dashboard/templates', icon: ContactRound },
  ];

  const workflowNavigation = [
    { name: 'Deals', href: '/dashboard/playground', icon: BriefcaseBusiness },
    { name: 'Integration', href: '/dashboard/email-config', icon: Workflow },
    { name: 'Tasks', href: '/dashboard/settings', icon: Shapes },
    ...(user?.is_superuser ? [{ name: 'Admin', href: '/dashboard/admin', icon: ShieldAlert }] : []),
  ];

  const handleLogout = () => {
    logout();
    navigate('/auth/login');
  };

  const displayName = user?.email?.split('@')[0] ?? 'User';
  const closeSidebar = () => setSidebarOpen(false);

  const navItemClass = ({ isActive }: { isActive: boolean }) =>
    `group flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-semibold transition-colors ${
      isActive
        ? 'bg-[#e7ebf0] text-[#111827]'
        : 'text-[#5f6b7c] hover:bg-[#eef2f6] hover:text-[#111827]'
    }`;

  return (
    <div className="flex min-h-screen bg-[#edf1f5] text-[#111827]">
      {/* Mobile overlay */}
      {sidebarOpen && (
        <div 
          className="fixed inset-0 z-40 bg-[#0f172a]/50 lg:hidden"
          onClick={closeSidebar}
        />
      )}

      {/* Sidebar */}
      <aside className={`
        fixed inset-y-0 left-0 z-50 w-[278px]
        border-r border-[#d8dde6] bg-[#f8fafc]
        transform transition-transform duration-300 ease-out
        lg:static lg:translate-x-0
        ${sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
      `}>
        <div className="flex h-full flex-col justify-between">
          <div>
            <div className="flex items-center justify-between px-5 pb-4 pt-6">
              <div className="flex items-center gap-3">
                <div className="grid h-9 w-9 place-items-center rounded-md bg-[#0f172a]">
                  <span className="h-0 w-0 border-l-[5px] border-l-transparent border-r-[5px] border-r-transparent border-b-[10px] border-b-white" />
                </div>
                <p className="text-2xl font-extrabold tracking-tight text-[#0f172a]">Plan</p>
              </div>
              <button 
                onClick={closeSidebar}
                className="rounded-lg p-2 text-[#64748b] hover:bg-[#edf1f4] hover:text-[#0f172a] lg:hidden"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="space-y-3 px-4">
              <nav className="space-y-1">
                {primaryNavigation.map((item) => (
                  <NavLink key={item.name} to={item.href} onClick={closeSidebar} className={navItemClass}>
                    <item.icon className="h-4 w-4" />
                    <span>{item.name}</span>
                  </NavLink>
                ))}
              </nav>

              <section className="rounded-2xl bg-[#eef2f6] p-2">
                <div className="mb-1 flex items-center justify-between px-2 py-1">
                  <p className="text-xs font-bold uppercase tracking-[0.22em] text-[#8490a3]">Contacts</p>
                  <ChevronDown className="h-3.5 w-3.5 text-[#94a3b8]" />
                </div>
                <nav className="space-y-1">
                  {contactsNavigation.map((item) => (
                    <NavLink key={item.name} to={item.href} onClick={closeSidebar} className={navItemClass}>
                      <item.icon className="h-4 w-4" />
                      <span>{item.name}</span>
                    </NavLink>
                  ))}
                </nav>
              </section>

              <nav className="space-y-1 pt-1">
                {workflowNavigation.map((item) => (
                  <NavLink key={item.name} to={item.href} onClick={closeSidebar} className={navItemClass}>
                    <item.icon className="h-4 w-4" />
                    <span>{item.name}</span>
                  </NavLink>
                ))}
              </nav>
            </div>
          </div>

          <div className="border-t border-[#d8dde6] p-4">
            <div className="mb-3 flex items-center gap-3 rounded-xl bg-white px-3 py-2">
              <div className="grid h-9 w-9 place-items-center rounded-full bg-[#dbe3ee] text-sm font-bold text-[#0f172a]">
                {displayName.slice(0, 2).toUpperCase()}
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold text-[#0f172a]">{displayName}</p>
                <p className="truncate text-xs text-[#64748b]">{user?.email ?? 'Signed in'}</p>
              </div>
            </div>

            <NavLink
              to="/dashboard/settings"
              onClick={closeSidebar}
              className="mb-1 flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-semibold text-[#5f6b7c] transition-colors hover:bg-[#edf2f8] hover:text-[#111827]"
            >
              <Settings className="h-4 w-4" />
              <span>Settings</span>
            </NavLink>
            <NavLink
              to="/dashboard/email-config"
              onClick={closeSidebar}
              className="mb-2 flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-semibold text-[#5f6b7c] transition-colors hover:bg-[#edf2f8] hover:text-[#111827]"
            >
              <CircleHelp className="h-4 w-4" />
              <span>Help &amp; Support</span>
            </NavLink>
            <button 
              onClick={handleLogout}
              className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-semibold text-[#b42318] transition-colors hover:bg-[#fee4e2]"
            >
              <LogOut className="h-4 w-4"/>
              <span>Log out</span>
            </button>
            {user?.is_superuser && (
              <p className="mt-2 text-[11px] font-semibold uppercase tracking-[0.16em] text-[#94a3b8]">
                Superuser mode
              </p>
            )}
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="relative flex-1 overflow-hidden">
        <div className="pointer-events-none absolute inset-0">
          <div className="absolute -right-10 -top-20 h-72 w-72 rounded-full bg-white/80 blur-3xl" />
          <div className="absolute bottom-0 left-1/4 h-56 w-56 rounded-full bg-[#dde5ef]/80 blur-3xl" />
        </div>

        <header className="relative z-20 border-b border-[#d8dde6] bg-[#f4f7fb]/95 backdrop-blur">
          <div className="flex h-20 items-center gap-3 px-4 sm:px-6 lg:px-8">
            <button 
              onClick={() => setSidebarOpen(true)}
              className="rounded-lg p-2 text-[#4b5563] transition-colors hover:bg-white lg:hidden"
            >
              <Menu className="h-6 w-6" />
            </button>

            <div className="relative w-full max-w-xl">
              <Search className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-[#94a3b8]" />
              <input
                type="text"
                placeholder="Search or type a command"
                className="h-11 w-full rounded-xl border border-[#d8dde6] bg-white pl-11 pr-4 text-sm text-[#111827] outline-none ring-0 transition focus:border-[#bcc6d3]"
              />
            </div>

            <div className="ml-auto flex items-center gap-2">
              <button
                className="grid h-10 w-10 place-items-center rounded-xl border border-[#d8dde6] bg-white text-[#64748b] transition hover:text-[#111827]"
                aria-label="Notifications"
              >
                <Bell className="h-4 w-4" />
              </button>
              <button
                className="hidden h-10 items-center gap-2 rounded-xl border border-[#d8dde6] bg-white px-3 text-sm font-semibold text-[#334155] transition hover:text-[#0f172a] sm:flex"
                aria-label="Profile"
              >
                <User className="h-4 w-4" />
                <span>{displayName}</span>
              </button>
            </div>
          </div>
        </header>

        <section className="relative z-10 h-[calc(100vh-5rem)] overflow-y-auto px-3 pb-6 pt-4 sm:px-6 sm:pt-6 lg:px-8">
          {location.pathname === '/dashboard' ? (
            <Outlet />
          ) : (
            <div className="mx-auto max-w-6xl rounded-2xl border border-[#d9dee7] bg-white p-4 shadow-[0_10px_30px_rgba(15,23,42,0.08)] sm:p-6 lg:p-8">
              <Outlet />
            </div>
          )}
        </section>
      </main>
    </div>
  );
}
