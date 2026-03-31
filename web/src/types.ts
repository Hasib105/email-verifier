export interface User {
  id: string
  name: string
  email: string
  api_key: string
  webhook_url: string
  is_superuser: boolean
  active: boolean
  created_at: number
  updated_at: number
}

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  user: User
  api_key: string
}

export interface RegisterRequest {
  name: string
  email: string
  password: string
  webhook_url?: string
}

export interface RegisterResponse {
  user: User
  api_key: string
}

export interface VerifyResponse {
  id: string
  email: string
  status: string
  message: string
  source: string
  cached: boolean
  finalized: boolean
  next_check_at?: number
}

export interface VerificationRecord {
  id: string
  email: string
  user_id: string
  status: string
  message: string
  source: string
  probe_token: string
  smtp_account_id: string
  check_count: number
  finalized: boolean
  first_checked_at: number
  last_checked_at: number
  next_check_at: number
  created_at: number
  updated_at: number
}

export interface VerificationStats {
  total: number
  by_status: Record<string, number>
}

export interface CsvImportResponse {
  total: number
  accepted: number
  items: VerifyResponse[]
}

export interface TorCheckResponse {
  is_tor: boolean
  ip: string
  message: string
}

export interface SMTPAccount {
  id: string
  user_id: string
  host: string
  port: number
  username: string
  sender: string
  imap_host: string
  imap_port: number
  imap_mailbox: string
  daily_limit: number
  sent_today: number
  reset_date: string
  active: boolean
  created_at: number
  updated_at: number
}

export interface EmailTemplate {
  id: string
  user_id: string
  name: string
  subject_template: string
  body_template: string
  active: boolean
  created_at: number
  updated_at: number
}

export interface SMTPAccountCreateRequest {
  host: string
  port: number
  username: string
  password: string
  sender: string
  imap_host: string
  imap_port: number
  imap_mailbox: string
  daily_limit: number
  active: boolean
}

export interface EmailTemplateCreateRequest {
  name: string
  subject_template: string
  body_template: string
  active: boolean
}
