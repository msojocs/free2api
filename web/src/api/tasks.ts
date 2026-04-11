import api from './axios'

export interface TaskBatch {
  id: number
  name: string
  type: string
  total: number
  completed: number
  failed: number
  status: 'pending' | 'running' | 'paused' | 'completed' | 'failed'
  config: Record<string, unknown>
  created_at: string
}

export interface TaskProgressLog {
  task_id: number
  progress: number
  message: string
  status: string
}

export interface CreateTaskPayload {
  type: string
  total: number
  config?: Record<string, unknown>
}

export function getTasks() {
  return api.get<{ tasks: TaskBatch[]; total: number }>('/tasks')
}

export function createTask(payload: CreateTaskPayload) {
  return api.post<{ task: TaskBatch }>('/tasks', payload)
}

export function startTask(id: number) {
  return api.post(`/tasks/${id}/start`)
}

export function pauseTask(id: number) {
  return api.post(`/tasks/${id}/pause`)
}

export function deleteTask(id: number) {
  return api.delete(`/tasks/${id}`)
}

export function getTaskLogs(id: number) {
  return api.get<{ logs: TaskProgressLog[] }>(`/tasks/${id}/logs`)
}
