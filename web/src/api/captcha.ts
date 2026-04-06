import api from './axios'

export interface CaptchaStats {
  total_solved: number
  success_rate: number
  avg_time_ms: number
}

export function getCaptchaStats() {
  return api.get<CaptchaStats>('/captcha/stats')
}
