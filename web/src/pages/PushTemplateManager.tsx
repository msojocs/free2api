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
  Tooltip,
} from 'antd'
import {
  PlusOutlined,
  CopyOutlined,
  ExperimentOutlined,
  LockOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import {
  getPushTemplates,
  createPushTemplate,
  updatePushTemplate,
  deletePushTemplate,
  copyPushTemplate,
  testPushTemplate,
  type PushTemplate,
} from '../api/pushTemplates'

const { Title, Text } = Typography
const { TextArea } = Input

const METHOD_OPTIONS = ['GET', 'POST', 'PUT']

export default function PushTemplateManager() {
  const [templates, setTemplates] = useState<PushTemplate[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<PushTemplate | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [testingId, setTestingId] = useState<number | null>(null)
  const [copyingId, setCopyingId] = useState<number | null>(null)
  const [testResultModal, setTestResultModal] = useState<{ ok: boolean; status_code: number; response: string } | null>(null)
  const [form] = Form.useForm()
  const { t } = useTranslation()

  const ACCOUNT_TYPE_OPTIONS = [
    { value: '', label: t('pushTemplates.allTypes') },
    { value: 'chatgpt', label: 'ChatGPT' },
    { value: 'cursor', label: 'Cursor' },
    { value: 'trae', label: 'Trae' },
    { value: 'grok', label: 'Grok' },
    { value: 'tavily', label: 'Tavily' },
    { value: 'kiro', label: 'Kiro' },
  ]

  const VARIABLE_HINTS = [
    { key: '{{.email}}', desc: t('pushTemplates.varEmail') },
    { key: '{{.password}}', desc: t('pushTemplates.varPassword') },
    { key: '{{.type}}', desc: t('pushTemplates.varType') },
    { key: '{{.status}}', desc: t('pushTemplates.varStatus') },
    { key: '{{.extra}}', desc: t('pushTemplates.varExtra') },
    { key: '{{.task_id}}', desc: t('pushTemplates.varTaskId') },
    { key: '{{.created_at}}', desc: t('pushTemplates.varCreatedAt') },
  ]

  async function fetchTemplates() {
    setLoading(true)
    try {
      const { data } = await getPushTemplates({ page: 1, limit: 100 })
      setTemplates(data.push_templates ?? [])
      setTotal(data.total)
    } catch {
      message.error(t('pushTemplates.failedToLoad'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchTemplates()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function openCreate() {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ method: 'POST', enabled: true, account_type: '' })
    setModalOpen(true)
  }

  function openEdit(record: PushTemplate) {
    setEditing(record)
    form.setFieldsValue({
      name: record.name,
      url: record.url,
      method: record.method,
      headers: record.headers,
      body_template: record.body_template,
      description: record.description,
      enabled: record.enabled,
      account_type: record.account_type ?? '',
    })
    setModalOpen(true)
  }

  async function handleSubmit(values: {
    name: string
    url: string
    method: string
    headers?: string
    body_template?: string
    description?: string
    enabled: boolean
    account_type?: string
  }) {
    setSubmitting(true)
    try {
      if (editing) {
        await updatePushTemplate(editing.id, {
          name: values.name,
          url: values.url,
          method: values.method,
          headers: values.headers ?? '',
          body_template: values.body_template ?? '',
          description: values.description ?? '',
          enabled: values.enabled,
          account_type: values.account_type ?? '',
        })
        message.success(t('pushTemplates.updated'))
      } else {
        await createPushTemplate({
          name: values.name,
          url: values.url,
          method: values.method,
          headers: values.headers ?? '',
          body_template: values.body_template ?? '',
          description: values.description ?? '',
          account_type: values.account_type ?? '',
        })
        message.success(t('pushTemplates.created'))
      }
      setModalOpen(false)
      fetchTemplates()
    } catch {
      message.error(t('pushTemplates.failedToSave'))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDelete(id: number) {
    try {
      await deletePushTemplate(id)
      message.success(t('pushTemplates.deleted'))
      fetchTemplates()
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      message.error(msg ?? t('pushTemplates.failedToDelete'))
    }
  }

  async function handleCopy(id: number) {
    setCopyingId(id)
    try {
      await copyPushTemplate(id)
      message.success(t('pushTemplates.copied'))
      fetchTemplates()
    } catch {
      message.error(t('pushTemplates.failedToCopy'))
    } finally {
      setCopyingId(null)
    }
  }

  async function handleTest(id: number) {
    setTestingId(id)
    try {
      const { data } = await testPushTemplate(id)
      setTestResultModal(data)
    } catch {
      message.error(t('pushTemplates.testPushFailed'))
    } finally {
      setTestingId(null)
    }
  }

  const columns: ColumnsType<PushTemplate> = [
    {
      title: t('common.name'),
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: PushTemplate) => (
        <Space>
          {name}
          {record.is_system && (
            <Tooltip title={t('pushTemplates.systemTemplate')}>
              <LockOutlined style={{ color: '#faad14' }} />
            </Tooltip>
          )}
        </Space>
      ),
    },
    {
      title: t('pushTemplates.accountType'),
      dataIndex: 'account_type',
      key: 'account_type',
      render: (v: string) =>
        v ? (
          <Tag color="blue">{v}</Tag>
        ) : (
          <Tag color="default">{t('pushTemplates.allTypes')}</Tag>
        ),
    },
    { title: t('pushTemplates.method'), dataIndex: 'method', key: 'method', render: (m: string) => <Tag color="purple">{m}</Tag> },
    {
      title: t('pushTemplates.url'),
      dataIndex: 'url',
      key: 'url',
      ellipsis: true,
      render: (url: string) => (
        <Text code style={{ fontSize: 12 }}>
          {url}
        </Text>
      ),
    },
    {
      title: t('pushTemplates.enabled'),
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean) =>
        enabled ? <Tag color="green">{t('pushTemplates.on')}</Tag> : <Tag color="default">{t('pushTemplates.off')}</Tag>,
    },
    {
      title: t('common.actions'),
      key: 'actions',
      render: (_: unknown, record: PushTemplate) => (
        <Space size="small" wrap>
          <Button size="small" onClick={() => openEdit(record)}>
            {t('common.edit')}
          </Button>
          <Button
            size="small"
            icon={<CopyOutlined />}
            loading={copyingId === record.id}
            onClick={() => handleCopy(record.id)}
          >
            {t('common.copy')}
          </Button>
          <Button
            size="small"
            icon={<ExperimentOutlined />}
            loading={testingId === record.id}
            onClick={() => handleTest(record.id)}
          >
            {t('common.test')}
          </Button>
          {!record.is_system && (
            <Popconfirm
              title={t('common.delete') + '?'}
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
          {t('pushTemplates.title')} ({total})
        </Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          {t('pushTemplates.addTemplate')}
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={templates}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 20 }}
      />

      {/* Create / Edit modal */}
      <Modal
        title={editing ? t('pushTemplates.editTitle', { name: editing.name }) : t('pushTemplates.newTitle')}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={submitting}
        width={680}
        okText={editing ? t('common.save') : t('common.create')}
        cancelText={t('common.cancel')}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label={t('common.name')} rules={[{ required: true }]}>
            <Input placeholder="e.g. My Webhook" />
          </Form.Item>

          <Form.Item
            name="account_type"
            label={t('pushTemplates.accountType')}
            tooltip={t('pushTemplates.accountTypeTooltip')}
          >
            <Select options={ACCOUNT_TYPE_OPTIONS} />
          </Form.Item>

          <Space style={{ width: '100%' }} size="middle">
            <Form.Item name="method" label={t('pushTemplates.method')} rules={[{ required: true }]} style={{ width: 120 }}>
              <Select>
                {METHOD_OPTIONS.map((m) => (
                  <Select.Option key={m} value={m}>
                    {m}
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item
              name="url"
              label={t('pushTemplates.url')}
              rules={[{ required: true }]}
              style={{ flex: 1 }}
            >
              <Input placeholder={t('pushTemplates.urlPlaceholder')} />
            </Form.Item>
          </Space>

          <Form.Item
            name="headers"
            label={t('pushTemplates.headers')}
            tooltip={t('pushTemplates.headersTooltip')}
          >
            <TextArea rows={2} placeholder={t('pushTemplates.headersPlaceholder')} />
          </Form.Item>

          <Form.Item
            name="body_template"
            label={
              <Space>
                {t('pushTemplates.bodyTemplate')}
                <Tooltip
                  title={
                    <div>
                      <div style={{ marginBottom: 4, fontWeight: 'bold' }}>{t('pushTemplates.variablesHint')}</div>
                      {VARIABLE_HINTS.map((v) => (
                        <div key={v.key}>
                          <code>{v.key}</code> — {v.desc}
                        </div>
                      ))}
                    </div>
                  }
                >
                  <Tag color="blue" style={{ cursor: 'help' }}>
                    {t('pushTemplates.variables')}
                  </Tag>
                </Tooltip>
              </Space>
            }
          >
            <TextArea
              rows={5}
              placeholder={t('pushTemplates.bodyTemplatePlaceholder')}
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item name="description" label={t('pushTemplates.descriptionLabel')}>
            <Input.TextArea rows={2} placeholder={t('pushTemplates.descriptionPlaceholder')} />
          </Form.Item>

          {editing && (
            <Form.Item name="enabled" label={t('pushTemplates.enabled')} valuePropName="checked">
              <Switch />
            </Form.Item>
          )}
        </Form>
      </Modal>

      {/* Test result modal */}
      <Modal
        title={t('pushTemplates.testResultTitle')}
        open={testResultModal !== null}
        onCancel={() => setTestResultModal(null)}
        footer={[
          <Button key="close" onClick={() => setTestResultModal(null)}>
            {t('common.close')}
          </Button>,
        ]}
      >
        {testResultModal && (
          <div>
            <p>
              <b>{t('pushTemplates.testStatus')}</b>{' '}
              <Tag color={testResultModal.ok ? 'green' : 'red'}>
                {testResultModal.ok ? t('pushTemplates.testSuccess') : t('pushTemplates.testFailed')}
              </Tag>
              <Tag>{testResultModal.status_code || 'N/A'}</Tag>
            </p>
            <p>
              <b>{t('pushTemplates.testResponse')}</b>
            </p>
            <pre
              style={{
                background: '#f5f5f5',
                padding: 12,
                borderRadius: 4,
                maxHeight: 300,
                overflow: 'auto',
                fontSize: 12,
              }}
            >
              {testResultModal.response || t('pushTemplates.testEmpty')}
            </pre>
          </div>
        )}
      </Modal>
    </div>
  )
}
