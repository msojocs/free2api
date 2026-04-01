import { useEffect, useRef, useState } from 'react'
import { Modal, Progress, List, Typography, Badge } from 'antd'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '../store/auth'

const { Text } = Typography

interface TaskProgressProps {
  taskId: number | null
  open: boolean
  onClose: () => void
}

interface ProgressEvent {
  progress: number
  message: string
  status?: string
}

export default function TaskProgress({ taskId, open, onClose }: TaskProgressProps) {
  const [percent, setPercent] = useState(0)
  const [logs, setLogs] = useState<ProgressEvent[]>([])
  const token = useAuthStore((s) => s.token)
  const abortRef = useRef<AbortController | null>(null)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const { t } = useTranslation()

  useEffect(() => {
    if (!open || taskId == null) return

    setPercent(0)
    setLogs([])

    const controller = new AbortController()
    abortRef.current = controller

    void (async () => {
      try {
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
                const data = JSON.parse(dataLines.join('\n')) as ProgressEvent
                setPercent(data.progress ?? 0)
                setLogs((prev) => [...prev, data])
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
  }, [open, taskId, token])

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  function handleClose() {
    abortRef.current?.abort()
    abortRef.current = null
    onClose()
  }

  const statusColor = percent >= 100 ? 'success' : 'active'

  return (
    <Modal
      title={t('tasks.taskProgressTitle', { id: taskId })}
      open={open}
      onCancel={handleClose}
      footer={null}
      width={600}
    >
      <Progress percent={percent} status={statusColor} style={{ marginBottom: 16 }} />
      <div
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
                color={item.status === 'error' ? 'red' : item.status === 'completed' ? 'green' : 'blue'}
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
