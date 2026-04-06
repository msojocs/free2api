import axios from 'axios'
import { useAuthStore } from '../store/auth'

const api = axios.create({
  baseURL: '/api',
  timeout: 15000,
})

api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (res) => {
    // Blob responses (file downloads) are not JSON-wrapped — skip unwrapping.
    if (res.config.responseType === 'blob') return res
    const body = res.data as { code: number; msg: string; data: unknown }
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code !== 0) {
        return Promise.reject(new Error(body.msg || 'Request failed'))
      }
      res.data = body.data
    }
    return res
  },
  (err) => {
    if (err.response?.status === 401) {
      useAuthStore.getState().logout()
      window.location.href = '/login'
    }
    const msg =
      err.response?.data?.msg || err.response?.data?.error || err.message
    return Promise.reject(new Error(msg))
  },
)

export default api
