import { useEffect, useState } from 'react'
import {
  Table,
  Button,
  Space,
  Typography,
  Popconfirm,
  Tag,
  message,
} from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import type { TableProps } from 'antd'
import { useTranslation } from 'react-i18next'
import {
  getTempMailProviders,
  createTempMailProvider,
  updateTempMailProvider,
  deleteTempMailProvider,
  testTempMailProvider,
  type TempMailProvider,
} from '../api/tempMailProviders'
import TempMailProviderFormModal from '../components/TempMailProviderFormModal'

const { Title } = Typography

export default function TempMailProviderManager() {
  const [providers, setProviders] = useState<TempMailProvider[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<TempMailProvider | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [testingId, setTestingId] = useState<number | null>(null)
  const { t } = useTranslation()

  async function fetchProviders() {
    setLoading(true)
    try {
      const { data } = await getTempMailProviders()
      setProviders(data.providers ?? [])
    } catch {
      message.error(t('tempMail.failedToLoad'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchProviders()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function openCreate() {
    setEditing(null)
    setModalOpen(true)
  }

  function openEdit(record: TempMailProvider) {
    setEditing(record)
    setModalOpen(true)
  }

  async function handleSubmit(values: Record<string, unknown>) {
    setSubmitting(true)
    const config: Record<string, string> = {}
    for (const [k, v] of Object.entries(values)) {
      if (k.startsWith('cfg_') && typeof v === 'string' && v.trim() !== '') {
        config[k.slice(4)] = v.trim()
      }
    }
    const payload = {
      name: values.name as string,
      provider_type: values.provider_type as string,
      config,
      enabled: values.enabled as boolean,
      description: (values.description as string | undefined) ?? '',
    }
    try {
      if (editing) {
        await updateTempMailProvider(editing.id, payload)
        message.success(t('tempMail.updated'))
      } else {
        await createTempMailProvider(payload)
        message.success(t('tempMail.created'))
      }
      setModalOpen(false)
      fetchProviders()
    } catch {
      message.error(t('tempMail.failedToSave'))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDelete(id: number) {
    try {
      await deleteTempMailProvider(id)
      message.success(t('tempMail.deleted'))
      fetchProviders()
    } catch {
      message.error(t('tempMail.failedToDelete'))
    }
  }

  async function handleTest(id: number) {
    setTestingId(id)
    try {
      const { data } = await testTempMailProvider(id)
      if (data.ok && data.email) {
        message.success(t('tempMail.testSuccess', { email: data.email }))
      } else {
        message.error(t('tempMail.testFailed', { error: data.error ?? 'unknown' }))
      }
    } catch {
      message.error(t('tempMail.testFailed', { error: 'request failed' }))
    } finally {
      setTestingId(null)
    }
  }

  const columns: TableProps<TempMailProvider>['columns'] = [
    { title: t('common.name'), dataIndex: 'name', key: 'name' },
    {
      title: t('tempMail.providerType'),
      dataIndex: 'provider_type',
      key: 'provider_type',
      render: (v: string) => (
        <Tag color="blue">
          {t(`tempMail.providerTypes.${v}` as Parameters<typeof t>[0], { defaultValue: v })}
        </Tag>
      ),
    },
    {
      title: t('tempMail.enabled'),
      dataIndex: 'enabled',
      key: 'enabled',
      render: (v: boolean) =>
        v ? <Tag color="green">{t('tempMail.on')}</Tag> : <Tag color="default">{t('tempMail.off')}</Tag>,
    },
    { title: t('common.description'), dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: t('common.actions'),
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Button size="small" onClick={() => openEdit(record)}>
            {t('common.edit')}
          </Button>
          <Button
            size="small"
            loading={testingId === record.id}
            onClick={() => handleTest(record.id)}
          >
            {t('common.test')}
          </Button>
          {!record.is_system && (
            <Popconfirm
              title={t('tempMail.deleteConfirm')}
              onConfirm={() => handleDelete(record.id)}
              okText={t('common.yes')}
              cancelText={t('common.no')}
            >
              <Button size="small" danger>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>
          {t('tempMail.title')}
        </Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          {t('tempMail.addProvider')}
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={providers}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 20 }}
      />

      <TempMailProviderFormModal
        open={modalOpen}
        editing={editing}
        submitting={submitting}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
      />
    </div>
  )
}

