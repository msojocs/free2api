import type { ColumnsType } from 'antd/es/table'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { Account } from '../../api/accounts'
import AccountTableTemplate from './AccountTableTemplate'
import { parseAccountExtra, extractChatGPTAccountIdFromToken, extractJwtExp } from './accountExtra'

export default function ChatGPTAccountList() {
  const { t } = useTranslation()

  const columns: ColumnsType<Account> = useMemo(() => [
    {
      title: t('accounts.accountId'),
      key: 'account_id',
      render: (_, record) => {
        const extra = parseAccountExtra(record.extra)
        const fromExtra = extra.account_id
        if (typeof fromExtra === 'string' && fromExtra) return fromExtra

        const accessToken = extra.access_token
        if (typeof accessToken !== 'string') return '-'
        const parsed = extractChatGPTAccountIdFromToken(accessToken)
        return parsed || '-'
      },
    },
    {
      title: t('accounts.accessTokenExpiresAt'),
      key: 'access_token_expires_at',
      render: (_, record) => {
        const extra = parseAccountExtra(record.extra)
        const accessToken = extra.access_token
        if (typeof accessToken !== 'string') return '-'
        const exp = extractJwtExp(accessToken)
        if (!exp) return '-'
        return new Date(exp * 1000).toLocaleString()
      },
    },
  ], [t])

  return (
    <AccountTableTemplate
      title={t('accounts.chatgptTitle')}
      accountType="chatgpt"
      extraColumns={columns}
      hideTypeColumn
    />
  )
}
