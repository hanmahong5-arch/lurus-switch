import { useTranslation } from 'react-i18next'
import { useConfigStore } from '../stores/configStore'

export function StatusBar() {
  const { status } = useConfigStore()
  const { t } = useTranslation()

  return (
    <footer className="h-6 bg-muted/50 border-t border-border flex items-center justify-between px-4 text-xs text-muted-foreground">
      <span>{t('statusBar.status')}: {status}</span>
      <span>v1.0.0</span>
    </footer>
  )
}
