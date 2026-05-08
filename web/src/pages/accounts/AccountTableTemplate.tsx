import { useEffect, useMemo, useState, useCallback } from 'react'
import { Table, Button, Space, Select, Typography, message, Dropdown, Popconfirm } from 'antd'
import { DownloadOutlined, ReloadOutlined, UploadOutlined, SafetyOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { MenuProps } from 'antd'
import { useTranslation } from 'react-i18next'
import StatusTag from '../../components/StatusTag'
import { getAccounts, exportAccounts, checkAccount, deleteAccount, type Account } from '../../api/accounts'
import { getTemplatesForUpload, pushAccountToTemplate, type PushTemplate } from '../../api/pushTemplates'

const { Title } = Typography

interface AccountTableTemplateProps {
  title: string
  accountType?: string
  extraColumns?: ColumnsType<Account>
  hideTypeColumn?: boolean
}

export default function AccountTableTemplate({
  title,
  accountType,
  extraColumns,
  hideTypeColumn = false,
}: AccountTableTemplateProps) {
  const [accounts, setAccounts] = useState<Account[]>([])
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState('')
  const [templatesByType, setTemplatesByType] = useState<Record<string, PushTemplate[]>>({})
  const [pushingKey, setPushingKey] = useState<string | null>(null)
  const [checkingId, setCheckingId] = useState<number | null>(null)
  const [deletingId, setDeletingId] = useState<number | null>(null)
  const { t } = useTranslation()

  const statusOptions = [
    { value: '', label: t('accounts.allStatuses') },
    { value: 'active', label: t('accounts.active') },
    { value: 'banned', label: t('accounts.banned') },
    { value: 'pending', label: t('accounts.pending') },
  ]

  const fetchTemplatesForType = useCallback(async (type: string) => {
    if (!type || type in templatesByType) return
    try {
      const { data } = await getTemplatesForUpload(type)
      setTemplatesByType((prev) => ({ ...prev, [type]: data.push_templates ?? [] }))
    } catch {
      // ignore template loading errors in action dropdown
    }
  }, [templatesByType])

  const fetchAccounts = useCallback(async () => {
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
  }, [accountType, status, t])

  useEffect(() => {
    void fetchAccounts()
  }, [fetchAccounts])

  const handleExport = useCallback(async (format: 'csv' | 'json') => {
    try {
      const { data } = await exportAccounts(format, accountType)
      const url = URL.createObjectURL(data as Blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `accounts.${format}`
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      message.error(t('accounts.exportFailed'))
    }
  }, [accountType, t])

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

  const handleCheckAccount = useCallback(
    async (account: Account) => {
      setCheckingId(account.id)
      try {
        const { data } = await checkAccount(account.id)
        if (!data.supported) {
          message.warning(t('accounts.checkUnsupported'))
          return
        }
        if (data.valid) {
          message.success(t('accounts.checkSuccess'))
          return
        }
        message.error(t('accounts.checkInvalid', { message: data.message }))
      } catch {
        message.error(t('accounts.checkFailed'))
      } finally {
        setCheckingId(null)
      }
    },
    [t],
  )

  const handleDeleteAccount = useCallback(
    async (id: number) => {
      setDeletingId(id)
      try {
        await deleteAccount(id)
        message.success(t('accounts.deleted'))
        await fetchAccounts()
      } catch {
        message.error(t('accounts.failedToDelete'))
      } finally {
        setDeletingId(null)
      }
    },
    [fetchAccounts, t],
  )

  const columns: ColumnsType<Account> = useMemo(() => {
    const baseColumns: ColumnsType<Account> = [{ title: t('common.email'), dataIndex: 'email', key: 'email' }]

    if (!hideTypeColumn) {
      baseColumns.push({ title: t('common.type'), dataIndex: 'type', key: 'type' })
    }

    if (extraColumns && extraColumns.length > 0) {
      baseColumns.push(...extraColumns)
    }

    baseColumns.push(
      {
        title: t('common.status'),
        dataIndex: 'status',
        key: 'status',
        render: (s: string) => <StatusTag status={s} />,
      },
      {
        title: t('common.created'),
        dataIndex: 'created_at',
        key: 'created_at',
        render: (v: string) => new Date(v).toLocaleString(),
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
            onClick: () => void handlePushToTemplate(record, tmpl.id, tmpl.name),
          }))

          return (
            <Space>
              <Button
                size="small"
                icon={<SafetyOutlined />}
                loading={checkingId === record.id}
                onClick={() => void handleCheckAccount(record)}
              >
                {t('accounts.check')}
              </Button>
              <Dropdown
                menu={{
                  items: menuItems.length > 0
                    ? menuItems
                    : [{ key: 'none', label: t('accounts.noTemplates'), disabled: true }],
                }}
                trigger={['click']}
                onOpenChange={(open) => {
                  if (open) {
                    void fetchTemplatesForType(record.type)
                  }
                }}
              >
                <Button size="small" icon={<UploadOutlined />}>
                  {t('accounts.upload')}
                </Button>
              </Dropdown>
              <Popconfirm
                title={t('accounts.deleteConfirm')}
                onConfirm={() => void handleDeleteAccount(record.id)}
                okText={t('common.yes')}
                cancelText={t('common.no')}
              >
                <Button size="small" danger loading={deletingId === record.id}>
                  {t('common.delete')}
                </Button>
              </Popconfirm>
            </Space>
          )
        },
      },
    )

    return baseColumns
  }, [
    checkingId,
    deletingId,
    extraColumns,
    fetchAccounts,
    fetchTemplatesForType,
    handleCheckAccount,
    handleDeleteAccount,
    handlePushToTemplate,
    hideTypeColumn,
    pushingKey,
    t,
    templatesByType,
  ])

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>
          {title}
        </Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchAccounts()} loading={loading}>
            {t('accounts.refresh')}
          </Button>
          <Button icon={<DownloadOutlined />} onClick={() => void handleExport('csv')}>
            {t('accounts.exportCsv')}
          </Button>
          <Button icon={<DownloadOutlined />} onClick={() => void handleExport('json')}>
            {t('accounts.exportJson')}
          </Button>
        </Space>
      </div>

      <Space style={{ marginBottom: 16 }}>
        <Select
          value={status}
          onChange={setStatus}
          options={statusOptions}
          style={{ width: 180 }}
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
