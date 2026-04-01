import { useState } from 'react';
import { Mail, CheckCircle2, AlertCircle, Clock, Loader2, Server } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { VerifyResponse } from '../types';

export function Playground() {
  const { config } = useAuth();
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<VerifyResponse | null>(null);
  const [error, setError] = useState('');

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email) return;

    setLoading(true);
    setError('');
    setResult(null);

    try {
      const response = await api.verifyEmail(config, email);
      setResult(response);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred during verification.');
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'valid':
        return <CheckCircle2 className="w-8 h-8 text-green-500" />;
      case 'invalid':
      case 'bounced':
      case 'error':
        return <AlertCircle className="w-8 h-8 text-red-500" />;
      case 'pending_bounce_check':
      case 'greylisted':
        return <Clock className="w-8 h-8 text-yellow-500" />;
      default:
        return <AlertCircle className="w-8 h-8 text-gray-500" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'valid':
        return 'bg-green-100 text-green-800 border-green-200';
      case 'invalid':
      case 'bounced':
        return 'bg-red-100 text-red-800 border-red-200';
      case 'pending_bounce_check':
      case 'greylisted':
        return 'bg-yellow-100 text-yellow-800 border-yellow-200';
      case 'error':
      default:
        return 'bg-gray-100 text-gray-800 border-gray-200';
    }
  };

  return (
    <div className="p-8 max-w-4xl mx-auto">
      <div className="mb-8">
        <h1 className="text-2xl font-bold mb-2 flex items-center gap-2">
          <Mail className="w-6 h-6 text-yellow-500" />
          Verification Playground
        </h1>
        <p className="text-gray-600">
          Test the email verification API instantly. Enter an email address below to see real-time results.
        </p>
      </div>

      <div className="bg-white rounded-xl shadow-sm border p-6 mb-8">
        <form onSubmit={handleVerify} className="flex gap-4">
          <div className="flex-1 relative">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <Mail className="h-5 w-5 text-gray-400" />
            </div>
            <input
              type="email"
              required
              className="block w-full pl-10 pr-3 py-3 border border-gray-300 rounded-lg focus:ring-yellow-500 focus:border-yellow-500 sm:text-sm transition-colors"
              placeholder="Enter email address to verify (e.g. test@example.com)"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={loading}
            />
          </div>
          <button
            type="submit"
            disabled={loading || !email}
            className="inline-flex items-center px-6 py-3 border border-transparent text-sm font-medium rounded-lg shadow-sm text-white bg-yellow-600 hover:bg-yellow-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-yellow-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {loading ? (
              <>
                <Loader2 className="animate-spin -ml-1 mr-2 h-5 w-5" />
                Verifying...
              </>
            ) : (
              'Verify Email'
            )}
          </button>
        </form>
      </div>

      {error && (
        <div className="bg-red-50 border-l-4 border-red-400 p-4 mb-8 text-red-700 rounded-r-lg">
          <div className="flex">
            <div className="flex-shrink-0">
              <AlertCircle className="h-5 w-5 text-red-400" />
            </div>
            <div className="ml-3">
              <p className="text-sm font-medium">{error}</p>
            </div>
          </div>
        </div>
      )}

      {result && (
        <div className="bg-white rounded-xl shadow-sm border overflow-hidden animate-in fade-in slide-in-from-bottom-4 duration-500">
          <div className="px-6 py-5 border-b bg-gray-50 flex items-center justify-between">
            <h3 className="text-lg font-medium leading-6 text-gray-900 flex items-center gap-2">
              <Server className="w-5 h-5 text-gray-500" />
              Verification Result
            </h3>
            <span className={`px-3 py-1 rounded-full text-xs font-semibold border uppercase tracking-wider ${getStatusColor(result.status)}`}>
              {result.status.replace(/_/g, ' ')}
            </span>
          </div>
          <div className="px-6 py-6">
            <div className="flex items-start gap-6">
              <div className="flex-shrink-0">
                {getStatusIcon(result.status)}
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-500 uppercase tracking-wider mb-1">Target Email</p>
                <p className="text-xl font-bold text-gray-900 mb-4">{result.email}</p>
                
                <div className="bg-gray-50 rounded-lg p-4 border border-gray-100 mb-6">
                  <p className="text-sm font-medium text-gray-500 uppercase tracking-wider mb-2">Message Detail</p>
                  <p className="text-gray-900">{result.message}</p>
                </div>

                <dl className="grid grid-cols-1 gap-x-4 gap-y-6 sm:grid-cols-2">
                  <div className="sm:col-span-1">
                    <dt className="text-sm font-medium text-gray-500 uppercase tracking-wider">Source</dt>
                    <dd className="mt-1 text-sm text-gray-900">{result.source}</dd>
                  </div>
                  <div className="sm:col-span-1">
                    <dt className="text-sm font-medium text-gray-500 uppercase tracking-wider">Cached</dt>
                    <dd className="mt-1 text-sm text-gray-900">{result.cached ? 'Yes' : 'No'}</dd>
                  </div>
                  <div className="sm:col-span-1">
                    <dt className="text-sm font-medium text-gray-500 uppercase tracking-wider">Finalized</dt>
                    <dd className="mt-1 text-sm text-gray-900">{result.finalized ? 'Yes' : 'No'}</dd>
                  </div>
                </dl>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
