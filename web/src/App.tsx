import { Routes, Route, Navigate } from 'react-router-dom';
import { AuthLayout } from './layouts/AuthLayout';
import { DashboardLayout } from './layouts/DashboardLayout';
import { Login } from './pages/Login';
import { Register } from './pages/Register';
import { Dashboard } from './pages/Dashboard';
import { Playground } from './pages/Playground';
import { Settings } from './pages/Settings';
import { EmailStatus } from './pages/EmailStatus';
import { AdminPanel } from './pages/Admin';
import { ProtectedRoute } from './components/ProtectedRoute';

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/auth/login" replace />} />
      <Route path="/auth" element={<AuthLayout />}>
        <Route path="login" element={<Login />} />
        <Route path="register" element={<Register />} />
      </Route>
      <Route 
        path="/dashboard" 
        element={
          <ProtectedRoute>
            <DashboardLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="playground" element={<Playground />} />
        <Route path="settings" element={<Settings />} />
        <Route path="verifications" element={<EmailStatus />} />
        <Route path="admin" element={<AdminPanel />} />
      </Route>
    </Routes>
  );
}
