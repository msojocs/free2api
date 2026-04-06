import api from './axios'

export interface Account {
  id: number
  email: string
  type: string
  status: string
  task_batch_id: number
  created_at: string
}

export function getAccounts(params?: { type?: string; status?: string }) {
  return api.get<{ accounts: Account[]; total: number }>('/accounts', { params })
}

export function exportAccounts(format: 'csv' | 'json') {
  return api.get('/accounts/export', {
    params: { format },
    responseType: 'blob',
  })
}
