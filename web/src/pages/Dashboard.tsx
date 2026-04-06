import { useEffect, useState } from 'react'
import { Row, Col, Typography, Spin, message } from 'antd'
import {
  UserOutlined,
  CheckCircleOutlined,
  UnorderedListOutlined,
  GlobalOutlined,
  InboxOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import ResourceCard from '../components/ResourceCard'
import { getDashboardStats, type DashboardStats } from '../api/dashboard'

const { Title } = Typography

const POLL_INTERVAL = 30_000

export default function Dashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [loading, setLoading] = useState(true)
  const { t } = useTranslation()

  async function fetchStats() {
    try {
      const { data } = await getDashboardStats()
      setStats(data)
    } catch {
      message.error(t('dashboard.failedToLoad'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchStats()
    const timer = setInterval(fetchStats, POLL_INTERVAL)
    return () => clearInterval(timer)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <div>
      <Title level={4} style={{ marginTop: 0, marginBottom: 24 }}>
        {t('dashboard.title')}
      </Title>
      <Spin spinning={loading && stats == null}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={8}>
            <ResourceCard
              title={t('dashboard.totalAccounts')}
              value={stats?.total_accounts ?? 0}
              icon={<UserOutlined />}
            />
          </Col>
          <Col xs={24} sm={12} lg={8}>
            <ResourceCard
              title={t('dashboard.activeAccounts')}
              value={stats?.active_accounts ?? 0}
              icon={<CheckCircleOutlined />}
              color="#52c41a"
            />
          </Col>
          <Col xs={24} sm={12} lg={8}>
            <ResourceCard
              title={t('dashboard.totalTasks')}
              value={stats?.total_tasks ?? 0}
              icon={<UnorderedListOutlined />}
              color="#1677ff"
            />
          </Col>
          <Col xs={24} sm={12} lg={8}>
            <ResourceCard
              title={t('dashboard.proxiesAvailable')}
              value={stats?.proxies_available ?? 0}
              icon={<GlobalOutlined />}
              color="#722ed1"
            />
          </Col>
          <Col xs={24} sm={12} lg={8}>
            <ResourceCard
              title={t('dashboard.tempMailProviders')}
              value={stats?.temp_mail_providers ?? 0}
              icon={<InboxOutlined />}
              color="#fa8c16"
            />
          </Col>
        </Row>
      </Spin>
    </div>
  )
}
