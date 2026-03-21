import { WifiOff } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useConnectivityStore } from '../stores/connectivityStore'

/**
 * Persistent banner shown at the top of the app when backend connectivity is lost.
 * Auto-hides when connectivity recovers.
 */
export function ConnectionBanner() {
  const { t } = useTranslation()
  const online = useConnectivityStore((s) => s.online)

  if (online) return null

  return (
    <div className="bg-amber-500/15 border-b border-amber-500/30 px-4 py-2 flex items-center gap-2 text-xs text-amber-600 shrink-0">
      <WifiOff className="h-3.5 w-3.5 shrink-0 animate-pulse" />
      <span>{t('error.offline', 'Connection to backend lost — some data may be stale')}</span>
    </div>
  )
}
