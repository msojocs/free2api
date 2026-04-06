import api from './axios'
import type { ProxyGroup } from './proxyGroups'

export interface Proxy {
  id: number
  host: string
  port: string
  proxy_group_id?: number
  proxy_group?: ProxyGroup
  username: string
  password: string
  protocol: string
  status: string
}

export interface CreateProxyPayload {
  host: string
  port: string
  proxy_group_id?: number
  username?: string
  password?: string
  protocol?: string
}

export function getProxies(params?: { page?: number; limit?: number }) {
  return api.get<{ proxies: Proxy[]; total: number }>('/proxies', { params })
}

export function createProxy(payload: CreateProxyPayload) {
  return api.post<{ proxy: Proxy }>('/proxies', payload)
}

export function updateProxy(id: number, payload: CreateProxyPayload) {
  return api.put<{ proxy: Proxy }>(`/proxies/${id}`, payload)
}

export function deleteProxy(id: number) {
  return api.delete(`/proxies/${id}`)
}

export function testProxy(id: number) {
  return api.post<{ ok: boolean }>(`/proxies/${id}/test`)
}
