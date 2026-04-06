import { useEffect, useState } from 'react'
import { Tabs, Form, Input, Button, Card, Space, Typography, message } from 'antd'
import { useTranslation } from 'react-i18next'
import ProxyGroupManager from '../components/ProxyGroupManager'
import { getSystemSettings, updateSystemSettings } from '../api/settings'

const { Text } = Typography

export default function SystemSettings() {
  const { t } = useTranslation()
  const [form] = Form.useForm<{ sentinel_base_url: string }>()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  async function fetchSettings() {
    setLoading(true)
    try {
      const { data } = await getSystemSettings()
      form.setFieldsValue({ sentinel_base_url: data.sentinel_base_url })
    } catch {
      message.error(t('settings.failedToLoad'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchSettings()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  async function handleSave(values: { sentinel_base_url: string }) {
    setSaving(true)
    try {
      const { data } = await updateSystemSettings(values)
      form.setFieldsValue({ sentinel_base_url: data.sentinel_base_url })
      message.success(t('settings.saved'))
    } catch {
      message.error(t('settings.failedToSave'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Tabs
      items={[
        {
          key: 'runtime',
          label: t('settings.runtimeTab'),
          children: (
            <Card loading={loading}>
              <Form form={form} layout="vertical" onFinish={handleSave}>
                <Form.Item
                  label={t('settings.sentinelBaseUrl')}
                  name="sentinel_base_url"
                  rules={[{ required: true, message: t('settings.sentinelBaseUrlRequired') }]}
                >
                  <Input placeholder={t('settings.sentinelBaseUrlPlaceholder')} />
                </Form.Item>
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  <Text type="secondary">{t('settings.sentinelBaseUrlHelp')}</Text>
                  <Button type="primary" htmlType="submit" loading={saving}>
                    {t('common.save')}
                  </Button>
                </Space>
              </Form>
            </Card>
          ),
        },
        {
          key: 'proxy-groups',
          label: t('proxyGroups.title'),
          children: <ProxyGroupManager />,
        },
      ]}
    />
  )
}