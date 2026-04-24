import { useEffect, useState } from 'react'
import { Tabs, Form, Input, Button, Card, Space, message, Select, Switch, InputNumber } from 'antd'
import { useTranslation } from 'react-i18next'
import ProxyGroupManager from '../components/ProxyGroupManager'
import { getSystemSettings, updateSystemSettings } from '../api/settings'
import { getProxyGroups, type ProxyGroup } from '../api/proxyGroups'

export default function SystemSettings() {
  const { t } = useTranslation()
  const [form] = Form.useForm<{
    sentinel_base_url: string
    account_action_proxy_group_id?: number
    account_check_enabled: boolean
    account_check_interval_minutes: number
  }>()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [proxyGroups, setProxyGroups] = useState<ProxyGroup[]>([])

  async function fetchSettings() {
    setLoading(true)
    try {
      const { data } = await getSystemSettings()
      form.setFieldsValue({
        sentinel_base_url: data.sentinel_base_url,
        account_action_proxy_group_id: data.account_action_proxy_group_id,
        account_check_enabled: data.account_check_enabled ?? false,
        account_check_interval_minutes: data.account_check_interval_minutes ?? 60,
      })
    } catch {
      message.error(t('settings.failedToLoad'))
    } finally {
      setLoading(false)
    }
  }

  async function fetchProxyGroups() {
    try {
      const { data } = await getProxyGroups()
      setProxyGroups(data.groups ?? [])
    } catch {
      message.error(t('proxyGroups.failedToLoad'))
    }
  }

  useEffect(() => {
    void fetchSettings()
    void fetchProxyGroups()
  }, [])

  async function handleSave(values: {
    sentinel_base_url: string
    account_action_proxy_group_id?: number
    account_check_enabled: boolean
    account_check_interval_minutes: number
  }) {
    setSaving(true)
    try {
      const { data } = await updateSystemSettings(values)
      form.setFieldsValue({
        sentinel_base_url: data.sentinel_base_url,
        account_action_proxy_group_id: data.account_action_proxy_group_id,
        account_check_enabled: data.account_check_enabled ?? false,
        account_check_interval_minutes: data.account_check_interval_minutes ?? 60,
      })
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
                  extra={t('settings.sentinelBaseUrlHelp')}
                  rules={[{ required: true, message: t('settings.sentinelBaseUrlRequired') }]}
                >
                  <Input placeholder={t('settings.sentinelBaseUrlPlaceholder')} />
                </Form.Item>
                <Form.Item
                  label={t('settings.accountActionProxyGroup')}
                  name="account_action_proxy_group_id"
                  extra={t('settings.proxyGroupHelp')}
                >
                  <Select
                    allowClear
                    options={proxyGroups.map((group) => ({ label: group.name, value: group.id }))}
                    placeholder={t('settings.proxyGroupPlaceholder')}
                  />
                </Form.Item>
                <Form.Item
                  label={t('settings.accountCheckEnabled')}
                  name="account_check_enabled"
                  valuePropName="checked"
                >
                  <Switch />
                </Form.Item>
                <Form.Item
                  label={t('settings.accountCheckIntervalMinutes')}
                  name="account_check_interval_minutes"
                  extra={t('settings.accountCheckIntervalMinutesHelp')}
                  rules={[{ required: true, type: 'number', min: 1 }]}
                >
                  <InputNumber min={1} style={{ width: '100%' }} />
                </Form.Item>
                <Space style={{ width: '100%' }}>
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
