import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import type { User } from '../types';
import { api, DEFAULT_BASE_URL, storageKeys, type ApiConfig } from '../api';

interface AuthContextType {
  user: User | null;
  apiKey: string;
  baseUrl: string;
  isLoading: boolean;
  isAuthenticated: boolean;
  isSuperuser: boolean;
  config: ApiConfig;
  login: (email: string, password: string) => Promise<void>;
  register: (name: string, email: string, password: string) => Promise<void>;
  logout: () => void;
  setBaseUrl: (url: string) => void;
  updateApiKey: (apiKey: string) => void;
}

const AuthContext = createContext<AuthContextType | null>(null);
const legacyBaseUrls = new Set(['http://localhost:3000', 'https://localhost:3000']);

const resolveInitialBaseUrl = () => {
  const storedBaseUrl = localStorage.getItem(storageKeys.baseUrl);
  if (!storedBaseUrl || legacyBaseUrls.has(storedBaseUrl)) {
    localStorage.setItem(storageKeys.baseUrl, DEFAULT_BASE_URL);
    return DEFAULT_BASE_URL;
  }
  return storedBaseUrl;
};

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [apiKey, setApiKey] = useState(() => localStorage.getItem(storageKeys.apiKey) || '');
  const [baseUrl, setBaseUrlState] = useState(resolveInitialBaseUrl);
  const [isLoading, setIsLoading] = useState(true);

  const config: ApiConfig = { baseUrl, apiKey };

  useEffect(() => {
    const loadUser = async () => {
      const storedUser = localStorage.getItem(storageKeys.user);
      const storedApiKey = localStorage.getItem(storageKeys.apiKey);
      
      if (storedUser && storedApiKey) {
        try {
          const parsedUser = JSON.parse(storedUser) as User;
          setUser(parsedUser);
          setApiKey(storedApiKey);
          
          // Verify the session is still valid
          const freshUser = await api.getCurrentUser({ baseUrl, apiKey: storedApiKey });
          setUser(freshUser);
          localStorage.setItem(storageKeys.user, JSON.stringify(freshUser));
        } catch {
          // Session expired or invalid
          localStorage.removeItem(storageKeys.user);
          localStorage.removeItem(storageKeys.apiKey);
          setUser(null);
          setApiKey('');
        }
      }
      setIsLoading(false);
    };

    loadUser();
  }, [baseUrl]);

  const login = async (email: string, password: string) => {
    const response = await api.login(baseUrl, { email, password });
    setUser(response.user);
    setApiKey(response.api_key);
    localStorage.setItem(storageKeys.user, JSON.stringify(response.user));
    localStorage.setItem(storageKeys.apiKey, response.api_key);
  };

  const register = async (name: string, email: string, password: string) => {
    const response = await api.register(baseUrl, { name, email, password });
    setUser(response.user);
    setApiKey(response.api_key);
    localStorage.setItem(storageKeys.user, JSON.stringify(response.user));
    localStorage.setItem(storageKeys.apiKey, response.api_key);
  };

  const logout = () => {
    setUser(null);
    setApiKey('');
    localStorage.removeItem(storageKeys.user);
    localStorage.removeItem(storageKeys.apiKey);
  };

  const setBaseUrl = (url: string) => {
    setBaseUrlState(url);
    localStorage.setItem(storageKeys.baseUrl, url);
  };

  const updateApiKey = (newKey: string) => {
    setApiKey(newKey);
    localStorage.setItem(storageKeys.apiKey, newKey);
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        apiKey,
        baseUrl,
        isLoading,
        isAuthenticated: !!user && !!apiKey,
        isSuperuser: user?.is_superuser ?? false,
        config,
        login,
        register,
        logout,
        setBaseUrl,
        updateApiKey,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
