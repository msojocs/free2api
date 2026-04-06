import api from './axios'

export interface DashboardStats {
  total_accounts: number
  active_accounts: number
  total_tasks: number
  proxies_available: number
  temp_mail_providers: number
}

export function getDashboardStats() {
  return api.get<DashboardStats>('/dashboard/stats')
}
