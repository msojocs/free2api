import { useEffect, useState } from 'react'
import { Modal, Form, Input, Switch, Select } from 'antd'
import { useTranslation } from 'react-i18next'
import type { TempMailProvider } from '../api/tempMailProviders'

// Fields required / optional per provider type
const PROVIDER_CONFIG_FIELDS: Record<
  string,
  { key: string; labelKey: string; placeholderKey: string; secret?: boolean; required?: boolean }[]
> = {
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
  linshiyouxiang: [
    { key: 'api_url', labelKey: 'tempMail.apiUrl', placeholderKey: 'tempMail.apiUrlPlaceholder' },
  ],
  tempmailorg: [
    { key: 'api_url', labelKey: 'tempMail.apiUrl', placeholderKey: 'tempMail.apiUrlPlaceholder' },
  ],
  secemail: [
    { key: 'api_url', labelKey: 'tempMail.apiUrl', placeholderKey: 'tempMail.apiUrlPlaceholder' },
  ],
}

const PROVIDER_TYPE_KEYS = Object.keys(PROVIDER_CONFIG_FIELDS)

interface Props {
  open: boolean
  editing: TempMailProvider | null
  submitting: boolean
  onOk: (values: Record<string, unknown>) => void
  onCancel: () => void
}

export default function TempMailProviderFormModal({ open, editing, submitting, onOk, onCancel }: Props) {
  const [form] = Form.useForm()
  const [selectedType, setSelectedType] = useState<string>('mailtm')
  const { t } = useTranslation()

  const PROVIDER_TYPE_OPTIONS = PROVIDER_TYPE_KEYS.map((k) => ({
    value: k,
    label: t(`tempMail.providerTypes.${k}` as Parameters<typeof t>[0]),
  }))

  useEffect(() => {
    if (!open) return
    if (editing) {
      setSelectedType(editing.provider_type)
      const cfgValues: Record<string, string> = {}
      for (const [k, v] of Object.entries(editing.config ?? {})) {
        cfgValues[`cfg_${k}`] = v
      }
      form.setFieldsValue({
        name: editing.name,
        provider_type: editing.provider_type,
        enabled: editing.enabled,
        description: editing.description,
        ...cfgValues,
      })
    } else {
      setSelectedType('mailtm')
      form.resetFields()
      form.setFieldsValue({ provider_type: 'mailtm', enabled: true })
    }
  }, [open, editing, form])

  function handleTypeChange(v: string) {
    setSelectedType(v)
    const cleared: Record<string, undefined> = {}
    for (const fields of Object.values(PROVIDER_CONFIG_FIELDS)) {
      for (const f of fields) {
        cleared[`cfg_${f.key}`] = undefined
      }
    }
    form.setFieldsValue(cleared)
  }

  const configFields = PROVIDER_CONFIG_FIELDS[selectedType] ?? []

  return (
    <Modal
      title={editing ? t('tempMail.editTitle', { name: editing.name }) : t('tempMail.newTitle')}
      open={open}
      onCancel={onCancel}
      onOk={() => form.submit()}
      confirmLoading={submitting}
      width={560}
      okText={editing ? t('common.save') : t('common.create')}
      cancelText={t('common.cancel')}
    >
      <Form form={form} layout="vertical" onFinish={onOk}>
        <Form.Item name="name" label={t('common.name')} rules={[{ required: true }]}>
          <Input placeholder="e.g. My DuckMail" />
        </Form.Item>

        <Form.Item name="provider_type" label={t('tempMail.providerType')} rules={[{ required: true }]}>
          <Select
            options={PROVIDER_TYPE_OPTIONS}
            placeholder={t('tempMail.selectProviderType')}
            onChange={handleTypeChange}
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
  )
}
