import { useEffect } from 'react'
import { Modal, Form, Input, Select } from 'antd'
import { useTranslation } from 'react-i18next'
import type { CreateProxyPayload, Proxy } from '../api/proxies'
import type { ProxyGroup } from '../api/proxyGroups'

const PROXY_PROTOCOLS = ['http', 'https', 'socks5'] as const

export type ProxyFormValues = CreateProxyPayload

interface ProxyFormModalProps {
  open: boolean
  submitting: boolean
  proxy?: Proxy | null
  groups: ProxyGroup[]
  onCancel: () => void
  onSubmit: (values: ProxyFormValues) => void | Promise<void>
}

export default function ProxyFormModal({
  open,
  submitting,
  proxy,
  groups,
  onCancel,
  onSubmit,
}: ProxyFormModalProps) {
  const [form] = Form.useForm<ProxyFormValues>()
  const { t } = useTranslation()

  useEffect(() => {
    if (!open) {
      form.resetFields()
      return
    }

    if (proxy) {
      form.setFieldsValue({
        host: proxy.host,
        port: proxy.port,
        proxy_group_id: proxy.proxy_group_id,
        protocol: proxy.protocol || 'http',
        username: proxy.username || undefined,
        password: proxy.password || undefined,
      })
      return
    }

    form.setFieldsValue({
      host: '',
      port: '',
      proxy_group_id: undefined,
      protocol: 'http',
      username: undefined,
      password: undefined,
    })
  }, [form, open, proxy])

  return (
    <Modal
      title={proxy ? t('proxies.editTitle', { host: proxy.host }) : t('proxies.newTitle')}
      open={open}
      onCancel={onCancel}
      onOk={() => form.submit()}
      confirmLoading={submitting}
      okText={proxy ? t('common.save') : t('common.add')}
      cancelText={t('common.cancel')}
      destroyOnHidden
    >
      <Form form={form} layout="vertical" onFinish={onSubmit}>
        <Form.Item name="host" label={t('proxies.host')} rules={[{ required: true }]}>
          <Input placeholder={t('proxies.hostPlaceholder')} />
        </Form.Item>
        <Form.Item name="port" label={t('proxies.port')} rules={[{ required: true }]}>
          <Input placeholder={t('proxies.portPlaceholder')} />
        </Form.Item>
        <Form.Item name="proxy_group_id" label={t('proxies.group')} rules={[{ required: true }]}>
          <Select
            allowClear
            placeholder={t('proxies.groupPlaceholder')}
            options={groups.map((group) => ({ value: group.id, label: group.name }))}
          />
        </Form.Item>
        <Form.Item name="protocol" label={t('proxies.protocol')} rules={[{ required: true }]} initialValue="http">
          <Select>
            {PROXY_PROTOCOLS.map((protocol) => (
              <Select.Option key={protocol} value={protocol}>
                {protocol}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="username" label={t('proxies.username')}>
          <Input placeholder={t('common.optional')} autoComplete='one-time-code' />
        </Form.Item>
        <Form.Item name="password" label={t('proxies.password')}>
          <Input.Password placeholder={t('common.optional')} autoComplete='new-password' />
        </Form.Item>
      </Form>
    </Modal>
  )
}