import api from './axios'

export interface LoginResponse {
  token: string
  user: { id: number; username: string }
}

export function login(username: string, password: string) {
  return api.post<LoginResponse>('/auth/login', { username, password })
}
