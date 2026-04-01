import { useState } from 'react';
import { Outlet, NavLink, useLocation, useNavigate } from 'react-router-dom';
import { LayoutDashboard, Settings, Activity, ShieldAlert, LogOut, Menu, X, PlayCircle } from 'lucide-react';
import { useAuth } from '../context/AuthContext';

export function DashboardLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { logout, user } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const navigation = [
    { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
    { name: 'Playground', href: '/dashboard/playground', icon: PlayCircle },
    { name: 'Verifications', href: '/dashboard/verifications', icon: Activity },
    { name: 'Settings', href: '/dashboard/settings', icon: Settings },
    // Only show Admin link for superusers
    ...(user?.is_superuser ? [{ name: 'Admin', href: '/dashboard/admin', icon: ShieldAlert }] : []),
  ];

  const handleLogout = () => {
    logout();
    navigate('/auth/login');
  };

  const closeSidebar = () => setSidebarOpen(false);

  return (
    <div className="flex h-screen bg-gray-50 text-gray-900">
      {/* Mobile overlay */}
      {sidebarOpen && (
        <div 
          className="fixed inset-0 bg-black/50 z-40 lg:hidden"
          onClick={closeSidebar}
        />
      )}

      {/* Sidebar */}
      <aside className={`
        fixed lg:static inset-y-0 left-0 z-50
        w-64 bg-white shadow-sm flex flex-col justify-between
        m-0 lg:m-4 lg:rounded-xl overflow-hidden border
        transform transition-transform duration-200 ease-in-out
        ${sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}
      `}>
        <div className="p-6 w-full h-full space-y-8 flex flex-col">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-3 text-xl font-bold bg-gradient-to-r from-yellow-400 to-yellow-600 bg-clip-text text-transparent">
              <div className="bg-yellow-400 text-white p-2 rounded-lg"><LayoutDashboard size={24}/></div>
              <span>Verifier</span>
            </div>
            <button 
              onClick={closeSidebar}
              className="lg:hidden p-2 text-gray-500 hover:text-gray-700"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          <div className="flex-1 mt-6 w-full">
            <p className="text-xs text-gray-500 font-semibold mb-4 uppercase">Main Menu</p>
            <nav className="flex flex-col space-y-1 w-full">
              {navigation.map((item) => {
                const isActive = location.pathname === item.href;
                return (
                  <NavLink
                    key={item.name}
                    to={item.href}
                    onClick={closeSidebar}
                    className={`flex items-center space-x-3 px-4 py-3 rounded-lg text-sm font-medium transition-colors ${
                      isActive 
                        ? 'bg-gray-100 text-black' 
                        : 'text-gray-500 hover:bg-gray-50 hover:text-black'
                    }`}
                  >
                    <item.icon className="w-5 h-5" />
                    <span>{item.name}</span>
                  </NavLink>
                );
              })}
            </nav>
          </div>
        </div>
        <div className="p-4 w-full border-t">
          {user && (
            <div className="px-4 py-2 mb-2 text-xs text-gray-500 truncate">
              {user.email}
              {user.is_superuser && (
                <span className="ml-2 px-1.5 py-0.5 bg-purple-100 text-purple-700 rounded text-[10px] font-semibold">
                  Admin
                </span>
              )}
            </div>
          )}
          <button 
            onClick={handleLogout}
            className="flex items-center space-x-3 px-4 py-3 rounded-lg text-sm font-medium text-red-500 hover:bg-red-50 transition-colors w-full"
          >
            <LogOut className="w-5 h-5"/>
            <span>Log out</span>
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-auto">
        {/* Mobile header */}
        <div className="lg:hidden sticky top-0 z-30 bg-white border-b px-4 py-3 flex items-center justify-between">
          <button 
            onClick={() => setSidebarOpen(true)}
            className="p-2 text-gray-700 hover:bg-gray-100 rounded-lg"
          >
            <Menu className="w-6 h-6" />
          </button>
          <div className="flex items-center space-x-2 text-lg font-bold bg-gradient-to-r from-yellow-400 to-yellow-600 bg-clip-text text-transparent">
            <div className="bg-yellow-400 text-white p-1.5 rounded-lg"><LayoutDashboard size={18}/></div>
            <span>Verifier</span>
          </div>
          <div className="w-10" /> {/* Spacer for centering */}
        </div>
        
        <div className="p-4 lg:p-8">
          <div className="max-w-6xl mx-auto border bg-white rounded-xl shadow-sm min-h-[calc(100vh-8rem)] lg:min-h-[85vh] p-4 sm:p-6 lg:p-8 hidden-scrollbar overflow-y-auto relative">
            <Outlet />
          </div>
        </div>
      </main>
    </div>
  );
}
