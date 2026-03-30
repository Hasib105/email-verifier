import type {
  CsvImportResponse,
  EmailTemplate,
  EmailTemplateCreateRequest,
  SMTPAccount,
  SMTPAccountCreateRequest,
  TorCheckResponse,
  User,
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

export const api = {
  getHealth: async (baseUrl: string) => {
    const url = `${normalizeBaseUrl(baseUrl || DEFAULT_BASE_URL)}/health`
    const response = await fetch(url)
    if (!response.ok) {
      throw new ApiError(`Health check failed with status ${response.status}`, response.status)
    }
    return response.text()
  },

  getTorStatus: (config: ApiConfig) =>
    request<TorCheckResponse>(config, '/check-tor', { method: 'GET' }),

  getCurrentUser: (config: ApiConfig) =>
    request<User>(config, '/users/me', { method: 'GET' }),

  updateWebhook: (config: ApiConfig, webhookUrl: string) =>
    request<{ message: string }>(config, '/users/webhook', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ webhook_url: webhookUrl }),
    }),

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

  createSmtpAccount: (config: ApiConfig, payload: SMTPAccountCreateRequest) =>
    request<SMTPAccount>(config, '/smtp-accounts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }),

  listSmtpAccounts: (config: ApiConfig) =>
    request<{ items: SMTPAccount[] }>(config, '/smtp-accounts', { method: 'GET' }),

  createEmailTemplate: (config: ApiConfig, payload: EmailTemplateCreateRequest) =>
    request<EmailTemplate>(config, '/email-templates', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }),

  listEmailTemplates: (config: ApiConfig) =>
    request<{ items: EmailTemplate[] }>(config, '/email-templates', { method: 'GET' }),
}

export const storageKeys = {
  baseUrl: 'email-verifier.base-url',
  apiKey: 'email-verifier.api-key',
}
