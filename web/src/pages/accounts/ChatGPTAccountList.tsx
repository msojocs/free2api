import type { ColumnsType } from 'antd/es/table'
import { useCallback, useMemo, useState } from 'react'
import { Button, Modal, Popover, Progress, Space, message } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import {
  getChatGPTAccountDetail,
  refreshChatGPTToken,
  type Account,
  type ChatGPTAccountDetailResult,
} from '../../api/accounts'
import AccountTableTemplate from './AccountTableTemplate'
import { extractJwtExp } from './accountExtra'

interface UsageInfo {
  used_percent: number
  limit_reached: boolean
  reset_at?: number
}

interface UsageSummary {
  usedPercent: number
  limitReached: boolean
  resetAt: number | null
}

export default function ChatGPTAccountList() {
  const { t } = useTranslation()
  const [refreshingId, setRefreshingId] = useState<number | null>(null)
  const [refreshErrorMap, setRefreshErrorMap] = useState<Record<number, string>>({})
  const [detailLoadingId, setDetailLoadingId] = useState<number | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<ChatGPTAccountDetailResult | null>(null)

  const extractUsageSummary = useCallback((record: Account): UsageSummary | null => {
    const usage = record.usage
    if (!usage || typeof usage !== 'object') {
      return null
    }
    const usageObj = usage as unknown as UsageInfo & Record<string, unknown>

    if (typeof usageObj.used_percent === 'number' && Number.isFinite(usageObj.used_percent)) {
      return {
        usedPercent: Math.max(0, Math.min(100, usageObj.used_percent)),
        limitReached: typeof usageObj.limit_reached === 'boolean' ? usageObj.limit_reached : false,
        resetAt: typeof usageObj.reset_at === 'number' && Number.isFinite(usageObj.reset_at) ? usageObj.reset_at : null,
      }
    }

    // Backward compatibility for old rows storing raw payload.
    const codeReview = usageObj.code_review_rate_limit as Record<string, unknown> | undefined
    const globalLimit = usageObj.rate_limit as Record<string, unknown> | undefined
    const codeReviewWindow = codeReview?.primary_window as Record<string, unknown> | undefined
    const globalWindow = globalLimit?.primary_window as Record<string, unknown> | undefined
    const usedPercentRaw = codeReviewWindow?.used_percent ?? globalWindow?.used_percent
    if (typeof usedPercentRaw !== 'number' || !Number.isFinite(usedPercentRaw)) {
      return null
    }
    const limitReachedRaw = codeReview?.limit_reached ?? globalLimit?.limit_reached
    const resetAtRaw = codeReviewWindow?.reset_at ?? globalWindow?.reset_at
    return {
      usedPercent: Math.max(0, Math.min(100, usedPercentRaw)),
      limitReached: typeof limitReachedRaw === 'boolean' ? limitReachedRaw : false,
      resetAt: typeof resetAtRaw === 'number' && Number.isFinite(resetAtRaw) ? resetAtRaw : null,
    }
  }, [])

  const columns: ColumnsType<Account> = useMemo(() => [
    {
      title: t('accounts.accessTokenExpiresAt'),
      key: 'access_token_expires_at',
      render: (_, record) => {
        const extra = record.extra ?? {}
        const accessToken = extra.access_token
        if (typeof accessToken !== 'string') return '-'
        const exp = extractJwtExp(accessToken)
        if (!exp) return '-'
        return new Date(exp * 1000).toLocaleString()
      },
    },
    {
      title: t('accounts.usage'),
      key: 'usage',
      render: (_, record) => {
        const usageSummary = extractUsageSummary(record)
        if (!usageSummary) {
          return '-'
        }
        const resetTime = usageSummary.resetAt ? new Date(usageSummary.resetAt * 1000).toLocaleString() : '-'
        return (
          <Popover
            trigger={['hover', 'click']}
            content={(
              <span>
                {t('accounts.resetTime')}: {resetTime}
              </span>
            )}
          >
            <div style={{ width: 170 }}>
              <Progress
                size="small"
                percent={usageSummary.usedPercent}
                status={usageSummary.limitReached ? 'exception' : 'normal'}
                format={(percent) => `${(percent ?? 0).toFixed(1)}%`}
              />
            </div>
          </Popover>
        )
      },
    },
  ], [extractUsageSummary, t])

  const handleRefreshToken = useCallback(async (record: Account) => {
    setRefreshingId(record.id)
    setRefreshErrorMap((prev) => {
      const next = { ...prev }
      delete next[record.id]
      return next
    })
    try {
      await refreshChatGPTToken(record.id)
      message.success(t('accounts.refreshTokenSuccess'))
    } catch (e) {
      const errMsg = e instanceof Error ? e.message : String(e ?? '')
      setRefreshErrorMap((prev) => ({ ...prev, [record.id]: errMsg || t('accounts.refreshTokenFailed') }))
      message.error(t('accounts.refreshTokenFailed'))
    } finally {
      setRefreshingId(null)
    }
  }, [t])

  const handleViewDetail = useCallback(async (record: Account) => {
    setDetailOpen(true)
    setDetailLoadingId(record.id)
    try {
      const { data } = await getChatGPTAccountDetail(record.id)
      setDetail(data)
    } catch (err) {
      message.error(
        t('accounts.detailFailed', {
          message: err instanceof Error ? err.message : 'unknown error',
        }),
      )
      setDetail(null)
    } finally {
      setDetailLoadingId(null)
    }
  }, [t])

  const renderExtraActions = useCallback((record: Account) => (
    <Space size={4}>
      {refreshErrorMap[record.id] ? (
        <Popover
          trigger={["hover", "click"]}
          content={(
            <div style={{ maxWidth: 320, display: 'block', whiteSpace: 'normal', wordBreak: 'break-word' }}>
              {refreshErrorMap[record.id]}
            </div>
          )}
          title={t('accounts.refreshTokenFailed')}
        >
          <Button
            size="small"
            icon={<ReloadOutlined />}
            loading={refreshingId === record.id}
            onClick={() => void handleRefreshToken(record)}
            danger
          >
            {t('accounts.refreshToken')}
          </Button>
        </Popover>
      ) : (
        <Button
          size="small"
          icon={<ReloadOutlined />}
          loading={refreshingId === record.id}
          onClick={() => void handleRefreshToken(record)}
        >
          {t('accounts.refreshToken')}
        </Button>
      )}
    </Space>
  ), [handleRefreshToken, refreshingId, refreshErrorMap, t])

  const renderEmail = useCallback((record: Account) => (
    <Button
      type="link"
      size="small"
      style={{ padding: 0, height: 'auto' }}
      loading={detailLoadingId === record.id}
      onClick={() => void handleViewDetail(record)}
    >
      {record.email}
    </Button>
  ), [detailLoadingId, handleViewDetail])

  return (
    <>
      <AccountTableTemplate
        title={t('accounts.chatgptTitle')}
        accountType="chatgpt"
        extraColumns={columns}
        hideTypeColumn
        renderExtraActions={renderExtraActions}
        renderEmail={renderEmail}
      />
      <Modal
        title={t('accounts.detailTitle')}
        open={detailOpen}
        onCancel={() => setDetailOpen(false)}
        footer={null}
        width={820}
      >
        <pre style={{ maxHeight: '60vh', overflow: 'auto', margin: 0 }}>
          {detailLoadingId !== null
            ? t('common.loading')
            : detail
              ? JSON.stringify(detail, null, 2)
              : t('accounts.detailEmpty')}
        </pre>
      </Modal>
    </>
  )
}
