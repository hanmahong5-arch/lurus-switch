import { Settings, Zap, Sparkles, Bot } from 'lucide-react'
import { cn } from '../lib/utils'
import { useConfigStore } from '../stores/configStore'

const tools = [
  { id: 'claude' as const, name: 'Claude Code', icon: Bot, color: 'text-orange-500' },
  { id: 'codex' as const, name: 'Codex', icon: Zap, color: 'text-green-500' },
  { id: 'gemini' as const, name: 'Gemini CLI', icon: Sparkles, color: 'text-blue-500' },
]

export function Sidebar() {
  const { activeTool, setActiveTool } = useConfigStore()

  return (
    <aside className="w-56 bg-muted/50 border-r border-border flex flex-col">
      {/* Logo / Title */}
      <div className="p-4 border-b border-border wails-drag">
        <h1 className="text-lg font-semibold">Lurus Switch</h1>
        <p className="text-xs text-muted-foreground">AI CLI Config Generator</p>
      </div>

      {/* Tool Selection */}
      <nav className="flex-1 p-2">
        <div className="space-y-1">
          {tools.map((tool) => (
            <button
              key={tool.id}
              onClick={() => setActiveTool(tool.id)}
              className={cn(
                'w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors',
                activeTool === tool.id
                  ? 'bg-primary text-primary-foreground'
                  : 'hover:bg-muted text-muted-foreground hover:text-foreground'
              )}
            >
              <tool.icon className={cn('h-5 w-5', activeTool !== tool.id && tool.color)} />
              {tool.name}
            </button>
          ))}
        </div>
      </nav>

      {/* Settings */}
      <div className="p-2 border-t border-border">
        <button className="w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors">
          <Settings className="h-5 w-5" />
          Settings
        </button>
      </div>
    </aside>
  )
}
