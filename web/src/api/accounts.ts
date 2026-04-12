import api from './axios'

export interface Account {
  id: number
  email: string
  type: string
  status: string
  extra?: Record<string, unknown>
  usage?: Record<string, unknown>
  task_batch_id: number
  created_at: string
}

export function getAccounts(params?: { type?: string; status?: string; page?: number; limit?: number }) {
  return api.get<{ accounts: Account[]; total: number }>('/accounts', { params })
}

export function exportAccounts(type?: string) {
  return api.get('/accounts/export', {
    params: { type },
    responseType: 'blob',
  })
}

export interface ImportAccountRecord {
  email: string
  password: string
  type: string
  status?: string
  extra?: Record<string, unknown>
}

export function importAccounts(records: ImportAccountRecord[]) {
  return api.post<{ imported: number; skipped: number; failed: number }>('/accounts/import', records)
}

export interface AccountCheckResult {
  supported: boolean
  valid: boolean
  message: string
  status?: string
  usage?: Record<string, unknown>
}

export function checkAccount(id: number) {
  return api.post<AccountCheckResult>(`/accounts/${id}/check`)
}

export function deleteAccount(id: number) {
  return api.delete(`/accounts/${id}`)
}

export interface ChatGPTRefreshTokenResult {
  account_id: string
  access_token: string
  access_token_expires_at?: string
  refresh_token: string
}

export interface ChatGPTAccountDetailResult {
  account_id: string
  default_account_id?: string
  email?: string
  plan_type?: string
  accounts?: Array<{
    id: string
    account_user_id: string
    structure: string
    plan_type: string
    name?: string | null
    profile_picture_url?: string | null
  }>
  usage?: Record<string, unknown>
  extra?: Record<string, unknown>
}

export function refreshChatGPTToken(id: number) {
  return api.post<ChatGPTRefreshTokenResult>(`/accounts/${id}/chatgpt/refresh-token`)
}

export function getChatGPTAccountDetail(id: number) {
  return api.get<ChatGPTAccountDetailResult>(`/accounts/${id}/chatgpt/detail`)
}
