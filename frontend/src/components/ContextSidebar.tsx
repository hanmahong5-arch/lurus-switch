import {
  Settings, Shield, Box, Sliders, Plug, Server, Database,
  Layers, Zap, Key, Bot, MoreHorizontal,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '../lib/utils'
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

interface ContextSidebarProps {
  toolId: string
  sections: SectionDescriptor[]
  activeSection: string
  onSectionClick: (sectionId: string) => void
}

export function ContextSidebar({ sections, activeSection, onSectionClick }: ContextSidebarProps) {
  const { t } = useTranslation()

  if (sections.length === 0) return null

  return (
    <div className="w-44 border-r border-border flex-shrink-0 overflow-y-auto bg-muted/30 px-2 py-3">
      <div className="space-y-0.5">
        {sections.map(({ id, titleKey, icon }) => {
          const Icon = ICON_MAP[icon] ?? Settings
          const isActive = activeSection === id
          return (
            <button
              key={id}
              onClick={() => onSectionClick(id)}
              className={cn(
                'w-full flex items-center gap-2 px-2.5 py-2 rounded-md text-xs font-medium transition-colors text-left',
                isActive
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted'
              )}
            >
              <Icon className="h-3.5 w-3.5 shrink-0" />
              <span className="truncate">{t(titleKey)}</span>
            </button>
          )
        })}
      </div>
    </div>
  )
}
