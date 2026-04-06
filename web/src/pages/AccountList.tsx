import { useEffect, useState, useCallback } from 'react'
import {
  Table,
  Button,
  Space,
  Select,
  Typography,
  message,
  Dropdown,
} from 'antd'
import { DownloadOutlined, ReloadOutlined, UploadOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { MenuProps } from 'antd'
import { useTranslation } from 'react-i18next'
import StatusTag from '../components/StatusTag'
import { getAccounts, exportAccounts, type Account } from '../api/accounts'
import { getTemplatesForUpload, pushAccountToTemplate, type PushTemplate } from '../api/pushTemplates'

const { Title } = Typography

export default function AccountList() {
  const [accounts, setAccounts] = useState<Account[]>([])
  const [loading, setLoading] = useState(false)
  const [accountType, setAccountType] = useState('')
  const [status, setStatus] = useState('')
  const [templatesByType, setTemplatesByType] = useState<Record<string, PushTemplate[]>>({})
  const [pushingKey, setPushingKey] = useState<string | null>(null)
  const { t } = useTranslation()

  const TYPE_OPTIONS = [
    { value: '', label: t('accounts.allTypes') },
    { value: 'chatgpt', label: 'ChatGPT' },
    { value: 'cursor', label: 'Cursor' },
  ]

  const STATUS_OPTIONS = [
    { value: '', label: t('accounts.allStatuses') },
    { value: 'active', label: t('accounts.active') },
    { value: 'banned', label: t('accounts.banned') },
    { value: 'pending', label: t('accounts.pending') },
  ]

  async function fetchTemplatesForType(type: string) {
    if (type in templatesByType) return
    try {
      const { data } = await getTemplatesForUpload(type)
      setTemplatesByType((prev) => ({ ...prev, [type]: data.push_templates ?? [] }))
    } catch {
      // ignore
    }
  }

  async function fetchAccounts() {
    setLoading(true)
    try {
      const { data } = await getAccounts({
        type: accountType || undefined,
        status: status || undefined,
      })
      setAccounts(data.accounts ?? [])
    } catch {
      message.error(t('accounts.failedToLoad'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchAccounts()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [accountType, status])

  async function handleExport(format: 'csv' | 'json') {
    try {
      const { data } = await exportAccounts(format)
      const url = URL.createObjectURL(data as Blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `accounts.${format}`
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      message.error(t('accounts.exportFailed'))
    }
  }

  const handlePushToTemplate = useCallback(
    async (account: Account, templateId: number, templateName: string) => {
      const key = `${account.id}-${templateId}`
      setPushingKey(key)
      try {
        const { data } = await pushAccountToTemplate(templateId, account.id)
        if (data.ok) {
          message.success(t('accounts.pushedSuccess', { name: templateName, code: data.status_code }))
        } else {
          message.error(t('accounts.pushedFailed', { name: templateName, code: data.status_code, response: data.response }))
        }
      } catch {
        message.error(t('accounts.pushFailed', { name: templateName }))
      } finally {
        setPushingKey(null)
      }
    },
    [t],
  )

  const columns: ColumnsType<Account> = [
    { title: t('common.email'), dataIndex: 'email', key: 'email' },
    { title: t('common.type'), dataIndex: 'type', key: 'type' },
    {
      title: t('common.status'),
      dataIndex: 'status',
      key: 'status',
      render: (s) => <StatusTag status={s} />,
    },
    {
      title: t('common.created'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (v) => new Date(v).toLocaleString(),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      render: (_, record) => {
        const templates = templatesByType[record.type] ?? []
        const menuItems: MenuProps['items'] = templates.map((tmpl) => ({
          key: String(tmpl.id),
          label: t('accounts.uploadTo', { name: tmpl.name }),
          disabled: pushingKey === `${record.id}-${tmpl.id}`,
          onClick: () => handlePushToTemplate(record, tmpl.id, tmpl.name),
        }))

        return (
          <Dropdown
            menu={{
              items: menuItems.length > 0
                ? menuItems
                : [{ key: 'none', label: t('accounts.noTemplates'), disabled: true }],
            }}
            trigger={['click']}
            onOpenChange={(open) => {
              if (open) fetchTemplatesForType(record.type)
            }}
          >
            <Button size="small" icon={<UploadOutlined />}>
              {t('accounts.upload')}
            </Button>
          </Dropdown>
        )
      },
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>
          {t('accounts.title')}
        </Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={fetchAccounts} loading={loading}>
            {t('accounts.refresh')}
          </Button>
          <Button icon={<DownloadOutlined />} onClick={() => handleExport('csv')}>
            {t('accounts.exportCsv')}
          </Button>
          <Button icon={<DownloadOutlined />} onClick={() => handleExport('json')}>
            {t('accounts.exportJson')}
          </Button>
        </Space>
      </div>

      <Space style={{ marginBottom: 16 }}>
        <Select
          value={accountType}
          onChange={setAccountType}
          options={TYPE_OPTIONS}
          style={{ width: 160 }}
        />
        <Select
          value={status}
          onChange={setStatus}
          options={STATUS_OPTIONS}
          style={{ width: 160 }}
        />
      </Space>

      <Table
        columns={columns}
        dataSource={accounts}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 20 }}
      />
    </div>
  )
}

