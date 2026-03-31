import { Bot, Zap, Sparkles, Terminal, Cpu, Globe } from 'lucide-react'

export type DepType = 'bun' | 'standalone'

export interface ToolMetaEntry {
  label: string
  icon: typeof Bot
  color: string
  bgColor: string
  dep: DepType
}

export const TOOL_ORDER = ['claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw'] as const

export const TOOL_DISPLAY: Record<string, string> = {
  claude: 'Claude Code', codex: 'Codex', gemini: 'Gemini CLI',
  picoclaw: 'PicoClaw', nullclaw: 'NullClaw', zeroclaw: 'ZeroClaw', openclaw: 'OpenClaw',
}

export const toolMeta: Record<string, ToolMetaEntry> = {
  claude:   { label: 'Claude Code', icon: Bot,      color: 'text-orange-500', bgColor: 'bg-orange-500/10', dep: 'bun' },
  codex:    { label: 'Codex',       icon: Zap,      color: 'text-green-500',  bgColor: 'bg-green-500/10',  dep: 'bun' },
  gemini:   { label: 'Gemini CLI',  icon: Sparkles, color: 'text-blue-500',   bgColor: 'bg-blue-500/10',   dep: 'bun' },
  picoclaw: { label: 'PicoClaw',    icon: Terminal, color: 'text-pink-500',   bgColor: 'bg-pink-500/10',   dep: 'standalone' },
  nullclaw: { label: 'NullClaw',    icon: Terminal, color: 'text-cyan-500',   bgColor: 'bg-cyan-500/10',   dep: 'standalone' },
  zeroclaw: { label: 'ZeroClaw',    icon: Cpu,      color: 'text-violet-500', bgColor: 'bg-violet-500/10', dep: 'standalone' },
  openclaw: { label: 'OpenClaw',    icon: Globe,    color: 'text-teal-500',   bgColor: 'bg-teal-500/10',   dep: 'bun' },
}

export const DEFAULT_TOOL_META: ToolMetaEntry = {
  label: 'Unknown', icon: Bot, color: 'text-gray-500', bgColor: 'bg-gray-500/10', dep: 'standalone' as DepType,
}
