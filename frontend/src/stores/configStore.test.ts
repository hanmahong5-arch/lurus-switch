import { describe, it, expect, beforeEach } from 'vitest'
import { useConfigStore, migrateLegacyRoute } from './configStore'

describe('configStore', () => {
  beforeEach(() => {
    useConfigStore.setState({
      activeTool: 'home',
      previewContent: '',
      status: 'Ready',
      subTabState: {},
      savedConfigs: {
        claude: [],
        codex: [],
        gemini: [],
        picoclaw: [],
        nullclaw: [],
        zeroclaw: [],
        openclaw: [],
      },
    })
  })

  // === Initial State Tests ===
  describe('initial state', () => {
    it('should have home as the default active tool', () => {
      const { activeTool } = useConfigStore.getState()
      expect(activeTool).toBe('home')
    })

    it('should have Ready status', () => {
      const { status } = useConfigStore.getState()
      expect(status).toBe('Ready')
    })

    it('should have empty preview content', () => {
      const { previewContent } = useConfigStore.getState()
      expect(previewContent).toBe('')
    })

    it('should have empty saved configs for all tools', () => {
      const { savedConfigs } = useConfigStore.getState()
      expect(savedConfigs.claude).toEqual([])
      expect(savedConfigs.codex).toEqual([])
      expect(savedConfigs.gemini).toEqual([])
      expect(savedConfigs.picoclaw).toEqual([])
      expect(savedConfigs.nullclaw).toEqual([])
      expect(savedConfigs.zeroclaw).toEqual([])
      expect(savedConfigs.openclaw).toEqual([])
    })
  })

  // === Tool Switching Tests ===
  describe('setActiveTool', () => {
    it('should switch to tools', () => {
      const { setActiveTool } = useConfigStore.getState()
      setActiveTool('tools')
      expect(useConfigStore.getState().activeTool).toBe('tools')
    })

    it('should switch to gateway', () => {
      const { setActiveTool } = useConfigStore.getState()
      setActiveTool('gateway')
      expect(useConfigStore.getState().activeTool).toBe('gateway')
    })

    it('should switch back to home', () => {
      const { setActiveTool } = useConfigStore.getState()
      setActiveTool('tools')
      setActiveTool('home')
      expect(useConfigStore.getState().activeTool).toBe('home')
    })

    it('should switch to all new tool values', () => {
      const tools = [
        'home', 'tools', 'gateway', 'workspace', 'account', 'settings', 'promotion', 'api-admin',
      ] as const

      for (const tool of tools) {
        useConfigStore.getState().setActiveTool(tool)
        expect(useConfigStore.getState().activeTool).toBe(tool)
      }
    })
  })

  // === Sub-tab State Tests ===
  describe('subTabState', () => {
    it('should set and get sub-tab for a page', () => {
      const { setSubTab, getSubTab } = useConfigStore.getState()
      setSubTab('tools', 'codex')
      expect(useConfigStore.getState().getSubTab('tools', 'claude')).toBe('codex')
    })

    it('should return default when no sub-tab set', () => {
      const { getSubTab } = useConfigStore.getState()
      expect(getSubTab('tools', 'claude')).toBe('claude')
    })

    it('should maintain independent sub-tabs per page', () => {
      const { setSubTab } = useConfigStore.getState()
      setSubTab('tools', 'gemini')
      setSubTab('gateway', 'usage')
      setSubTab('workspace', 'context')

      const state = useConfigStore.getState()
      expect(state.getSubTab('tools', 'claude')).toBe('gemini')
      expect(state.getSubTab('gateway', 'control')).toBe('usage')
      expect(state.getSubTab('workspace', 'prompts')).toBe('context')
    })
  })

  // === Legacy Migration Tests ===
  describe('migrateLegacyRoute', () => {
    it('should map dashboard to home', () => {
      expect(migrateLegacyRoute('dashboard')).toEqual({ tool: 'home' })
    })

    it('should map tool names to tools page with subTab', () => {
      expect(migrateLegacyRoute('claude')).toEqual({ tool: 'tools', subTab: 'claude' })
      expect(migrateLegacyRoute('codex')).toEqual({ tool: 'tools', subTab: 'codex' })
    })

    it('should map billing to account', () => {
      expect(migrateLegacyRoute('billing')).toEqual({ tool: 'account', subTab: 'billing' })
    })

    it('should map process to workspace', () => {
      expect(migrateLegacyRoute('process')).toEqual({ tool: 'workspace', subTab: 'process' })
    })

    it('should map gateway-channels to api-admin', () => {
      expect(migrateLegacyRoute('gateway-channels')).toEqual({ tool: 'api-admin', subTab: 'channels' })
    })

    it('should map unknown to home', () => {
      expect(migrateLegacyRoute('unknown-page')).toEqual({ tool: 'home' })
    })
  })

  // === Preview Tests ===
  describe('preview', () => {
    it('setPreviewContent should update preview', () => {
      const { setPreviewContent } = useConfigStore.getState()
      setPreviewContent('{"model": "test"}')

      const { previewContent } = useConfigStore.getState()
      expect(previewContent).toBe('{"model": "test"}')
    })

    it('setPreviewContent should handle empty content', () => {
      const { setPreviewContent } = useConfigStore.getState()
      setPreviewContent('some content')
      setPreviewContent('')

      const { previewContent } = useConfigStore.getState()
      expect(previewContent).toBe('')
    })
  })

  // === Status Tests ===
  describe('status', () => {
    it('setStatus should update status', () => {
      const { setStatus } = useConfigStore.getState()
      setStatus('Saving...')

      const { status } = useConfigStore.getState()
      expect(status).toBe('Saving...')
    })
  })

  // === Saved Configs Tests ===
  describe('savedConfigs', () => {
    it('setSavedConfigs should update claude configs', () => {
      const { setSavedConfigs } = useConfigStore.getState()
      setSavedConfigs('claude', ['config1', 'config2'])

      const { savedConfigs } = useConfigStore.getState()
      expect(savedConfigs.claude).toEqual(['config1', 'config2'])
    })

    it('setSavedConfigs should not affect other tools', () => {
      const { setSavedConfigs } = useConfigStore.getState()
      setSavedConfigs('claude', ['claude-config'])
      setSavedConfigs('codex', ['codex-config'])

      const { savedConfigs } = useConfigStore.getState()
      expect(savedConfigs.claude).toEqual(['claude-config'])
      expect(savedConfigs.codex).toEqual(['codex-config'])
      expect(savedConfigs.gemini).toEqual([])
    })
  })

  // === Edge Cases ===
  describe('edge cases', () => {
    it('should handle rapid tool switches', () => {
      const { setActiveTool } = useConfigStore.getState()

      for (let i = 0; i < 100; i++) {
        setActiveTool(i % 2 === 0 ? 'home' : 'tools')
      }

      expect(useConfigStore.getState().activeTool).toBe('tools')
    })

    it('should handle concurrent updates', () => {
      const { setActiveTool, setStatus, setPreviewContent } = useConfigStore.getState()

      setActiveTool('tools')
      setStatus('Saving...')
      setPreviewContent('preview content')
      setActiveTool('gateway')
      setStatus('Ready')

      const state = useConfigStore.getState()
      expect(state.activeTool).toBe('gateway')
      expect(state.status).toBe('Ready')
      expect(state.previewContent).toBe('preview content')
    })
  })
})
