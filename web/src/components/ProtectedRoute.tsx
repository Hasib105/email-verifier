import { Navigate, useLocation } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

interface ProtectedRouteProps {
  children: React.ReactNode;
  requireSuperuser?: boolean;
}

export function ProtectedRoute({ children, requireSuperuser = false }: ProtectedRouteProps) {
  const { isAuthenticated, isLoading, isSuperuser } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-yellow-500"></div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/auth/login" state={{ from: location }} replace />;
  }

  if (requireSuperuser && !isSuperuser) {
    return <Navigate to="/dashboard" replace />;
  }

  return <>{children}</>;
}
