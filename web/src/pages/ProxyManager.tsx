import { useEffect, useState } from 'react'
import {
  App as AntdApp,
  Table,
  Button,
  Space,
  Typography,
  Popconfirm,
  Tag,
  type TableProps,
} from 'antd'
import { EditOutlined, PlusOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { getProxies, createProxy, updateProxy, deleteProxy, testProxy, type CreateProxyPayload, type Proxy } from '../api/proxies'
import { getProxyGroups, type ProxyGroup } from '../api/proxyGroups'
import ProxyFormModal, { type ProxyFormValues } from '../components/ProxyFormModal'

const { Title } = Typography

export default function ProxyManager() {
  const [proxies, setProxies] = useState<Proxy[]>([])
  const [groups, setGroups] = useState<ProxyGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingProxy, setEditingProxy] = useState<Proxy | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [testingId, setTestingId] = useState<number | null>(null)
  const { t } = useTranslation()
  const { message } = AntdApp.useApp()

  async function fetchProxies() {
    setLoading(true)
    try {
      const { data } = await getProxies()
      setProxies(data.proxies ?? [])
    } catch {
      message.error(t('proxies.failedToLoad'))
    } finally {
      setLoading(false)
    }
  }

  async function fetchGroups() {
    try {
      const { data } = await getProxyGroups()
      setGroups(data.groups ?? [])
    } catch {
      setGroups([])
    }
  }

  useEffect(() => {
    fetchProxies()
    fetchGroups()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function normalizePayload(values: ProxyFormValues): CreateProxyPayload {
    return {
      host: values.host.trim(),
      port: values.port.trim(),
      proxy_group_id: values.proxy_group_id || undefined,
      protocol: values.protocol,
      username: values.username?.trim() || undefined,
      password: values.password?.trim() || undefined,
    }
  }

  function openCreate() {
    setEditingProxy(null)
    setModalOpen(true)
  }

  function openEdit(proxy: Proxy) {
    setEditingProxy(proxy)
    setModalOpen(true)
  }

  function closeModal() {
    setModalOpen(false)
    setEditingProxy(null)
  }

  async function handleSubmit(values: ProxyFormValues) {
    setSubmitting(true)
    const payload = normalizePayload(values)

    try {
      if (editingProxy) {
        await updateProxy(editingProxy.id, payload)
        message.success(t('proxies.updated'))
      } else {
        await createProxy(payload)
        message.success(t('proxies.added'))
      }

      closeModal()
      await fetchProxies()
      await fetchGroups()
    } catch {
      message.error(editingProxy ? t('proxies.failedToUpdate') : t('proxies.failedToAdd'))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteProxy(id)
      message.success(t('proxies.deleted'))
      await fetchProxies()
    } catch {
      message.error(t('proxies.failedToDelete'))
    }
  }

  async function handleTest(id: number) {
    setTestingId(id)
    try {
      console.info('Testing proxy', id)
      await testProxy(id)
      console.info('Proxy is reachable')
      message.success(t('proxies.reachable'))
    } catch {
      message.error(t('proxies.testFailed'))
    } finally {
      setTestingId(null)
    }
  }

  const columns: TableProps<Proxy>['columns'] = [
    { title: t('proxies.host'), dataIndex: 'host', key: 'host' },
    { title: t('proxies.port'), dataIndex: 'port', key: 'port' },
    {
      title: t('proxies.group'),
      dataIndex: 'proxy_group',
      key: 'group',
      render: (_, record) => record.proxy_group?.name || <Tag>{t('proxies.noGroup')}</Tag>,
    },
    { title: t('proxies.protocol'), dataIndex: 'protocol', key: 'protocol' },
    { title: t('proxies.username'), dataIndex: 'username', key: 'username' },
    {
      title: t('common.status'),
      dataIndex: 'status',
      key: 'status',
      render: (v) => (
        v === 'active'
          ? <Tag color="green">{t('proxies.statusActive')}</Tag>
          : <Tag color="red">{t('proxies.statusInactive')}</Tag>
      ),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => openEdit(record)}
          >
            {t('common.edit')}
          </Button>
          <Button
            size="small"
            loading={testingId === record.id}
            onClick={() => handleTest(record.id)}
          >
            {t('common.test')}
          </Button>
          <Popconfirm
            title={t('proxies.deleteConfirm')}
            onConfirm={() => handleDelete(record.id)}
            okText={t('common.yes')}
            cancelText={t('common.no')}
          >
            <Button size="small" danger>
              {t('common.delete')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>
          {t('proxies.title')}
        </Title>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={openCreate}
        >
          {t('proxies.addProxy')}
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={proxies}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 20 }}
      />

      <ProxyFormModal
        open={modalOpen}
        proxy={editingProxy}
        groups={groups}
        onCancel={closeModal}
        onSubmit={handleSubmit}
        submitting={submitting}
      />
    </div>
  )
}

