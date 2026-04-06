import { useEffect } from 'react'
import {
  Modal,
  Form,
  Input,
  Switch,
  Select,
  Space,
  Tooltip,
  Tag,
} from 'antd'
import { useTranslation } from 'react-i18next'
import type { PushTemplate } from '../api/pushTemplates'

const { TextArea } = Input

const METHOD_OPTIONS = ['GET', 'POST', 'PUT']

const ACCOUNT_TYPE_OPTION_VALUES = ['', 'chatgpt', 'cursor', 'trae', 'grok', 'tavily', 'kiro']

interface PushTemplateFormValues {
  name: string
  url: string
  method: string
  headers?: string
  query_params?: string
  body_template?: string
  description?: string
  enabled: boolean
  account_type?: string
}

interface Props {
  open: boolean
  editing: PushTemplate | null
  submitting: boolean
  onOk: (values: PushTemplateFormValues) => void
  onCancel: () => void
}

export default function PushTemplateFormModal({ open, editing, submitting, onOk, onCancel }: Props) {
  const [form] = Form.useForm<PushTemplateFormValues>()
  const { t } = useTranslation()

  const ACCOUNT_TYPE_OPTIONS = ACCOUNT_TYPE_OPTION_VALUES.map((v) => ({
    value: v,
    label: v ? v.charAt(0).toUpperCase() + v.slice(1) : t('pushTemplates.allTypes'),
  }))

  const VARIABLE_HINTS = [
    { key: '{{.email}}', desc: t('pushTemplates.varEmail') },
    { key: '{{.password}}', desc: t('pushTemplates.varPassword') },
    { key: '{{.type}}', desc: t('pushTemplates.varType') },
    { key: '{{.status}}', desc: t('pushTemplates.varStatus') },
    { key: '{{.extra}}', desc: t('pushTemplates.varExtra') },
    { key: '{{.task_id}}', desc: t('pushTemplates.varTaskId') },
    { key: '{{.created_at}}', desc: t('pushTemplates.varCreatedAt') },
  ]

  useEffect(() => {
    if (!open) return
    if (editing) {
      form.setFieldsValue({
        name: editing.name,
        url: editing.url,
        method: editing.method,
        headers: editing.headers,
        query_params: editing.query_params,
        body_template: editing.body_template,
        description: editing.description,
        enabled: editing.enabled,
        account_type: editing.account_type ?? '',
      })
    } else {
      form.resetFields()
      form.setFieldsValue({ method: 'POST', enabled: true, account_type: '' })
    }
  }, [open, editing, form])

  const variableHintTooltip = (
    <div>
      <div style={{ marginBottom: 4, fontWeight: 'bold' }}>{t('pushTemplates.variablesHint')}</div>
      {VARIABLE_HINTS.map((v) => (
        <div key={v.key}>
          <code>{v.key}</code> — {v.desc}
        </div>
      ))}
    </div>
  )

  const varTag = (
    <Tooltip title={variableHintTooltip}>
      <Tag color="blue" style={{ cursor: 'help' }}>
        {t('pushTemplates.variables')}
      </Tag>
    </Tooltip>
  )

  return (
    <Modal
      title={editing ? t('pushTemplates.editTitle', { name: editing.name }) : t('pushTemplates.newTitle')}
      open={open}
      onCancel={onCancel}
      onOk={() => form.submit()}
      confirmLoading={submitting}
      width={680}
      okText={editing ? t('common.save') : t('common.create')}
      cancelText={t('common.cancel')}
    >
      <Form form={form} layout="vertical" onFinish={onOk}>
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
          <Form.Item
            name="method"
            label={t('pushTemplates.method')}
            rules={[{ required: true }]}
            style={{ width: 120 }}
          >
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
            style={{ flex: 1, width: 400 }}
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
          name="query_params"
          label={
            <Space size={4}>
              {t('pushTemplates.queryParams')}
              {varTag}
            </Space>
          }
          tooltip={t('pushTemplates.queryParamsTooltip')}
        >
          <TextArea
            rows={2}
            placeholder={t('pushTemplates.queryParamsPlaceholder')}
            style={{ fontFamily: 'monospace' }}
          />
        </Form.Item>

        <Form.Item
          name="body_template"
          label={
            <Space size={4}>
              {t('pushTemplates.bodyTemplate')}
              {varTag}
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
  )
}
