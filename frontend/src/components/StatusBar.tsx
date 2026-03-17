import { useTranslation } from 'react-i18next'
import { useConfigStore } from '../stores/configStore'
import { AccountStatusBadge } from './AccountStatusBadge'
import { useDashboardStore } from '../stores/dashboardStore'

export function StatusBar() {
  const { status } = useConfigStore()
  const { appVersion } = useDashboardStore()
  const { t } = useTranslation()

  return (
    <footer className="h-6 bg-muted/50 border-t border-border flex items-center justify-between px-4 text-xs text-muted-foreground">
      <span>{t('statusBar.status')}: {status}</span>
      <div className="flex items-center gap-3">
        <AccountStatusBadge />
        <span>v{appVersion || '1.0.0'}</span>
      </div>
    </footer>
  )
}
