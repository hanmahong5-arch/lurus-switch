import { type ReactNode } from 'react'
import * as RadixTabs from '@radix-ui/react-tabs'
import { type LucideIcon } from 'lucide-react'
import { cn } from '../../lib/utils'

export interface TabItem {
  value: string
  label: string
  icon?: LucideIcon
  disabled?: boolean
}

export type TabsVariant = 'underline' | 'pill'

interface TabsProps {
  tabs: TabItem[]
  value: string
  onValueChange: (v: string) => void
  variant?: TabsVariant
  className?: string
  listClassName?: string
  children?: ReactNode
}

export function Tabs({
  tabs, value, onValueChange, variant = 'underline', className, listClassName, children,
}: TabsProps) {
  return (
    <RadixTabs.Root value={value} onValueChange={onValueChange} className={className}>
      <RadixTabs.List
        className={cn(
          'flex items-center gap-1',
          variant === 'underline' && 'border-b border-border',
          listClassName,
        )}
      >
        {tabs.map((tab) => {
          const Icon = tab.icon
          const active = tab.value === value
          return (
            <RadixTabs.Trigger
              key={tab.value}
              value={tab.value}
              disabled={tab.disabled}
              className={cn(
                'group inline-flex items-center gap-1.5 px-3 py-1.5 transition-all duration-150',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary',
                'disabled:opacity-40 disabled:cursor-not-allowed',
                variant === 'underline' && [
                  '-mb-px border-b-2 border-transparent',
                  'data-[state=active]:border-primary data-[state=active]:text-primary',
                  'text-muted-foreground hover:text-foreground',
                ],
                variant === 'pill' && [
                  'rounded-md',
                  'data-[state=active]:bg-primary/10 data-[state=active]:text-primary',
                  'text-muted-foreground hover:bg-muted hover:text-foreground',
                ],
              )}
            >
              {Icon && <Icon className="h-3.5 w-3.5" />}
              <span
                className={cn(
                  active ? 'font-mono text-[11px] tracking-[0.12em]' : 'text-xs',
                )}
              >
                {active ? `[ ${tab.label.toUpperCase()} ]` : tab.label}
              </span>
            </RadixTabs.Trigger>
          )
        })}
      </RadixTabs.List>
      {children}
    </RadixTabs.Root>
  )
}

export const TabsContent = RadixTabs.Content
