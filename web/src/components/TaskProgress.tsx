import { useEffect, useRef, useState } from 'react'
import { Modal, Progress, List, Typography, Badge } from 'antd'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '../store/auth'
import { getTaskLogs, type TaskProgressLog } from '../api/tasks'

const { Text } = Typography

interface TaskProgressProps {
  taskId: number | null
  taskStatus?: 'pending' | 'running' | 'paused' | 'completed' | 'failed' | null
  open: boolean
  onClose: () => void
}

type BatchStats = {
  success: number
  failed: number
  total: number
}

const batchStatsPattern = /Batch progress:\s*\d+\/(\d+)\s*completed,\s*success=(\d+),\s*failed=(\d+)/i

function parseBatchStats(message: string): BatchStats | null {
  const match = batchStatsPattern.exec(message)
  if (!match) {
    return null
  }
  const total = Number(match[1])
  const success = Number(match[2])
  const failed = Number(match[3])
  if (!Number.isFinite(total) || !Number.isFinite(success) || !Number.isFinite(failed)) {
    return null
  }
  return {
    total: Math.max(0, total),
    success: Math.max(0, success),
    failed: Math.max(0, failed),
  }
}

function clamp01(value: number): number {
  if (value < 0) {
    return 0
  }
  if (value > 1) {
    return 1
  }
  return value
}

function rgbLerp(from: number, to: number, t: number): number {
  return Math.round(from + (to - from) * t)
}

function progressColorByFailureRate(stats: BatchStats | null): string {
  if (!stats || stats.total <= 0) {
    return '#52c41a'
  }
  const ratio = clamp01(stats.failed / stats.total)
  const r = rgbLerp(82, 255, ratio)
  const g = rgbLerp(196, 77, ratio)
  const b = rgbLerp(26, 79, ratio)
  return `rgb(${r}, ${g}, ${b})`
}

const SCROLL_BOTTOM_THRESHOLD_PX = 8

export default function TaskProgress({ taskId, taskStatus, open, onClose }: TaskProgressProps) {
  const [percent, setPercent] = useState(0)
  const [logs, setLogs] = useState<TaskProgressLog[]>([])
  const [batchStats, setBatchStats] = useState<BatchStats | null>(null)
  const token = useAuthStore((s) => s.token)
  const abortRef = useRef<AbortController | null>(null)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const logsContainerRef = useRef<HTMLDivElement>(null)
  const { t } = useTranslation()

  useEffect(() => {
    if (!open || taskId == null) return

    setPercent(0)
    setLogs([])
  setBatchStats(null)

    const controller = new AbortController()
    abortRef.current = controller

    void (async () => {
      try {
        const { data: logData } = await getTaskLogs(taskId)
        const initialLogs = logData.logs ?? []
        setLogs(initialLogs)
        if (initialLogs.length > 0) {
          setPercent(initialLogs[initialLogs.length - 1].progress ?? 0)
          for (let i = initialLogs.length - 1; i >= 0; i -= 1) {
            const stats = parseBatchStats(initialLogs[i].message ?? '')
            if (stats) {
              setBatchStats(stats)
              break
            }
          }
        }

        if (taskStatus === 'completed') {
          return
        }

        const response = await fetch(`/api/tasks/${taskId}/progress`, {
          headers: token ? { Authorization: `Bearer ${token}` } : undefined,
          signal: controller.signal,
        })

        if (!response.ok || !response.body) {
          return
        }

        const reader = response.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''

        while (true) {
          const { value, done } = await reader.read()
          if (done) {
            break
          }

          buffer += decoder.decode(value, { stream: true })

          let separatorIndex = buffer.indexOf('\n\n')
          while (separatorIndex >= 0) {
            const rawEvent = buffer.slice(0, separatorIndex)
            buffer = buffer.slice(separatorIndex + 2)

            const lines = rawEvent.split(/\r?\n/)
            let eventName = 'message'
            const dataLines: string[] = []

            for (const line of lines) {
              if (line.startsWith('event:')) {
                eventName = line.slice(6).trim()
                continue
              }
              if (line.startsWith('data:')) {
                dataLines.push(line.slice(5).trim())
              }
            }

            if (eventName === 'progress' && dataLines.length > 0) {
              try {
                const data = JSON.parse(dataLines.join('\n')) as TaskProgressLog
                setPercent(data.progress ?? 0)
                const stats = parseBatchStats(data.message ?? '')
                if (stats) {
                  setBatchStats(stats)
                }
                setLogs((prev) => {
                  const last = prev[prev.length - 1]
                  if (
                    last &&
                    last.task_id === data.task_id &&
                    last.progress === data.progress &&
                    last.message === data.message &&
                    last.status === data.status
                  ) {
                    return prev
                  }
                  return [...prev, data]
                })
              } catch {
                // ignore parse errors
              }
            }

            separatorIndex = buffer.indexOf('\n\n')
          }
        }
      } catch {
        // request aborted or stream interrupted
      }
    })()

    return () => {
      controller.abort()
      abortRef.current = null
    }
  }, [open, taskId, taskStatus, token])

  useEffect(() => {
    const container = logsContainerRef.current
    if (!container) return
    const isAtBottom = container.scrollHeight - container.scrollTop - container.clientHeight <= SCROLL_BOTTOM_THRESHOLD_PX
    if (isAtBottom) {
      logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [logs])

  function handleClose() {
    abortRef.current?.abort()
    abortRef.current = null
    onClose()
  }
  const progressColor = progressColorByFailureRate(batchStats)

  return (
    <Modal
      title={t('tasks.taskProgressTitle', { id: taskId })}
      open={open}
      onCancel={handleClose}
      footer={null}
      width={600}
    >
      <Progress percent={percent} strokeColor={progressColor} style={{ marginBottom: 16 }} />
      <div
        ref={logsContainerRef}
        style={{
          height: 240,
          overflowY: 'auto',
          border: '1px solid #f0f0f0',
          borderRadius: 4,
          padding: 8,
          background: '#fafafa',
        }}
      >
        <List
          size="small"
          dataSource={logs}
          renderItem={(item, idx) => (
            <List.Item key={idx} style={{ padding: '2px 0', border: 'none' }}>
              <Badge
                color={item.status === 'failed' ? 'red' : item.status === 'completed' ? 'green' : 'blue'}
                text={<Text style={{ fontSize: 12 }}>{item.message}</Text>}
              />
            </List.Item>
          )}
        />
        <div ref={logsEndRef} />
      </div>
    </Modal>
  )
}
