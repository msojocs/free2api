import api from './axios'

export interface SystemSettings {
  id: number
  sentinel_base_url: string
}

export interface UpdateSystemSettingsPayload {
  sentinel_base_url: string
}

export function getSystemSettings() {
  return api.get<SystemSettings>('/settings')
}

export function updateSystemSettings(payload: UpdateSystemSettingsPayload) {
  return api.put<SystemSettings>('/settings', payload)
}
