import { useTranslation } from 'react-i18next'
import AccountTableTemplate from './AccountTableTemplate'

export default function AllAccountList() {
  const { t } = useTranslation()

  return <AccountTableTemplate title={t('accounts.title')} />
}
