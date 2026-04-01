import type {
  CsvImportResponse,
  EmailTemplate,
  EmailTemplateCreateRequest,
  HealthResponse,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RegisterResponse,
  SMTPAccount,
  SMTPAccountCreateRequest,
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

async function request<T>(
  config: ApiConfig,
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const baseUrl = normalizeBaseUrl(config.baseUrl || DEFAULT_BASE_URL)
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (config.apiKey.trim() !== '') {
    headers.set('X-API-Key', config.apiKey.trim())
  }

  const res = await fetch(`${baseUrl}${path}`, { ...init, headers })
  if (!res.ok) {
    let message = `Request failed with status ${res.status}`
    try {
      const data = (await res.json()) as { error?: string; message?: string }
      if (data.error) {
        message = data.error
      } else if (data.message) {
        message = data.message
      }
    } catch {
      // keep default message when non-json response
    }
    throw new ApiError(message, res.status)
  }

  return (await res.json()) as T
}

async function requestNoAuth<T>(
  baseUrl: string,
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const url = `${normalizeBaseUrl(baseUrl || DEFAULT_BASE_URL)}${path}`
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')

  const res = await fetch(url, { ...init, headers })
  if (!res.ok) {
    let message = `Request failed with status ${res.status}`
    try {
      const data = (await res.json()) as { error?: string; message?: string }
      if (data.error) {
        message = data.error
      } else if (data.message) {
        message = data.message
      }
    } catch {
      // keep default message when non-json response
    }
    throw new ApiError(message, res.status)
  }

  return (await res.json()) as T
}

export const api = {
  // Health
  getHealth: (config: ApiConfig) =>
    request<HealthResponse>(config, '/health', { method: 'GET' }),

  // Auth
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

  // User
  getCurrentUser: (config: ApiConfig) =>
    request<User>(config, '/users/me', { method: 'GET' }),

  regenerateAPIKey: (config: ApiConfig) =>
    request<{ api_key: string }>(config, '/users/api-key/regenerate', {
      method: 'POST',
    }),

  updateWebhook: (config: ApiConfig, webhookUrl: string) =>
    request<{ message: string }>(config, '/users/webhook', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ webhook_url: webhookUrl }),
    }),

  // Verification
  verifyEmail: (config: ApiConfig, email: string) =>
    request<VerifyResponse>(config, '/verify', {
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

    const res = await fetch(`${baseUrl}/verify/import-csv`, {
      method: 'POST',
      headers,
      body: form,
    })
    if (!res.ok) {
      let message = `CSV import failed with status ${res.status}`
      try {
        const data = (await res.json()) as { error?: string }
        if (data.error) {
          message = data.error
        }
      } catch {
        // keep default message
      }
      throw new ApiError(message, res.status)
    }
    return (await res.json()) as CsvImportResponse
  },

  listVerifications: (config: ApiConfig, limit = 50, offset = 0) =>
    request<{ items: VerificationRecord[] }>(config, `/verifications?limit=${limit}&offset=${offset}`, { method: 'GET' }),

  getVerification: (config: ApiConfig, id: string) =>
    request<VerificationRecord>(config, `/verifications/${id}`, { method: 'GET' }),

  deleteVerification: (config: ApiConfig, id: string) =>
    request<{ message: string }>(config, `/verifications/${id}`, { method: 'DELETE' }),

  getVerificationStats: (config: ApiConfig) =>
    request<VerificationStats>(config, '/verifications/stats', { method: 'GET' }),

  // SMTP Accounts
  createSmtpAccount: (config: ApiConfig, payload: SMTPAccountCreateRequest) =>
    request<SMTPAccount>(config, '/smtp-accounts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }),

  listSmtpAccounts: (config: ApiConfig) =>
    request<{ items: SMTPAccount[] }>(config, '/smtp-accounts', { method: 'GET' }),

  getSmtpAccount: (config: ApiConfig, id: string) =>
    request<SMTPAccount>(config, `/smtp-accounts/${id}`, { method: 'GET' }),

  updateSmtpAccount: (config: ApiConfig, id: string, payload: SMTPAccountCreateRequest) =>
    request<SMTPAccount>(config, `/smtp-accounts/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }),

  deleteSmtpAccount: (config: ApiConfig, id: string) =>
    request<{ message: string }>(config, `/smtp-accounts/${id}`, { method: 'DELETE' }),

  // Email Templates
  createEmailTemplate: (config: ApiConfig, payload: EmailTemplateCreateRequest) =>
    request<EmailTemplate>(config, '/email-templates', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }),

  listEmailTemplates: (config: ApiConfig) =>
    request<{ items: EmailTemplate[] }>(config, '/email-templates', { method: 'GET' }),

  getEmailTemplate: (config: ApiConfig, id: string) =>
    request<EmailTemplate>(config, `/email-templates/${id}`, { method: 'GET' }),

  updateEmailTemplate: (config: ApiConfig, id: string, payload: EmailTemplateCreateRequest) =>
    request<EmailTemplate>(config, `/email-templates/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }),

  deleteEmailTemplate: (config: ApiConfig, id: string) =>
    request<{ message: string }>(config, `/email-templates/${id}`, { method: 'DELETE' }),

  // Admin endpoints
  adminListUsers: (config: ApiConfig) =>
    request<{ items: User[] }>(config, '/admin/users', { method: 'GET' }),

  adminListVerifications: (config: ApiConfig, limit = 50, offset = 0) =>
    request<{ items: VerificationRecord[] }>(config, `/admin/verifications?limit=${limit}&offset=${offset}`, { method: 'GET' }),

  adminListSmtpAccounts: (config: ApiConfig) =>
    request<{ items: SMTPAccount[] }>(config, '/admin/smtp-accounts', { method: 'GET' }),

  adminListTemplates: (config: ApiConfig) =>
    request<{ items: EmailTemplate[] }>(config, '/admin/email-templates', { method: 'GET' }),

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

  // Settings
  getSettings: (config: ApiConfig) =>
    request<{ webhook_url: string }>(config, '/users/me', { method: 'GET' }),

  updateSettings: (config: ApiConfig, payload: { webhook_url: string }) =>
    request<{ message: string }>(config, '/users/webhook', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }),

  testWebhook: (config: ApiConfig, webhookUrl: string) =>
    request<{ message: string }>(config, '/users/webhook/test', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ webhook_url: webhookUrl }),
    }),
}

export const storageKeys = {
  baseUrl: 'email-verifier.base-url',
  apiKey: 'email-verifier.api-key',
  user: 'email-verifier.user',
}
