import { Bot, Zap, Sparkles, Terminal } from 'lucide-react'
import { cn } from '../lib/utils'
import type { ActiveTool } from '../stores/configStore'

interface TabItem {
  id: ActiveTool
  label: string
  icon: React.ComponentType<{ className?: string }>
  color: string
}

const ALL_TOOLS: TabItem[] = [
  { id: 'claude',    label: 'Claude Code', icon: Bot,      color: 'text-orange-500' },
  { id: 'codex',     label: 'Codex',       icon: Zap,      color: 'text-green-500'  },
  { id: 'gemini',    label: 'Gemini CLI',  icon: Sparkles, color: 'text-blue-500'   },
  { id: 'picoclaw',  label: 'PicoClaw',    icon: Terminal, color: 'text-pink-500'   },
  { id: 'nullclaw',  label: 'NullClaw',    icon: Terminal, color: 'text-cyan-500'   },
  { id: 'zeroclaw',  label: 'ZeroClaw',    icon: Terminal, color: 'text-violet-500' },
  { id: 'openclaw',  label: 'OpenClaw',    icon: Terminal, color: 'text-rose-500'   },
]

interface ProductTabBarProps {
  activeTool: string
  onSelect: (tool: ActiveTool) => void
}

export function ProductTabBar({ activeTool, onSelect }: ProductTabBarProps) {
  return (
    <div className="flex items-center gap-0.5 px-3 py-1 border-b border-border bg-background shrink-0 overflow-x-auto">
      {ALL_TOOLS.map(({ id, label, icon: Icon, color }) => {
        const isActive = activeTool === id
        return (
          <button
            key={id}
            onClick={() => onSelect(id)}
            className={cn(
              'flex items-center gap-1.5 px-3 py-2 text-xs font-medium rounded-t-md transition-colors whitespace-nowrap',
              'border-b-2',
              isActive
                ? 'border-primary text-foreground bg-muted/50'
                : 'border-transparent text-muted-foreground hover:text-foreground hover:bg-muted/30'
            )}
          >
            <Icon className={cn('h-3.5 w-3.5', isActive ? color : '')} />
            {label}
          </button>
        )
      })}
    </div>
  )
}
