import { describe, it, expect, beforeEach } from 'vitest'
import { useConfigStore } from './configStore'

describe('configStore', () => {
  // Reset store before each test
  beforeEach(() => {
    useConfigStore.setState({
      activeTool: 'claude',
      claudeConfig: {
        model: 'claude-sonnet-4-20250514',
        maxTokens: 8192,
        permissions: {
          allowBash: true,
          allowRead: true,
          allowWrite: true,
          allowWebFetch: false,
        },
        sandbox: {
          enabled: false,
          type: 'none',
        },
        advanced: {
          verbose: false,
          disableTelemetry: false,
          timeout: 300,
        },
      },
      codexConfig: {
        model: 'o4-mini',
        approvalMode: 'suggest',
        provider: { type: 'openai' },
        security: {
          networkAccess: 'local',
          fileAccess: {
            allowedDirs: ['.'],
            deniedPatterns: ['**/.env', '**/*.key', '**/secrets/**'],
          },
          commandExecution: { enabled: true },
        },
        mcp: { enabled: false },
        sandbox: { enabled: true, type: 'none' },
        history: { enabled: true, maxEntries: 1000 },
      },
      geminiConfig: {
        model: 'gemini-2.0-flash',
        auth: { type: 'api_key' },
        behavior: {
          sandbox: false,
          yoloMode: false,
          maxFileSize: 10 * 1024 * 1024,
        },
        instructions: { customRules: [] },
        display: {
          theme: 'auto',
          syntaxHighlight: true,
          markdownRender: true,
        },
      },
      previewContent: '',
      status: 'Ready',
      savedConfigs: {
        claude: [],
        codex: [],
        gemini: [],
      },
    })
  })

  // === Initial State Tests ===
  describe('initial state', () => {
    it('should have claude as the default active tool', () => {
      const { activeTool } = useConfigStore.getState()
      expect(activeTool).toBe('claude')
    })

    it('should have default claude config', () => {
      const { claudeConfig } = useConfigStore.getState()
      expect(claudeConfig.model).toBe('claude-sonnet-4-20250514')
      expect(claudeConfig.maxTokens).toBe(8192)
    })

    it('should have default codex config', () => {
      const { codexConfig } = useConfigStore.getState()
      expect(codexConfig.model).toBe('o4-mini')
      expect(codexConfig.approvalMode).toBe('suggest')
    })

    it('should have default gemini config', () => {
      const { geminiConfig } = useConfigStore.getState()
      expect(geminiConfig.model).toBe('gemini-2.0-flash')
      expect(geminiConfig.auth?.type).toBe('api_key')
    })

    it('should have Ready status', () => {
      const { status } = useConfigStore.getState()
      expect(status).toBe('Ready')
    })

    it('should have empty preview content', () => {
      const { previewContent } = useConfigStore.getState()
      expect(previewContent).toBe('')
    })

    it('should have empty saved configs', () => {
      const { savedConfigs } = useConfigStore.getState()
      expect(savedConfigs.claude).toEqual([])
      expect(savedConfigs.codex).toEqual([])
      expect(savedConfigs.gemini).toEqual([])
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

    it('should switch back to claude', () => {
      const { setActiveTool } = useConfigStore.getState()
      setActiveTool('codex')
      setActiveTool('claude')
      expect(useConfigStore.getState().activeTool).toBe('claude')
    })
  })

  // === Claude Config Tests ===
  describe('claude config', () => {
    it('setClaudeConfig should replace entire config', () => {
      const { setClaudeConfig } = useConfigStore.getState()
      const newConfig = {
        model: 'claude-3-opus',
        maxTokens: 16384,
      }
      setClaudeConfig(newConfig)

      const { claudeConfig } = useConfigStore.getState()
      expect(claudeConfig.model).toBe('claude-3-opus')
      expect(claudeConfig.maxTokens).toBe(16384)
      // Other fields should be undefined since we replaced
      expect(claudeConfig.permissions).toBeUndefined()
    })

    it('updateClaudeConfig should merge updates', () => {
      const { updateClaudeConfig } = useConfigStore.getState()
      updateClaudeConfig({ model: 'claude-3-opus' })

      const { claudeConfig } = useConfigStore.getState()
      expect(claudeConfig.model).toBe('claude-3-opus')
      // Other fields should be preserved
      expect(claudeConfig.maxTokens).toBe(8192)
    })

    it('updateClaudeConfig should update permissions', () => {
      const { updateClaudeConfig } = useConfigStore.getState()
      updateClaudeConfig({
        permissions: {
          allowBash: false,
          allowRead: true,
          allowWrite: false,
          allowWebFetch: true,
        },
      })

      const { claudeConfig } = useConfigStore.getState()
      expect(claudeConfig.permissions?.allowBash).toBe(false)
      expect(claudeConfig.permissions?.allowWebFetch).toBe(true)
    })

    it('updateClaudeConfig should update sandbox settings', () => {
      const { updateClaudeConfig } = useConfigStore.getState()
      updateClaudeConfig({
        sandbox: {
          enabled: true,
          type: 'docker',
          dockerImage: 'ubuntu:22.04',
        },
      })

      const { claudeConfig } = useConfigStore.getState()
      expect(claudeConfig.sandbox?.enabled).toBe(true)
      expect(claudeConfig.sandbox?.type).toBe('docker')
      expect(claudeConfig.sandbox?.dockerImage).toBe('ubuntu:22.04')
    })

    it('updateClaudeConfig should update advanced settings', () => {
      const { updateClaudeConfig } = useConfigStore.getState()
      updateClaudeConfig({
        advanced: {
          verbose: true,
          disableTelemetry: true,
          timeout: 600,
        },
      })

      const { claudeConfig } = useConfigStore.getState()
      expect(claudeConfig.advanced?.verbose).toBe(true)
      expect(claudeConfig.advanced?.disableTelemetry).toBe(true)
      expect(claudeConfig.advanced?.timeout).toBe(600)
    })

    it('updateClaudeConfig should update mcpServers', () => {
      const { updateClaudeConfig } = useConfigStore.getState()
      updateClaudeConfig({
        mcpServers: {
          fs: {
            command: 'mcp-server-fs',
            args: ['--root', '/'],
          },
        },
      })

      const { claudeConfig } = useConfigStore.getState()
      expect(claudeConfig.mcpServers?.fs).toBeDefined()
      expect(claudeConfig.mcpServers?.fs.command).toBe('mcp-server-fs')
    })

    it('updateClaudeConfig should update customInstructions', () => {
      const { updateClaudeConfig } = useConfigStore.getState()
      updateClaudeConfig({ customInstructions: 'Be helpful and concise' })

      const { claudeConfig } = useConfigStore.getState()
      expect(claudeConfig.customInstructions).toBe('Be helpful and concise')
    })

    it('updateClaudeConfig should update apiKey', () => {
      const { updateClaudeConfig } = useConfigStore.getState()
      updateClaudeConfig({ apiKey: 'sk-ant-test123' })

      const { claudeConfig } = useConfigStore.getState()
      expect(claudeConfig.apiKey).toBe('sk-ant-test123')
    })
  })

  // === Codex Config Tests ===
  describe('codex config', () => {
    it('setCodexConfig should replace entire config', () => {
      const { setCodexConfig } = useConfigStore.getState()
      const newConfig = {
        model: 'gpt-4',
        approvalMode: 'full-auto',
      }
      setCodexConfig(newConfig)

      const { codexConfig } = useConfigStore.getState()
      expect(codexConfig.model).toBe('gpt-4')
      expect(codexConfig.approvalMode).toBe('full-auto')
    })

    it('updateCodexConfig should merge updates', () => {
      const { updateCodexConfig } = useConfigStore.getState()
      updateCodexConfig({ model: 'gpt-4-turbo' })

      const { codexConfig } = useConfigStore.getState()
      expect(codexConfig.model).toBe('gpt-4-turbo')
      expect(codexConfig.approvalMode).toBe('suggest')
    })

    it('updateCodexConfig should update provider', () => {
      const { updateCodexConfig } = useConfigStore.getState()
      updateCodexConfig({
        provider: {
          type: 'azure',
          baseUrl: 'https://my.azure.com',
          azureDeployment: 'gpt-4',
        },
      })

      const { codexConfig } = useConfigStore.getState()
      expect(codexConfig.provider?.type).toBe('azure')
      expect(codexConfig.provider?.baseUrl).toBe('https://my.azure.com')
    })

    it('updateCodexConfig should update security', () => {
      const { updateCodexConfig } = useConfigStore.getState()
      updateCodexConfig({
        security: {
          networkAccess: 'full',
          fileAccess: {
            allowedDirs: ['.', '/tmp'],
          },
          commandExecution: { enabled: false },
        },
      })

      const { codexConfig } = useConfigStore.getState()
      expect(codexConfig.security?.networkAccess).toBe('full')
      expect(codexConfig.security?.commandExecution?.enabled).toBe(false)
    })

    it('updateCodexConfig should update mcp', () => {
      const { updateCodexConfig } = useConfigStore.getState()
      updateCodexConfig({
        mcp: {
          enabled: true,
          servers: [
            { name: 'fs', command: 'mcp-fs' },
          ],
        },
      })

      const { codexConfig } = useConfigStore.getState()
      expect(codexConfig.mcp?.enabled).toBe(true)
      expect(codexConfig.mcp?.servers?.length).toBe(1)
    })

    it('updateCodexConfig should update sandbox', () => {
      const { updateCodexConfig } = useConfigStore.getState()
      updateCodexConfig({
        sandbox: {
          enabled: true,
          type: 'landlock',
        },
      })

      const { codexConfig } = useConfigStore.getState()
      expect(codexConfig.sandbox?.enabled).toBe(true)
      expect(codexConfig.sandbox?.type).toBe('landlock')
    })

    it('updateCodexConfig should update history', () => {
      const { updateCodexConfig } = useConfigStore.getState()
      updateCodexConfig({
        history: {
          enabled: false,
          maxEntries: 500,
        },
      })

      const { codexConfig } = useConfigStore.getState()
      expect(codexConfig.history?.enabled).toBe(false)
      expect(codexConfig.history?.maxEntries).toBe(500)
    })
  })

  // === Gemini Config Tests ===
  describe('gemini config', () => {
    it('setGeminiConfig should replace entire config', () => {
      const { setGeminiConfig } = useConfigStore.getState()
      const newConfig = {
        model: 'gemini-1.5-pro',
      }
      setGeminiConfig(newConfig)

      const { geminiConfig } = useConfigStore.getState()
      expect(geminiConfig.model).toBe('gemini-1.5-pro')
    })

    it('updateGeminiConfig should merge updates', () => {
      const { updateGeminiConfig } = useConfigStore.getState()
      updateGeminiConfig({ model: 'gemini-2.0-pro' })

      const { geminiConfig } = useConfigStore.getState()
      expect(geminiConfig.model).toBe('gemini-2.0-pro')
      expect(geminiConfig.auth?.type).toBe('api_key')
    })

    it('updateGeminiConfig should update auth', () => {
      const { updateGeminiConfig } = useConfigStore.getState()
      updateGeminiConfig({
        auth: {
          type: 'oauth',
          oauthClientId: 'client-123',
        },
      })

      const { geminiConfig } = useConfigStore.getState()
      expect(geminiConfig.auth?.type).toBe('oauth')
      expect(geminiConfig.auth?.oauthClientId).toBe('client-123')
    })

    it('updateGeminiConfig should update behavior', () => {
      const { updateGeminiConfig } = useConfigStore.getState()
      updateGeminiConfig({
        behavior: {
          sandbox: true,
          yoloMode: true,
          maxFileSize: 50 * 1024 * 1024,
        },
      })

      const { geminiConfig } = useConfigStore.getState()
      expect(geminiConfig.behavior?.sandbox).toBe(true)
      expect(geminiConfig.behavior?.yoloMode).toBe(true)
      expect(geminiConfig.behavior?.maxFileSize).toBe(50 * 1024 * 1024)
    })

    it('updateGeminiConfig should update instructions', () => {
      const { updateGeminiConfig } = useConfigStore.getState()
      updateGeminiConfig({
        instructions: {
          projectDescription: 'My awesome project',
          techStack: 'React, Node.js',
          customRules: ['Rule 1', 'Rule 2'],
        },
      })

      const { geminiConfig } = useConfigStore.getState()
      expect(geminiConfig.instructions?.projectDescription).toBe('My awesome project')
      expect(geminiConfig.instructions?.customRules?.length).toBe(2)
    })

    it('updateGeminiConfig should update display', () => {
      const { updateGeminiConfig } = useConfigStore.getState()
      updateGeminiConfig({
        display: {
          theme: 'dark',
          syntaxHighlight: false,
          markdownRender: false,
        },
      })

      const { geminiConfig } = useConfigStore.getState()
      expect(geminiConfig.display?.theme).toBe('dark')
      expect(geminiConfig.display?.syntaxHighlight).toBe(false)
    })

    it('updateGeminiConfig should handle customRules array operations', () => {
      const { updateGeminiConfig } = useConfigStore.getState()

      // Add rules
      updateGeminiConfig({
        instructions: {
          customRules: ['Rule 1'],
        },
      })
      expect(useConfigStore.getState().geminiConfig.instructions?.customRules).toEqual(['Rule 1'])

      // Add more rules
      updateGeminiConfig({
        instructions: {
          customRules: ['Rule 1', 'Rule 2', 'Rule 3'],
        },
      })
      expect(useConfigStore.getState().geminiConfig.instructions?.customRules?.length).toBe(3)

      // Clear rules
      updateGeminiConfig({
        instructions: {
          customRules: [],
        },
      })
      expect(useConfigStore.getState().geminiConfig.instructions?.customRules?.length).toBe(0)
    })
  })

  // === Preview Tests ===
  describe('preview', () => {
    it('setPreviewContent should update preview', () => {
      const { setPreviewContent } = useConfigStore.getState()
      setPreviewContent('{"model": "claude-3-opus"}')

      const { previewContent } = useConfigStore.getState()
      expect(previewContent).toBe('{"model": "claude-3-opus"}')
    })

    it('setPreviewContent should handle multiline content', () => {
      const { setPreviewContent } = useConfigStore.getState()
      const content = `{
  "model": "claude-3-opus",
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
    })
  })

  // === Edge Cases ===
  describe('edge cases', () => {
    it('should handle rapid updates', () => {
      const { updateClaudeConfig } = useConfigStore.getState()

      for (let i = 0; i < 100; i++) {
        updateClaudeConfig({ maxTokens: i })
      }

      const { claudeConfig } = useConfigStore.getState()
      expect(claudeConfig.maxTokens).toBe(99)
    })

    it('should handle undefined values in updates', () => {
      const { updateClaudeConfig } = useConfigStore.getState()
      updateClaudeConfig({ customInstructions: undefined })

      const { claudeConfig } = useConfigStore.getState()
      // undefined should not override existing value
      expect(claudeConfig.maxTokens).toBe(8192)
    })

    it('should handle concurrent tool and config updates', () => {
      const { setActiveTool, updateClaudeConfig, updateCodexConfig } = useConfigStore.getState()

      setActiveTool('claude')
      updateClaudeConfig({ model: 'claude-3-opus' })
      setActiveTool('codex')
      updateCodexConfig({ model: 'gpt-4' })
      setActiveTool('claude')

      const state = useConfigStore.getState()
      expect(state.activeTool).toBe('claude')
      expect(state.claudeConfig.model).toBe('claude-3-opus')
      expect(state.codexConfig.model).toBe('gpt-4')
    })
  })
})
