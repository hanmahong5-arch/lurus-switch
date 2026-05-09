import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, User, Building2, Users, Briefcase } from 'lucide-react'
import { SetAppMode } from '../../wailsjs/go/main/App'
import type { AppMode } from '../stores/configStore'

interface Props {
  onPick: (mode: Exclude<AppMode, 'unset'>) => void
}

interface ModeChoice {
  id: Exclude<AppMode, 'unset'>
  Icon: typeof User
  iconColor: string
  ringColor: string
}

const CHOICES: ModeChoice[] = [
  { id: 'personal', Icon: User, iconColor: 'text-blue-400', ringColor: 'hover:ring-blue-500/50' },
  { id: 'reseller', Icon: Building2, iconColor: 'text-purple-400', ringColor: 'hover:ring-purple-500/50' },
  { id: 'enterprise', Icon: Briefcase, iconColor: 'text-amber-400', ringColor: 'hover:ring-amber-500/50' },
  { id: 'enduser', Icon: Users, iconColor: 'text-emerald-400', ringColor: 'hover:ring-emerald-500/50' },
]

// First-launch wizard for picking the app mode (S-Xa.2).
// EndUser white-label builds skip this entirely — App.tsx detects the lock
// and routes straight to the EndUser flow without rendering this page.
export function AppModeSelectPage({ onPick }: Props) {
  const { t } = useTranslation()
  const [submitting, setSubmitting] = useState<AppMode | null>(null)
  const [error, setError] = useState<string | null>(null)

  const handlePick = async (mode: Exclude<AppMode, 'unset'>) => {
    setSubmitting(mode)
    setError(null)
    try {
      await SetAppMode(mode)
      onPick(mode)
    } catch (e) {
      setError(String(e))
      setSubmitting(null)
    }
  }

  return (
    <div className="h-screen flex flex-col items-center justify-center bg-background text-foreground px-6">
      <div className="max-w-3xl w-full">
        <div className="text-center mb-10">
          <h1 className="text-3xl font-semibold mb-2">{t('mode.welcome.title', '欢迎使用 Lurus Switch')}</h1>
          <p className="text-muted-foreground">{t('mode.welcome.subtitle', '请选择你的使用模式（之后可在设置中更改）')}</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {CHOICES.map(({ id, Icon, iconColor, ringColor }) => {
            const isLoading = submitting === id
            return (
              <button
                key={id}
                disabled={submitting !== null}
                onClick={() => handlePick(id)}
                className={
                  'relative p-6 rounded-xl border border-border bg-card text-left transition-all ' +
                  'hover:border-transparent ring-1 ring-transparent ' + ringColor +
                  ' disabled:opacity-50 disabled:cursor-not-allowed'
                }
              >
                <Icon className={'w-8 h-8 mb-3 ' + iconColor} />
                <h3 className="text-lg font-semibold mb-1">
                  {t(`mode.${id}.label`, id)}
                </h3>
                <p className="text-sm text-muted-foreground mb-3">
                  {t(`mode.${id}.tagline`, '')}
                </p>
                <p className="text-xs text-muted-foreground/80 leading-relaxed">
                  {t(`mode.${id}.desc`, '')}
                </p>
                {isLoading && (
                  <div className="absolute inset-0 flex items-center justify-center bg-card/80 rounded-xl">
                    <Loader2 className="w-5 h-5 animate-spin" />
                  </div>
                )}
              </button>
            )
          })}
        </div>

        {error && (
          <p className="mt-6 text-sm text-red-400 text-center">{error}</p>
        )}

        <p className="text-xs text-muted-foreground/60 text-center mt-8">
          {t('mode.welcome.hint', '不知道选哪个？多数个人用户应选「Personal」。')}
        </p>
      </div>
    </div>
  )
}
