import { Search } from 'lucide-react'

interface SearchBarProps {
  value: string
  onChange: (value: string) => void
  onSearch: () => void
  placeholder?: string
  children?: React.ReactNode
}

export function SearchBar({ value, onChange, onSearch, placeholder = 'Search...', children }: SearchBarProps) {
  return (
    <div className="flex items-center gap-2 flex-wrap">
      <div className="relative flex-1 min-w-[200px]">
        <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <input
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && onSearch()}
          placeholder={placeholder}
          className="w-full pl-9 pr-3 py-1.5 rounded-md border border-border bg-background text-sm focus:outline-none focus:ring-1 focus:ring-primary"
        />
      </div>
      <button
        onClick={onSearch}
        className="px-3 py-1.5 rounded-md bg-indigo-600 hover:bg-indigo-500 text-white text-sm"
      >
        Search
      </button>
      {children}
    </div>
  )
}
