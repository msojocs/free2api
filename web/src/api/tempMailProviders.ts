import api from './axios'

export interface TempMailProvider {
  id: number
  name: string
  provider_type: string
  config: Record<string, string>
  enabled: boolean
  description: string
  created_at: string
  is_system: boolean
}

export interface CreateTempMailProviderPayload {
  name: string
  provider_type: string
  config: Record<string, string>
  enabled: boolean
  description?: string
}

export interface UpdateTempMailProviderPayload {
  name?: string
  provider_type?: string
  config?: Record<string, string>
  enabled: boolean
  description?: string
}

export function getTempMailProviders() {
  return api.get<{ providers: TempMailProvider[]; total: number }>('/temp-mail-providers')
}

export function createTempMailProvider(payload: CreateTempMailProviderPayload) {
  return api.post<{ provider: TempMailProvider }>('/temp-mail-providers', payload)
}

export function updateTempMailProvider(id: number, payload: UpdateTempMailProviderPayload) {
  return api.put<{ provider: TempMailProvider }>(`/temp-mail-providers/${id}`, payload)
}

export function deleteTempMailProvider(id: number) {
  return api.delete(`/temp-mail-providers/${id}`)
}

export function testTempMailProvider(id: number) {
  return api.post<{ ok: boolean; email?: string; error?: string }>(`/temp-mail-providers/${id}/test`)
}
