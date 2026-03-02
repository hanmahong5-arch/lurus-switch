import { describe, it, expect, beforeEach } from 'vitest'
import { useConfigStore } from './configStore'

describe('configStore', () => {
  beforeEach(() => {
    useConfigStore.setState({
      activeTool: 'dashboard',
      previewContent: '',
      status: 'Ready',
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
    it('should have dashboard as the default active tool', () => {
      const { activeTool } = useConfigStore.getState()
      expect(activeTool).toBe('dashboard')
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
    it('should switch to codex', () => {
      const { setActiveTool } = useConfigStore.getState()
      setActiveTool('codex')
      expect(useConfigStore.getState().activeTool).toBe('codex')
    })

    it('should switch to gemini', () => {
      const { setActiveTool } = useConfigStore.getState()
      setActiveTool('gemini')
      expect(useConfigStore.getState().activeTool).toBe('gemini')
    })

    it('should switch back to dashboard', () => {
      const { setActiveTool } = useConfigStore.getState()
      setActiveTool('codex')
      setActiveTool('dashboard')
      expect(useConfigStore.getState().activeTool).toBe('dashboard')
    })

    it('should switch to all tool values', () => {
      const tools = [
        'claude', 'codex', 'gemini', 'picoclaw', 'nullclaw', 'zeroclaw', 'openclaw',
        'billing', 'settings', 'process', 'prompts', 'documents', 'admin',
      ] as const

      for (const tool of tools) {
        useConfigStore.getState().setActiveTool(tool)
        expect(useConfigStore.getState().activeTool).toBe(tool)
      }
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

    it('setPreviewContent should handle multiline content', () => {
      const { setPreviewContent } = useConfigStore.getState()
      const content = `{
  "model": "test",
  "maxTokens": 8192
}`
      setPreviewContent(content)

      const { previewContent } = useConfigStore.getState()
      expect(previewContent).toBe(content)
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

    it('setStatus should handle various status messages', () => {
      const { setStatus } = useConfigStore.getState()

      const statuses = [
        'Ready',
        'Saving...',
        'Saved successfully',
        'Error: Invalid configuration',
        'Validating...',
        'Generating...',
        'Exporting...',
      ]

      for (const s of statuses) {
        setStatus(s)
        expect(useConfigStore.getState().status).toBe(s)
      }
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

    it('setSavedConfigs should update codex configs', () => {
      const { setSavedConfigs } = useConfigStore.getState()
      setSavedConfigs('codex', ['codex-config1'])

      const { savedConfigs } = useConfigStore.getState()
      expect(savedConfigs.codex).toEqual(['codex-config1'])
    })

    it('setSavedConfigs should update gemini configs', () => {
      const { setSavedConfigs } = useConfigStore.getState()
      setSavedConfigs('gemini', ['gemini-config1', 'gemini-config2', 'gemini-config3'])

      const { savedConfigs } = useConfigStore.getState()
      expect(savedConfigs.gemini).toEqual(['gemini-config1', 'gemini-config2', 'gemini-config3'])
    })

    it('setSavedConfigs should not affect other tools', () => {
      const { setSavedConfigs } = useConfigStore.getState()
      setSavedConfigs('claude', ['claude-config'])
      setSavedConfigs('codex', ['codex-config'])

      const { savedConfigs } = useConfigStore.getState()
      expect(savedConfigs.claude).toEqual(['claude-config'])
      expect(savedConfigs.codex).toEqual(['codex-config'])
      expect(savedConfigs.gemini).toEqual([])
      expect(savedConfigs.picoclaw).toEqual([])
      expect(savedConfigs.nullclaw).toEqual([])
      expect(savedConfigs.zeroclaw).toEqual([])
      expect(savedConfigs.openclaw).toEqual([])
    })
  })

  // === Edge Cases ===
  describe('edge cases', () => {
    it('should handle rapid tool switches', () => {
      const { setActiveTool } = useConfigStore.getState()

      for (let i = 0; i < 100; i++) {
        setActiveTool(i % 2 === 0 ? 'claude' : 'codex')
      }

      // Last iteration i=99 is odd, so final value is 'codex'
      expect(useConfigStore.getState().activeTool).toBe('codex')
    })

    it('should handle concurrent tool and status updates', () => {
      const { setActiveTool, setStatus, setPreviewContent } = useConfigStore.getState()

      setActiveTool('claude')
      setStatus('Saving...')
      setPreviewContent('preview content')
      setActiveTool('codex')
      setStatus('Ready')

      const state = useConfigStore.getState()
      expect(state.activeTool).toBe('codex')
      expect(state.status).toBe('Ready')
      expect(state.previewContent).toBe('preview content')
    })
  })
})
