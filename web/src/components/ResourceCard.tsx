import { Card, Statistic } from 'antd'
import type { ReactNode } from 'react'

interface ResourceCardProps {
  title: string
  value: number | string
  icon?: ReactNode
  color?: string
}

export default function ResourceCard({ title, value, icon, color }: ResourceCardProps) {
  return (
    <Card>
      <Statistic
        title={title}
        value={value}
        prefix={icon}
        valueStyle={color ? { color } : undefined}
      />
    </Card>
  )
}
