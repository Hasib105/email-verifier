import type {
  CsvImportResponse,
  HealthResponse,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RegisterResponse,
  User,
  VerificationRecord,
  VerificationStats,
  VerifyResponse,
} from './types'

const DEFAULT_BASE_URL = 'http://localhost:3000'

export interface ApiConfig {
  baseUrl: string
  apiKey: string
}

export class ApiError extends Error {
  readonly status: number

  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

const normalizeBaseUrl = (baseUrl: string) => {
  const trimmed = baseUrl.trim()
  return trimmed.endsWith('/') ? trimmed.slice(0, -1) : trimmed
}

async function request<T>(config: ApiConfig, path: string, init: RequestInit = {}): Promise<T> {
  const url = `${normalizeBaseUrl(config.baseUrl || DEFAULT_BASE_URL)}${path}`
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (config.apiKey.trim()) {
    headers.set('X-API-Key', config.apiKey.trim())
  }

  const res = await fetch(url, { ...init, headers })
  if (!res.ok) {
    let message = `Request failed with status ${res.status}`
    try {
      const data = (await res.json()) as { error?: string; message?: string }
      message = data.error || data.message || message
    } catch {
      // keep default
    }
    throw new ApiError(message, res.status)
  }

  return (await res.json()) as T
}

async function requestNoAuth<T>(baseUrl: string, path: string, init: RequestInit = {}): Promise<T> {
  const url = `${normalizeBaseUrl(baseUrl || DEFAULT_BASE_URL)}${path}`
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')

  const res = await fetch(url, { ...init, headers })
  if (!res.ok) {
    let message = `Request failed with status ${res.status}`
    try {
      const data = (await res.json()) as { error?: string; message?: string }
      message = data.error || data.message || message
    } catch {
      // keep default
    }
    throw new ApiError(message, res.status)
  }

  return (await res.json()) as T
}

export const api = {
  getHealth: (config: ApiConfig) =>
    request<HealthResponse>(config, '/health', { method: 'GET' }),

  login: (baseUrl: string, req: LoginRequest) =>
    requestNoAuth<LoginResponse>(baseUrl, '/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    }),

  register: (baseUrl: string, req: RegisterRequest) =>
    requestNoAuth<RegisterResponse>(baseUrl, '/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    }),

  getCurrentUser: (config: ApiConfig) =>
    request<User>(config, '/users/me', { method: 'GET' }),

  regenerateAPIKey: (config: ApiConfig) =>
    request<{ api_key: string }>(config, '/users/api-key/regenerate', { method: 'POST' }),

  createVerification: (config: ApiConfig, email: string) =>
    request<VerifyResponse>(config, '/verifications', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email }),
    }),

  importCsv: async (config: ApiConfig, file: File) => {
    const baseUrl = normalizeBaseUrl(config.baseUrl || DEFAULT_BASE_URL)
    const headers = new Headers()
    headers.set('Accept', 'application/json')
    headers.set('X-API-Key', config.apiKey.trim())

    const form = new FormData()
    form.append('file', file)

    const res = await fetch(`${baseUrl}/verifications/import-csv`, {
      method: 'POST',
      headers,
      body: form,
    })
    if (!res.ok) {
      let message = `CSV import failed with status ${res.status}`
      try {
        const data = (await res.json()) as { error?: string }
        if (data.error) message = data.error
      } catch {
        // keep default
      }
      throw new ApiError(message, res.status)
    }
    return (await res.json()) as CsvImportResponse
  },

  listVerifications: (config: ApiConfig, limit = 50, offset = 0) =>
    request<{ items: VerificationRecord[] }>(config, `/verifications?limit=${limit}&offset=${offset}`, { method: 'GET' }),

  getVerification: (config: ApiConfig, id: string) =>
    request<VerifyResponse>(config, `/verifications/${id}`, { method: 'GET' }),

  deleteVerification: (config: ApiConfig, id: string) =>
    request<{ message: string }>(config, `/verifications/${id}`, { method: 'DELETE' }),

  getVerificationStats: (config: ApiConfig) =>
    request<VerificationStats>(config, '/verifications/stats', { method: 'GET' }),

  adminListUsers: (config: ApiConfig) =>
    request<{ items: User[] }>(config, '/admin/users', { method: 'GET' }),

  adminListVerifications: (config: ApiConfig, limit = 50, offset = 0) =>
    request<{ items: VerificationRecord[] }>(config, `/admin/verifications?limit=${limit}&offset=${offset}`, { method: 'GET' }),

  adminUpdateUser: (config: ApiConfig, id: string, payload: { is_superuser?: boolean }) =>
    request<User>(config, `/admin/users/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }),

  adminDeleteUser: (config: ApiConfig, id: string) =>
    request<{ message: string }>(config, `/admin/users/${id}`, { method: 'DELETE' }),

  adminDeleteVerification: (config: ApiConfig, id: string) =>
    request<{ message: string }>(config, `/admin/verifications/${id}`, { method: 'DELETE' }),
}

export const storageKeys = {
  baseUrl: 'email-verifier.base-url',
  apiKey: 'email-verifier.api-key',
  user: 'email-verifier.user',
}
