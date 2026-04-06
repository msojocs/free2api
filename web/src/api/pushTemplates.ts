import api from './axios'

export interface PushTemplate {
  id: number
  name: string
  enabled: boolean
  url: string
  method: string
  headers: string
  body_template: string
  description: string
  is_system: boolean
  account_type: string
  created_at: string
  updated_at: string
}

export interface CreatePushTemplatePayload {
  name: string
  url: string
  method: string
  headers?: string
  body_template?: string
  description?: string
  account_type?: string
}

export interface UpdatePushTemplatePayload extends CreatePushTemplatePayload {
  enabled: boolean
}

export function getPushTemplates(params?: { page?: number; limit?: number }) {
  return api.get<{ push_templates: PushTemplate[]; total: number }>('/push-templates', { params })
}

export function getTemplatesForUpload(accountType: string) {
  return api.get<{ push_templates: PushTemplate[] }>('/push-templates/for-upload', {
    params: { type: accountType },
  })
}

export function createPushTemplate(payload: CreatePushTemplatePayload) {
  return api.post<{ push_template: PushTemplate }>('/push-templates', payload)
}

export function updatePushTemplate(id: number, payload: UpdatePushTemplatePayload) {
  return api.put<{ push_template: PushTemplate }>(`/push-templates/${id}`, payload)
}

export function deletePushTemplate(id: number) {
  return api.delete(`/push-templates/${id}`)
}

export function copyPushTemplate(id: number) {
  return api.post<{ push_template: PushTemplate }>(`/push-templates/${id}/copy`)
}

export function testPushTemplate(id: number) {
  return api.post<{ ok: boolean; status_code: number; response: string }>(`/push-templates/${id}/test`)
}

export function pushAccountToTemplate(templateId: number, accountId: number) {
  return api.post<{ ok: boolean; status_code: number; response: string }>(
    `/push-templates/${templateId}/push-account`,
    { account_id: accountId },
  )
}
