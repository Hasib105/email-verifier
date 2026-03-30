import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { ApiError, api, storageKeys } from './api'
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
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'

const DEFAULT_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:3000'

const toErrorMessage = (error: unknown) => {
  if (error instanceof ApiError || error instanceof Error) {
    return error.message
  }
  return 'Unexpected error'
}

const formatTimestamp = (value: number | undefined) => {
  if (!value) {
    return '-'
  }
  return new Date(value * 1000).toLocaleString()
}

const statusClassName = (status: string) => {
  if (status === 'valid' || status === 'accepted_no_bounce') return 'text-emerald-700'
  if (status === 'pending_bounce_check' || status === 'greylisted') return 'text-amber-700'
  if (status === 'invalid' || status === 'bounced' || status === 'error') return 'text-red-700'
  return 'text-slate-700'
}

function App() {
  const [baseUrl, setBaseUrl] = useState(() => localStorage.getItem(storageKeys.baseUrl) || DEFAULT_BASE_URL)
  const [apiKey, setApiKey] = useState(() => localStorage.getItem(storageKeys.apiKey) || '')
  const config = useMemo(() => ({ baseUrl, apiKey }), [baseUrl, apiKey])

  const [healthText, setHealthText] = useState('')
  const [torData, setTorData] = useState<TorCheckResponse | null>(null)
  const [systemLoading, setSystemLoading] = useState(false)
  const [systemError, setSystemError] = useState('')

  const [userData, setUserData] = useState<User | null>(null)
  const [webhookUrl, setWebhookUrl] = useState('')
  const [userLoading, setUserLoading] = useState(false)
  const [userMessage, setUserMessage] = useState('')
  const [userError, setUserError] = useState('')

  const [verifyEmail, setVerifyEmail] = useState('')
  const [verifyResult, setVerifyResult] = useState<VerifyResponse | null>(null)
  const [verifyLoading, setVerifyLoading] = useState(false)
  const [verifyError, setVerifyError] = useState('')

  const [csvFile, setCsvFile] = useState<File | null>(null)
  const [csvResult, setCsvResult] = useState<CsvImportResponse | null>(null)
  const [csvLoading, setCsvLoading] = useState(false)
  const [csvError, setCsvError] = useState('')

  const [smtpAccounts, setSmtpAccounts] = useState<SMTPAccount[]>([])
  const [smtpLoading, setSmtpLoading] = useState(false)
  const [smtpError, setSmtpError] = useState('')
  const [smtpMessage, setSmtpMessage] = useState('')
  const [smtpForm, setSmtpForm] = useState<SMTPAccountCreateRequest>({
    host: '',
    port: 587,
    username: '',
    password: '',
    sender: '',
    imap_host: '',
    imap_port: 993,
    imap_mailbox: 'INBOX',
    daily_limit: 100,
    active: true,
  })

  const [templates, setTemplates] = useState<EmailTemplate[]>([])
  const [templateLoading, setTemplateLoading] = useState(false)
  const [templateError, setTemplateError] = useState('')
  const [templateMessage, setTemplateMessage] = useState('')
  const [templateForm, setTemplateForm] = useState<EmailTemplateCreateRequest>({
    name: '',
    subject_template: 'Email verification probe {{token}}',
    body_template:
      'Hello,\n\nVerification probe for {{email}}.\nToken: {{token}}\nSender: {{sender}}\n',
    active: true,
  })

  useEffect(() => {
    localStorage.setItem(storageKeys.baseUrl, baseUrl)
  }, [baseUrl])

  useEffect(() => {
    localStorage.setItem(storageKeys.apiKey, apiKey)
  }, [apiKey])

  const loadSystemStatus = async () => {
    setSystemError('')
    setSystemLoading(true)
    try {
      const [health, tor] = await Promise.all([api.getHealth(baseUrl), api.getTorStatus(config)])
      setHealthText(health)
      setTorData(tor)
    } catch (error) {
      setSystemError(toErrorMessage(error))
    } finally {
      setSystemLoading(false)
    }
  }

  const loadUser = async (clearMessage = true) => {
    setUserError('')
    if (clearMessage) {
      setUserMessage('')
    }
    setUserLoading(true)
    try {
      const user = await api.getCurrentUser(config)
      setUserData(user)
      setWebhookUrl(user.webhook_url || '')
    } catch (error) {
      setUserError(toErrorMessage(error))
      setUserData(null)
    } finally {
      setUserLoading(false)
    }
  }

  const updateWebhook = async () => {
    setUserError('')
    setUserMessage('')
    setUserLoading(true)
    try {
      const result = await api.updateWebhook(config, webhookUrl)
      setUserMessage(result.message)
      await loadUser(false)
    } catch (error) {
      setUserError(toErrorMessage(error))
    } finally {
      setUserLoading(false)
    }
  }

  const runVerify = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!verifyEmail.trim()) {
      setVerifyError('Email is required')
      return
    }
    setVerifyLoading(true)
    setVerifyError('')
    try {
      const result = await api.verifyEmail(config, verifyEmail)
      setVerifyResult(result)
    } catch (error) {
      setVerifyError(toErrorMessage(error))
      setVerifyResult(null)
    } finally {
      setVerifyLoading(false)
    }
  }

  const runCsvImport = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!csvFile) {
      setCsvError('Select a CSV file first')
      return
    }
    setCsvError('')
    setCsvLoading(true)
    try {
      const result = await api.importCsv(config, csvFile)
      setCsvResult(result)
    } catch (error) {
      setCsvError(toErrorMessage(error))
      setCsvResult(null)
    } finally {
      setCsvLoading(false)
    }
  }

  const loadSmtpAccounts = async () => {
    setSmtpError('')
    setSmtpLoading(true)
    try {
      const result = await api.listSmtpAccounts(config)
      setSmtpAccounts(result.items)
    } catch (error) {
      setSmtpError(toErrorMessage(error))
      setSmtpAccounts([])
    } finally {
      setSmtpLoading(false)
    }
  }

  const createSmtpAccount = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSmtpError('')
    setSmtpMessage('')
    setSmtpLoading(true)
    try {
      await api.createSmtpAccount(config, smtpForm)
      setSmtpMessage('SMTP account created')
      setSmtpForm((prev) => ({ ...prev, password: '' }))
      await loadSmtpAccounts()
    } catch (error) {
      setSmtpError(toErrorMessage(error))
    } finally {
      setSmtpLoading(false)
    }
  }

  const loadTemplates = async () => {
    setTemplateError('')
    setTemplateLoading(true)
    try {
      const result = await api.listEmailTemplates(config)
      setTemplates(result.items)
    } catch (error) {
      setTemplateError(toErrorMessage(error))
      setTemplates([])
    } finally {
      setTemplateLoading(false)
    }
  }

  const createTemplate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setTemplateError('')
    setTemplateMessage('')
    setTemplateLoading(true)
    try {
      await api.createEmailTemplate(config, templateForm)
      setTemplateMessage('Template created')
      await loadTemplates()
    } catch (error) {
      setTemplateError(toErrorMessage(error))
    } finally {
      setTemplateLoading(false)
    }
  }

  return (
    <div className="mx-auto min-h-screen max-w-7xl space-y-4 p-4">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">Email Verifier Dashboard</h1>
          <p className="text-sm text-muted-foreground">Shadcn UI frontend for managing API workflows.</p>
        </div>
        <Button asChild variant="secondary">
          <a href={`${baseUrl.replace(/\/$/, '')}/swagger/`} target="_blank" rel="noreferrer">
            Open Swagger
          </a>
        </Button>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>API Connection</CardTitle>
          <CardDescription>Saved locally in browser storage for quick reuse.</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="baseUrl">API Base URL</Label>
            <Input id="baseUrl" value={baseUrl} onChange={(event) => setBaseUrl(event.target.value)} placeholder="http://localhost:3000" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="apiKey">API Key (X-API-Key)</Label>
            <Input id="apiKey" value={apiKey} onChange={(event) => setApiKey(event.target.value)} placeholder="evk_..." />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>System Status</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <Button disabled={systemLoading} onClick={loadSystemStatus}>
            {systemLoading ? 'Checking...' : 'Check Health + Tor'}
          </Button>
          {systemError ? <p className="text-sm text-red-700">{systemError}</p> : null}
          <div className="grid gap-2 text-sm md:grid-cols-2">
            <div><strong>Health:</strong> {healthText || '-'}</div>
            <div>
              <strong>Tor:</strong>{' '}
              <span className={torData ? (torData.is_tor ? 'text-emerald-700' : 'text-red-700') : 'text-slate-700'}>
                {torData ? (torData.is_tor ? 'Connected' : 'Not Routed') : '-'}
              </span>
            </div>
            <div><strong>Tor IP:</strong> {torData?.ip || '-'}</div>
            <div><strong>Message:</strong> {torData?.message || '-'}</div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>User & Webhook</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <Button
            disabled={userLoading}
            onClick={() => {
              void loadUser()
            }}
          >
            {userLoading ? 'Loading...' : 'Load Current User'}
          </Button>
          {userError ? <p className="text-sm text-red-700">{userError}</p> : null}
          {userMessage ? <p className="text-sm text-emerald-700">{userMessage}</p> : null}
          {userData ? (
            <div className="space-y-3">
              <div className="grid gap-2 text-sm md:grid-cols-2">
                <div><strong>Name:</strong> {userData.name}</div>
                <div><strong>Email:</strong> {userData.email}</div>
                <div><strong>Active:</strong> {userData.active ? 'Yes' : 'No'}</div>
                <div><strong>User ID:</strong> {userData.id}</div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="webhookUrl">Webhook URL</Label>
                <Input id="webhookUrl" value={webhookUrl} onChange={(event) => setWebhookUrl(event.target.value)} placeholder="https://example.com/webhook" />
              </div>
              <Button disabled={userLoading} onClick={updateWebhook}>
                {userLoading ? 'Saving...' : 'Update Webhook'}
              </Button>
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Single Email Verify</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <form onSubmit={runVerify} className="space-y-3">
            <div className="space-y-2">
              <Label htmlFor="verifyEmail">Email</Label>
              <Input id="verifyEmail" type="email" value={verifyEmail} onChange={(event) => setVerifyEmail(event.target.value)} placeholder="user@example.com" />
            </div>
            <Button type="submit" disabled={verifyLoading}>
              {verifyLoading ? 'Verifying...' : 'Verify Email'}
            </Button>
          </form>
          {verifyError ? <p className="text-sm text-red-700">{verifyError}</p> : null}
          {verifyResult ? (
            <div className="grid gap-1 rounded-md border p-3 text-sm">
              <div>
                <strong>Status:</strong> <span className={statusClassName(verifyResult.status)}>{verifyResult.status}</span>
              </div>
              <div><strong>Email:</strong> {verifyResult.email}</div>
              <div><strong>Message:</strong> {verifyResult.message}</div>
              <div><strong>Source:</strong> {verifyResult.source}</div>
              <div><strong>Cached:</strong> {verifyResult.cached ? 'Yes' : 'No'}</div>
              <div><strong>Finalized:</strong> {verifyResult.finalized ? 'Yes' : 'No'}</div>
              <div><strong>Next Check:</strong> {formatTimestamp(verifyResult.next_check_at)}</div>
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>CSV Import Verify</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <form onSubmit={runCsvImport} className="space-y-3">
            <div className="space-y-2">
              <Label htmlFor="csvFile">CSV File</Label>
              <Input id="csvFile" type="file" accept=".csv,text/csv" onChange={(event) => setCsvFile(event.target.files?.[0] || null)} />
            </div>
            <Button type="submit" disabled={csvLoading}>
              {csvLoading ? 'Importing...' : 'Import CSV'}
            </Button>
          </form>
          {csvError ? <p className="text-sm text-red-700">{csvError}</p> : null}
          {csvResult ? (
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">Total: {csvResult.total} | Accepted: {csvResult.accepted}</p>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Email</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Message</TableHead>
                    <TableHead>Source</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {csvResult.items.map((item, index) => (
                    <TableRow key={`${item.email}-${index}`}>
                      <TableCell>{item.email}</TableCell>
                      <TableCell><span className={statusClassName(item.status)}>{item.status}</span></TableCell>
                      <TableCell>{item.message}</TableCell>
                      <TableCell>{item.source || '-'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>SMTP Accounts</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <form onSubmit={createSmtpAccount} className="space-y-3">
            <div className="grid gap-3 md:grid-cols-3">
              <div className="space-y-2">
                <Label htmlFor="smtpHost">Host</Label>
                <Input id="smtpHost" value={smtpForm.host} onChange={(event) => setSmtpForm((prev) => ({ ...prev, host: event.target.value }))} placeholder="smtp.gmail.com" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="smtpPort">Port</Label>
                <Input id="smtpPort" type="number" value={smtpForm.port} onChange={(event) => setSmtpForm((prev) => ({ ...prev, port: Number(event.target.value) || 0 }))} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="smtpUsername">Username</Label>
                <Input id="smtpUsername" value={smtpForm.username} onChange={(event) => setSmtpForm((prev) => ({ ...prev, username: event.target.value }))} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="smtpPassword">Password</Label>
                <Input id="smtpPassword" type="password" value={smtpForm.password} onChange={(event) => setSmtpForm((prev) => ({ ...prev, password: event.target.value }))} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="smtpSender">Sender</Label>
                <Input id="smtpSender" value={smtpForm.sender} onChange={(event) => setSmtpForm((prev) => ({ ...prev, sender: event.target.value }))} placeholder="you@example.com" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="imapHost">IMAP Host</Label>
                <Input id="imapHost" value={smtpForm.imap_host} onChange={(event) => setSmtpForm((prev) => ({ ...prev, imap_host: event.target.value }))} placeholder="imap.gmail.com" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="imapPort">IMAP Port</Label>
                <Input id="imapPort" type="number" value={smtpForm.imap_port} onChange={(event) => setSmtpForm((prev) => ({ ...prev, imap_port: Number(event.target.value) || 0 }))} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="imapMailbox">IMAP Mailbox</Label>
                <Input id="imapMailbox" value={smtpForm.imap_mailbox} onChange={(event) => setSmtpForm((prev) => ({ ...prev, imap_mailbox: event.target.value }))} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="dailyLimit">Daily Limit</Label>
                <Input id="dailyLimit" type="number" value={smtpForm.daily_limit} onChange={(event) => setSmtpForm((prev) => ({ ...prev, daily_limit: Number(event.target.value) || 0 }))} />
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <Checkbox id="smtpActive" checked={smtpForm.active} onCheckedChange={(checked) => setSmtpForm((prev) => ({ ...prev, active: checked === true }))} />
              <Label htmlFor="smtpActive">Active</Label>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button type="submit" disabled={smtpLoading}>{smtpLoading ? 'Saving...' : 'Create SMTP Account'}</Button>
              <Button type="button" variant="secondary" disabled={smtpLoading} onClick={loadSmtpAccounts}>Refresh List</Button>
            </div>
          </form>
          {smtpError ? <p className="text-sm text-red-700">{smtpError}</p> : null}
          {smtpMessage ? <p className="text-sm text-emerald-700">{smtpMessage}</p> : null}
          {smtpAccounts.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Host</TableHead>
                  <TableHead>Sender</TableHead>
                  <TableHead>Usage</TableHead>
                  <TableHead>Reset Date</TableHead>
                  <TableHead>Active</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {smtpAccounts.map((account) => (
                  <TableRow key={account.id}>
                    <TableCell>{account.host}:{account.port}</TableCell>
                    <TableCell>{account.sender}</TableCell>
                    <TableCell>{account.sent_today}/{account.daily_limit}</TableCell>
                    <TableCell>{account.reset_date}</TableCell>
                    <TableCell>
                      <span className={account.active ? 'text-emerald-700' : 'text-red-700'}>{account.active ? 'Yes' : 'No'}</span>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Email Templates</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <form onSubmit={createTemplate} className="space-y-3">
            <div className="grid gap-3 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="templateName">Name</Label>
                <Input id="templateName" value={templateForm.name} onChange={(event) => setTemplateForm((prev) => ({ ...prev, name: event.target.value }))} placeholder="default-template" />
              </div>
              <div className="flex items-center space-x-2 pt-7">
                <Checkbox id="templateActive" checked={templateForm.active} onCheckedChange={(checked) => setTemplateForm((prev) => ({ ...prev, active: checked === true }))} />
                <Label htmlFor="templateActive">Active</Label>
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="templateSubject">Subject Template</Label>
              <Input id="templateSubject" value={templateForm.subject_template} onChange={(event) => setTemplateForm((prev) => ({ ...prev, subject_template: event.target.value }))} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="templateBody">Body Template</Label>
              <Textarea id="templateBody" rows={6} value={templateForm.body_template} onChange={(event) => setTemplateForm((prev) => ({ ...prev, body_template: event.target.value }))} />
            </div>
            <p className="text-sm text-muted-foreground">Placeholders: {'{{token}}'}, {'{{email}}'}, {'{{sender}}'}</p>
            <div className="flex flex-wrap gap-2">
              <Button type="submit" disabled={templateLoading}>{templateLoading ? 'Saving...' : 'Create Template'}</Button>
              <Button type="button" variant="secondary" disabled={templateLoading} onClick={loadTemplates}>Refresh List</Button>
            </div>
          </form>
          {templateError ? <p className="text-sm text-red-700">{templateError}</p> : null}
          {templateMessage ? <p className="text-sm text-emerald-700">{templateMessage}</p> : null}
          {templates.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Subject</TableHead>
                  <TableHead>Active</TableHead>
                  <TableHead>Updated</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {templates.map((template) => (
                  <TableRow key={template.id}>
                    <TableCell>{template.name}</TableCell>
                    <TableCell>{template.subject_template}</TableCell>
                    <TableCell>
                      <span className={template.active ? 'text-emerald-700' : 'text-red-700'}>{template.active ? 'Yes' : 'No'}</span>
                    </TableCell>
                    <TableCell>{formatTimestamp(template.updated_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : null}
        </CardContent>
      </Card>
    </div>
  )
}

export default App
