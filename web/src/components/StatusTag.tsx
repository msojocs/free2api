import { Tag } from 'antd'

const colorMap: Record<string, string> = {
  active: 'green',
  running: 'blue',
  pending: 'gold',
  paused: 'orange',
  completed: 'cyan',
  failed: 'red',
  banned: 'red',
}

interface StatusTagProps {
  status: string
}

export default function StatusTag({ status }: StatusTagProps) {
  return <Tag color={colorMap[status] ?? 'default'}>{status.toUpperCase()}</Tag>
}
