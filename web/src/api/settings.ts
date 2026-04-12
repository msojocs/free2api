import api from './axios'

export interface SystemSettings {
  id: number
  sentinel_base_url: string
  account_action_proxy_group_id?: number
  account_check_enabled: boolean
  account_check_interval_minutes: number
}

export interface UpdateSystemSettingsPayload {
  sentinel_base_url: string
  account_action_proxy_group_id?: number
  account_check_enabled: boolean
  account_check_interval_minutes: number
}

export function getSystemSettings() {
  return api.get<SystemSettings>('/settings')
}

export function updateSystemSettings(payload: UpdateSystemSettingsPayload) {
  return api.put<SystemSettings>('/settings', payload)
}
