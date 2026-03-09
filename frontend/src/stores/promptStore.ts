import { create } from 'zustand'

interface Prompt {
  id: string
  name: string
  category: string
  tags: string[]
  content: string
  targetTools: string[]
  createdAt: string
  updatedAt: string
}

type Category = 'all' | 'coding' | 'writing' | 'analysis' | 'custom'

interface PromptStore {
  prompts: Prompt[]
  builtins: Prompt[]
  selectedId: string | null
  category: Category
  searchQuery: string

  setPrompts: (p: Prompt[]) => void
  setBuiltins: (p: Prompt[]) => void
  setSelectedId: (id: string | null) => void
  setCategory: (c: Category) => void
  setSearchQuery: (q: string) => void
}

export const usePromptStore = create<PromptStore>((set) => ({
  prompts: [],
  builtins: [],
  selectedId: null,
  category: 'all',
  searchQuery: '',

  setPrompts: (prompts) => set({ prompts }),
  setBuiltins: (builtins) => set({ builtins }),
  setSelectedId: (selectedId) => set({ selectedId }),
  setCategory: (category) => set({ category }),
  setSearchQuery: (searchQuery) => set({ searchQuery }),
}))
