export interface User {
  id: string
  name: string
  email: string
  api_key: string
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
}

export interface RegisterResponse {
  user: User
  api_key: string
}

export interface EnrichmentEvidence {
  id: string
  verification_id: string
  source: string
  kind: string
  signal: string
  weight: number
  summary: string
  created_at: number
}

export interface VerificationCallout {
  id: number
  verification_id: string
  smtp_host: string
  smtp_port: number
  stage: string
  recipient: string
  outcome: string
  smtp_code: number
  smtp_message: string
  duration_ms: number
  created_at: number
}

export interface VerificationRecord {
  id: string
  email: string
  domain: string
  user_id: string
  classification: 'deliverable' | 'undeliverable' | 'accept_all' | 'unknown'
  confidence_score: number
  risk_level: 'low' | 'medium' | 'high'
  deterministic: boolean
  state: 'completed' | 'enriching'
  reason_codes: string[]
  protocol_summary: string
  enrichment_summary: string
  expires_at: number
  last_verified_at: number
  last_enriched_at: number
  created_at: number
  updated_at: number
}

export interface VerifyResponse extends VerificationRecord {
  cached: boolean
  evidence?: EnrichmentEvidence[]
  callouts?: VerificationCallout[]
}

export interface VerificationStats {
  total: number
  by_classification: Record<string, number>
}

export interface CsvImportResponse {
  total: number
  accepted: number
  items: VerifyResponse[]
}

export interface HealthResponse {
  status: string
  mode: string
  mail_from: string
  ehlo_domain: string
  max_parallel: number
  baseline_ttl: string
  deliverable_ttl: string
}
