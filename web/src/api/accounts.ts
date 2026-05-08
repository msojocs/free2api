import api from './axios'

export interface Account {
  id: number
  email: string
  type: string
  status: string
  extra?: string
  task_batch_id: number
  created_at: string
}

export function getAccounts(params?: { type?: string; status?: string }) {
  return api.get<{ accounts: Account[]; total: number }>('/accounts', { params })
}

export function exportAccounts(format: 'csv' | 'json', type?: string) {
  return api.get('/accounts/export', {
    params: { format, type },
    responseType: 'blob',
  })
}

export interface AccountCheckResult {
  supported: boolean
  valid: boolean
  message: string
}

export function checkAccount(id: number) {
  return api.post<AccountCheckResult>(`/accounts/${id}/check`)
}

export function deleteAccount(id: number) {
  return api.delete(`/accounts/${id}`)
}
