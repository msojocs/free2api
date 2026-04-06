import api from './axios'

export interface ProxyGroup {
  id: number
  name: string
  created_at: string
  updated_at: string
}

export interface ProxyGroupPayload {
  name: string
}

export function getProxyGroups() {
  return api.get<{ groups: ProxyGroup[]; total: number }>('/proxy-groups')
}

export function createProxyGroup(payload: ProxyGroupPayload) {
  return api.post<{ group: ProxyGroup }>('/proxy-groups', payload)
}

export function updateProxyGroup(id: number, payload: ProxyGroupPayload) {
  return api.put<{ group: ProxyGroup }>(`/proxy-groups/${id}`, payload)
}

export function deleteProxyGroup(id: number) {
  return api.delete(`/proxy-groups/${id}`)
}