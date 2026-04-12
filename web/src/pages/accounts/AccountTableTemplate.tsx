import { useEffect, useMemo, useState, useCallback, useRef, type Key, type ReactNode } from 'react'
import { Table, Button, Space, Select, Typography, message, Dropdown, Popconfirm, Popover } from 'antd'
import { DownloadOutlined, ReloadOutlined, UploadOutlined, SafetyOutlined, MoreOutlined, DeleteOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { MenuProps } from 'antd'
import { useTranslation } from 'react-i18next'
import StatusTag from '../../components/StatusTag'
import { getAccounts, exportAccounts, importAccounts, checkAccount, deleteAccount, type Account } from '../../api/accounts'
import { getTemplatesForUpload, pushAccountToTemplate, type PushTemplate } from '../../api/pushTemplates'

const { Title } = Typography

interface AccountTableTemplateProps {
  title: string
  accountType?: string
  extraColumns?: ColumnsType<Account>
  hideTypeColumn?: boolean
  renderExtraActions?: (record: Account) => ReactNode
  renderEmail?: (record: Account) => ReactNode
}

interface CheckResult {
  status: 'success' | 'failed'
  message?: string
}

export default function AccountTableTemplate({
  title,
  accountType,
  extraColumns,
  hideTypeColumn = false,
  renderExtraActions,
  renderEmail,
}: AccountTableTemplateProps) {
  const [accounts, setAccounts] = useState<Account[]>([])
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const PAGE_SIZE = 20
  const [templatesByType, setTemplatesByType] = useState<Record<string, PushTemplate[]>>({})
  const [pushingKey, setPushingKey] = useState<string | null>(null)
  const [checkingId, setCheckingId] = useState<number | null>(null)
  const [deletingId, setDeletingId] = useState<number | null>(null)
  const [batchChecking, setBatchChecking] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([])
  const [checkResultMap, setCheckResultMap] = useState<Record<number, CheckResult>>({})
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

  const fetchAccounts = useCallback(async (p?: number) => {
    const currentPage = p ?? page
    setLoading(true)
    try {
      const { data } = await getAccounts({
        type: accountType || undefined,
        status: status || undefined,
        page: currentPage,
        limit: PAGE_SIZE,
      })
      setAccounts(data.accounts ?? [])
      setTotal(data.total ?? 0)
    } catch {
      message.error(t('accounts.failedToLoad'))
    } finally {
      setLoading(false)
    }
  }, [accountType, page, status, t])

  useEffect(() => {
    setPage(1)
  }, [accountType, status])

  useEffect(() => {
    void fetchAccounts(page)
  }, [page])

  const importFileRef = useRef<HTMLInputElement>(null)

  const handleExport = useCallback(async () => {
    try {
      const { data } = await exportAccounts(accountType)
      const url = URL.createObjectURL(data as Blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'accounts.json'
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      message.error(t('accounts.exportFailed'))
    }
  }, [accountType, t])

  const handleImportFile = useCallback(async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    try {
      const text = await file.text()
      const records = JSON.parse(text)
      const { data } = await importAccounts(records)
      message.success(t('accounts.importSuccess', { imported: data.imported, skipped: data.skipped, failed: data.failed }))
      await fetchAccounts()
    } catch {
      message.error(t('accounts.importFailed'))
    } finally {
      e.target.value = ''
    }
  }, [fetchAccounts, t])

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
    async (account: Account): Promise<boolean> => {
      setCheckingId(account.id)
      try {
        const { data } = await checkAccount(account.id)
        if (!data.supported) {
          message.warning(t('accounts.checkUnsupported'))
          setCheckResultMap((prev) => {
            const next = { ...prev }
            delete next[account.id]
            return next
          })
          return false
        }
        if (data.valid) {
          if (typeof data.status === 'string' && data.status) {
            setAccounts((prev) => prev.map((item) => (item.id === account.id ? { ...item, status: data.status as string } : item)))
          }
          if (data.usage && typeof data.usage === 'object') {
            setAccounts((prev) =>
              prev.map((item) => (item.id === account.id ? { ...item, usage: data.usage } : item)),
            )
          }
          setCheckResultMap((prev) => ({ ...prev, [account.id]: { status: 'success' } }))
          message.success(t('accounts.checkSuccess'))
          return true
        }
        const errorMessage = data.message || t('accounts.checkFailedUnknown')
        if (typeof data.status === 'string' && data.status) {
          setAccounts((prev) => prev.map((item) => (item.id === account.id ? { ...item, status: data.status as string } : item)))
        }
        setCheckResultMap((prev) => ({ ...prev, [account.id]: { status: 'failed', message: errorMessage } }))
        message.error(t('accounts.checkInvalid', { message: errorMessage }))
        return false
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : t('accounts.checkFailedUnknown')
        setCheckResultMap((prev) => ({ ...prev, [account.id]: { status: 'failed', message: errorMessage } }))
        message.error(t('accounts.checkFailed', { message: errorMessage }))
        return false
      } finally {
        setCheckingId(null)
      }
    },
    [t],
  )

  const handleBatchCheck = useCallback(async () => {
    const selectedAccounts = accounts.filter((account) => selectedRowKeys.includes(account.id))
    if (selectedAccounts.length === 0) {
      message.warning(t('accounts.batchCheckSelectFirst'))
      return
    }

    setBatchChecking(true)
    let successCount = 0
    try {
      for (const account of selectedAccounts) {
        // Run sequentially to avoid overwhelming upstream APIs.
        const ok = await handleCheckAccount(account)
        if (ok) {
          successCount += 1
        }
      }
      message.info(
        t('accounts.batchCheckSummary', {
          total: selectedAccounts.length,
          success: successCount,
          failed: selectedAccounts.length - successCount,
        }),
      )
    } finally {
      setBatchChecking(false)
    }
  }, [accounts, handleCheckAccount, selectedRowKeys, t])

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
    const baseColumns: ColumnsType<Account> = [
      {
        title: t('common.email'),
        dataIndex: 'email',
        key: 'email',
        render: (_, record) => (renderEmail ? renderEmail(record) : record.email),
      },
    ]

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

          const checkResult = checkResultMap[record.id]
          const checkStatus = checkResult?.status
          const checkButtonStyle = checkStatus === 'success'
            ? { backgroundColor: '#52c41a', borderColor: '#52c41a', color: '#fff' }
            : checkStatus === 'failed'
              ? { backgroundColor: '#ff4d4f', borderColor: '#ff4d4f', color: '#fff' }
              : undefined
          const checkButton = (
            <Button
              size="small"
              icon={<SafetyOutlined />}
              loading={checkingId === record.id}
              onClick={() => void handleCheckAccount(record)}
              style={checkButtonStyle}
            >
              {t('accounts.check')}
            </Button>
          )

          return (
            <Space>
              {checkStatus === 'failed' && checkResult?.message ? (
                <Popover
                  title={t('accounts.checkErrorPopoverTitle')}
                  content={(
                    <div style={{ maxWidth: 320, display: 'block', whiteSpace: 'normal', wordBreak: 'break-word' }}>
                      {checkResult.message}
                    </div>
                  )}
                  trigger={['hover', 'click']}
                >
                  {checkButton}
                </Popover>
              ) : checkButton}
              {renderExtraActions ? renderExtraActions(record) : null}
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
              <Dropdown
                trigger={['click']}
                menu={{
                  items: [
                    {
                      key: 'delete',
                      label: (
                        <Popconfirm
                          title={t('accounts.deleteConfirm')}
                          onConfirm={() => void handleDeleteAccount(record.id)}
                          okText={t('common.yes')}
                          cancelText={t('common.no')}
                        >
                          <span style={{ color: '#ff4d4f' }}>
                            <DeleteOutlined style={{ marginRight: 6 }} />
                            {t('common.delete')}
                          </span>
                        </Popconfirm>
                      ),
                    },
                  ],
                }}
              >
                <Button size="small" icon={<MoreOutlined />} loading={deletingId === record.id} />
              </Dropdown>
            </Space>
          )
        },
      },
    )

    return baseColumns
  }, [
    checkingId,
    checkResultMap,
    deletingId,
    extraColumns,
    fetchAccounts,
    fetchTemplatesForType,
    handleCheckAccount,
    handleDeleteAccount,
    handlePushToTemplate,
    hideTypeColumn,
    pushingKey,
    renderExtraActions,
    renderEmail,
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
          <Button
            icon={<SafetyOutlined />}
            onClick={() => void handleBatchCheck()}
            disabled={selectedRowKeys.length === 0}
            loading={batchChecking}
          >
            {t('accounts.batchCheck')}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => void fetchAccounts()} loading={loading}>
            {t('accounts.refresh')}
          </Button>
          <Button icon={<DownloadOutlined />} onClick={() => void handleExport()}>
            {t('accounts.exportJson')}
          </Button>
          <input
            ref={importFileRef}
            type="file"
            accept=".json"
            style={{ display: 'none' }}
            onChange={(e) => void handleImportFile(e)}
          />
          <Button icon={<UploadOutlined />} onClick={() => importFileRef.current?.click()}>
            {t('accounts.importJson')}
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
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys),
        }}
        loading={loading}
        pagination={{
          current: page,
          pageSize: PAGE_SIZE,
          total,
          onChange: (p) => setPage(p),
          showSizeChanger: false,
        }}
        scroll={{ x: 'max-content' }}
      />
    </div>
  )
}
