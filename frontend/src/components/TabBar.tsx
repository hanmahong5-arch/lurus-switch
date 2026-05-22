import { cn } from '../lib/utils'

interface Tab {
  id: string
  label: string
  icon?: React.ComponentType<{ className?: string }>
}

interface TabBarProps {
  tabs: Tab[]
  activeTab: string
  onTabChange: (id: string) => void
}

export function TabBar({ tabs, activeTab, onTabChange }: TabBarProps) {
  return (
    <div className="flex gap-1 border-b border-border px-4 pt-3">
      {tabs.map((tab) => {
        const Icon = tab.icon
        const isActive = activeTab === tab.id
        return (
          <button
            key={tab.id}
            onClick={() => onTabChange(tab.id)}
            className={cn(
              'group inline-flex items-center gap-1.5 px-3 py-2 -mb-px transition-all duration-150',
              'border-b-2 border-transparent',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary rounded-t-sm',
              isActive
                ? 'border-primary text-primary'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted/50',
            )}
          >
            {Icon && <Icon className="h-3.5 w-3.5" />}
            <span
              className={cn(
                isActive
                  ? 'font-mono text-[11px] tracking-[0.12em]'
                  : 'text-sm font-medium',
              )}
            >
              {isActive ? `[ ${tab.label.toUpperCase()} ]` : tab.label}
            </span>
          </button>
        )
      })}
    </div>
  )
}
