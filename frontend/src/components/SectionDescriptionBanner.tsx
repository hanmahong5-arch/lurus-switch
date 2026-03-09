import {
  Settings, Shield, Box, Sliders, Plug, Server, Database,
  Layers, Zap, Key, Bot, MoreHorizontal,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { SectionDescriptor } from '../lib/toolSchema'

const ICON_MAP: Record<string, React.ComponentType<{ className?: string }>> = {
  Settings,
  Shield,
  Box,
  Sliders,
  Plug,
  Server,
  Database,
  Layers,
  Zap,
  Key,
  Bot,
  MoreHorizontal,
}

interface SectionDescriptionBannerProps {
  toolId: string
  activeSection: string
  sections: SectionDescriptor[]
}

export function SectionDescriptionBanner({ activeSection, sections }: SectionDescriptionBannerProps) {
  const { t } = useTranslation()

  const descriptor = sections.find((s) => s.id === activeSection)
  if (!descriptor) return null

  const Icon = ICON_MAP[descriptor.icon] ?? Settings
  const title = t(descriptor.titleKey)
  const desc = t(descriptor.descKey)

  // If the translation key was not found (returns the key itself), skip rendering description
  const hasDesc = desc !== descriptor.descKey

  return (
    <div className="sticky top-0 z-10 bg-muted/60 backdrop-blur-sm border-b border-border px-6 py-2 flex items-center gap-3 shrink-0">
      <Icon className="h-4 w-4 text-muted-foreground shrink-0" />
      <span className="text-xs font-semibold">{title}</span>
      {hasDesc && (
        <span className="text-xs text-muted-foreground font-normal">{desc}</span>
      )}
    </div>
  )
}
