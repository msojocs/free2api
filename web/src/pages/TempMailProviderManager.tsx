import { useEffect, useState } from 'react'
import {
  Table,
  Button,
  Modal,
  Form,
  Input,
  Switch,
  Select,
  Space,
  Typography,
  Popconfirm,
  Tag,
  message,
} from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import {
  getTempMailProviders,
  createTempMailProvider,
  updateTempMailProvider,
  deleteTempMailProvider,
  testTempMailProvider,
  type TempMailProvider,
} from '../api/tempMailProviders'

const { Title } = Typography

// Fields required / optional per provider type
const PROVIDER_CONFIG_FIELDS: Record<string, { key: string; labelKey: string; placeholderKey: string; secret?: boolean; required?: boolean }[]> = {
  mailtm: [
    { key: 'api_url', labelKey: 'tempMail.apiUrl', placeholderKey: 'tempMail.apiUrlPlaceholder' },
  ],
  tempmail: [
    { key: 'api_url', labelKey: 'tempMail.apiUrl', placeholderKey: 'tempMail.apiUrlPlaceholder' },
  ],
  moemail: [
    { key: 'api_url', labelKey: 'tempMail.apiUrl', placeholderKey: 'tempMail.apiUrlPlaceholder' },
  ],
  cfworker: [
    { key: 'api_url', labelKey: 'tempMail.apiUrl', placeholderKey: 'tempMail.apiUrlPlaceholder', required: true },
    { key: 'admin_token', labelKey: 'tempMail.adminToken', placeholderKey: 'tempMail.adminTokenPlaceholder', secret: true, required: true },
    { key: 'domain', labelKey: 'tempMail.domain', placeholderKey: 'tempMail.domainPlaceholder', required: true },
    { key: 'fingerprint', labelKey: 'tempMail.fingerprint', placeholderKey: 'tempMail.fingerprintPlaceholder' },
  ],
  freemail: [
    { key: 'api_url', labelKey: 'tempMail.apiUrl', placeholderKey: 'tempMail.apiUrlPlaceholder', required: true },
    { key: 'admin_token', labelKey: 'tempMail.adminToken', placeholderKey: 'tempMail.adminTokenPlaceholder', secret: true },
    { key: 'username', labelKey: 'tempMail.username', placeholderKey: 'tempMail.usernamePlaceholder' },
    { key: 'password', labelKey: 'tempMail.password', placeholderKey: 'tempMail.passwordPlaceholder', secret: true },
  ],
  laoudo: [
    { key: 'auth_token', labelKey: 'tempMail.authToken', placeholderKey: 'tempMail.authTokenPlaceholder', secret: true, required: true },
    { key: 'email', labelKey: 'tempMail.email', placeholderKey: 'tempMail.emailPlaceholder', required: true },
    { key: 'account_id', labelKey: 'tempMail.accountId', placeholderKey: 'tempMail.accountIdPlaceholder', required: true },
  ],
  maliapi: [
    { key: 'api_url', labelKey: 'tempMail.apiUrl', placeholderKey: 'tempMail.apiUrlPlaceholder' },
    { key: 'api_key', labelKey: 'tempMail.apiKey', placeholderKey: 'tempMail.apiKeyPlaceholder', secret: true, required: true },
    { key: 'domain', labelKey: 'tempMail.domain', placeholderKey: 'tempMail.domainPlaceholder' },
  ],
  luckmail: [
    { key: 'api_url', labelKey: 'tempMail.apiUrl', placeholderKey: 'tempMail.apiUrlPlaceholder' },
    { key: 'api_key', labelKey: 'tempMail.apiKey', placeholderKey: 'tempMail.apiKeyPlaceholder', secret: true, required: true },
    { key: 'project_code', labelKey: 'tempMail.projectCode', placeholderKey: 'tempMail.projectCodePlaceholder', required: true },
    { key: 'email_type', labelKey: 'tempMail.emailType', placeholderKey: 'tempMail.emailTypePlaceholder' },
  ],
}

const PROVIDER_TYPE_KEYS = Object.keys(PROVIDER_CONFIG_FIELDS)

export default function TempMailProviderManager() {
  const [providers, setProviders] = useState<TempMailProvider[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<TempMailProvider | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [testingId, setTestingId] = useState<number | null>(null)
  const [selectedType, setSelectedType] = useState<string>('mailtm')
  const [form] = Form.useForm()
  const { t } = useTranslation()

  const PROVIDER_TYPE_OPTIONS = PROVIDER_TYPE_KEYS.map((k) => ({
    value: k,
    label: t(`tempMail.providerTypes.${k}` as Parameters<typeof t>[0]),
  }))

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
    setSelectedType('mailtm')
    form.resetFields()
    form.setFieldsValue({ provider_type: 'mailtm', enabled: true })
    setModalOpen(true)
  }

  function openEdit(record: TempMailProvider) {
    setEditing(record)
    setSelectedType(record.provider_type)
    // Flatten config into form fields prefixed with cfg_
    const cfgValues: Record<string, string> = {}
    for (const [k, v] of Object.entries(record.config ?? {})) {
      cfgValues[`cfg_${k}`] = v
    }
    form.setFieldsValue({
      name: record.name,
      provider_type: record.provider_type,
      enabled: record.enabled,
      description: record.description,
      ...cfgValues,
    })
    setModalOpen(true)
  }

  async function handleSubmit(values: Record<string, unknown>) {
    setSubmitting(true)
    // Extract cfg_ prefixed fields into config map
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

  const columns: ColumnsType<TempMailProvider> = [
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
          {
            !record.is_system && (
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
            )
          }
        </Space>
      ),
    },
  ]

  const configFields = PROVIDER_CONFIG_FIELDS[selectedType] ?? []

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

      <Modal
        title={editing ? t('tempMail.editTitle', { name: editing.name }) : t('tempMail.newTitle')}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={submitting}
        width={560}
        okText={editing ? t('common.save') : t('common.create')}
        cancelText={t('common.cancel')}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label={t('common.name')} rules={[{ required: true }]}>
            <Input placeholder="e.g. My DuckMail" />
          </Form.Item>

          <Form.Item name="provider_type" label={t('tempMail.providerType')} rules={[{ required: true }]}>
            <Select
              options={PROVIDER_TYPE_OPTIONS}
              placeholder={t('tempMail.selectProviderType')}
              onChange={(v: string) => {
                setSelectedType(v)
                // Clear previous config fields
                const cleared: Record<string, undefined> = {}
                for (const fields of Object.values(PROVIDER_CONFIG_FIELDS)) {
                  for (const f of fields) {
                    cleared[`cfg_${f.key}`] = undefined
                  }
                }
                form.setFieldsValue(cleared)
              }}
            />
          </Form.Item>

          {configFields.map((field) => (
            <Form.Item
              key={field.key}
              name={`cfg_${field.key}`}
              label={t(field.labelKey as Parameters<typeof t>[0])}
              rules={field.required ? [{ required: true }] : undefined}
            >
              {field.secret ? (
                <Input.Password placeholder={t(field.placeholderKey as Parameters<typeof t>[0])} />
              ) : (
                <Input placeholder={t(field.placeholderKey as Parameters<typeof t>[0])} />
              )}
            </Form.Item>
          ))}

          <Form.Item name="description" label={t('tempMail.descriptionLabel')}>
            <Input.TextArea rows={2} placeholder={t('tempMail.descriptionPlaceholder')} />
          </Form.Item>

          <Form.Item name="enabled" label={t('tempMail.enabled')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
