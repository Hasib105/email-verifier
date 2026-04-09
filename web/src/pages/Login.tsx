import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Settings } from 'lucide-react';
import { DEFAULT_BASE_URL } from '../api';
import { useAuth } from '../context/AuthContext';

export function Login() {
  const navigate = useNavigate();
  const { login, baseUrl, setBaseUrl } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [apiUrl, setApiUrl] = useState(baseUrl);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await login(email, password);
      navigate('/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveSettings = () => {
    setBaseUrl(apiUrl);
    setShowSettings(false);
  };

  return (
    <div>
      <div className="mb-8">
        <p className="text-xs font-semibold uppercase tracking-widest text-slate-500">Welcome back</p>
        <h1 className="mt-2 text-3xl font-bold tracking-tight text-slate-950">Sign in</h1>
        <p className="mt-2 text-sm text-slate-500">Access your verification console.</p>
      </div>

      {error && (
        <div className="mb-5 border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      <form className="space-y-4" onSubmit={handleLogin}>
        <div>
          <label className="block text-sm font-semibold text-slate-700">Email</label>
          <input
            className="mt-1 block h-11 w-full border border-slate-300 px-3 text-sm text-slate-950 outline-none transition-colors placeholder:text-slate-400 focus:border-slate-950 focus:ring-1 focus:ring-slate-950"
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            required
            type="email"
            value={email}
          />
        </div>

        <div>
          <label className="block text-sm font-semibold text-slate-700">Password</label>
          <input
            className="mt-1 block h-11 w-full border border-slate-300 px-3 text-sm text-slate-950 outline-none transition-colors placeholder:text-slate-400 focus:border-slate-950 focus:ring-1 focus:ring-slate-950"
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Password"
            required
            type="password"
            value={password}
          />
        </div>

        <button
          className="flex h-11 w-full items-center justify-center bg-slate-950 px-4 text-sm font-semibold text-white transition-colors hover:bg-slate-800 disabled:opacity-50"
          disabled={loading}
          type="submit"
        >
          {loading ? 'Signing in...' : 'Sign in'}
        </button>
      </form>

      <div className="mt-6 text-center text-sm">
        <span className="text-slate-500">No account yet? </span>
        <Link className="font-semibold text-slate-950 underline-offset-4 hover:underline" to="/auth/register">
          Create one
        </Link>
      </div>

      <div className="mt-6 border-t border-slate-200 pt-5">
        <button
          className="inline-flex items-center gap-2 text-sm font-medium text-slate-500 hover:text-slate-950"
          onClick={() => setShowSettings(!showSettings)}
          type="button"
        >
          <Settings className="h-4 w-4" />
          API Settings
        </button>

        {showSettings && (
          <div className="mt-4 border border-slate-200 bg-slate-50 p-4">
            <label className="block text-sm font-semibold text-slate-700">API Base URL</label>
            <input
              className="mt-2 block h-10 w-full border border-slate-300 bg-white px-3 text-sm outline-none focus:border-slate-950 focus:ring-1 focus:ring-slate-950"
              onChange={(e) => setApiUrl(e.target.value)}
              placeholder={DEFAULT_BASE_URL}
              type="url"
              value={apiUrl}
            />
            <button
              className="mt-3 h-9 w-full border border-slate-300 bg-white px-3 text-sm font-semibold text-slate-950 hover:bg-slate-100"
              onClick={handleSaveSettings}
              type="button"
            >
              Save Settings
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
