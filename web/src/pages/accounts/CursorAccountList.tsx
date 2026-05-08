import type { ColumnsType } from 'antd/es/table'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { Account } from '../../api/accounts'
import AccountTableTemplate from './AccountTableTemplate'
import { parseAccountExtra, maskToken } from './accountExtra'

export default function CursorAccountList() {
  const { t } = useTranslation()

  const columns: ColumnsType<Account> = useMemo(() => [
    {
      title: t('accounts.cursorToken'),
      key: 'cursor_token',
      render: (_, record) => {
        const extra = parseAccountExtra(record.extra)
        const token = extra.token
        if (typeof token !== 'string' || !token) return '-'
        return maskToken(token)
      },
    },
  ], [t])

  return (
    <AccountTableTemplate
      title={t('accounts.cursorTitle')}
      accountType="cursor"
      extraColumns={columns}
      hideTypeColumn
    />
  )
}
