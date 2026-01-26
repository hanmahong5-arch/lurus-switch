import { create } from 'zustand'

// Claude Code configuration types
export interface ClaudePermissions {
  allowBash?: boolean
  allowRead?: boolean
  allowWrite?: boolean
  allowWebFetch?: boolean
  trustedDirectories?: string[]
  allowedBashCommands?: string[]
  deniedBashCommands?: string[]
}

export interface MCPServer {
  command: string
  args?: string[]
  env?: Record<string, string>
}

export interface SandboxMount {
  source: string
  destination: string
  readOnly?: boolean
}

export interface ClaudeSandbox {
  enabled?: boolean
  type?: string
  dockerImage?: string
  mounts?: SandboxMount[]
}

export interface ClaudeAdvanced {
  verbose?: boolean
  disableTelemetry?: boolean
  apiEndpoint?: string
  timeout?: number
  experimentalFeatures?: boolean
}

export interface ClaudeConfig {
  model?: string
  customInstructions?: string
  apiKey?: string
  maxTokens?: number
  permissions?: ClaudePermissions
  mcpServers?: Record<string, MCPServer>
  sandbox?: ClaudeSandbox
  advanced?: ClaudeAdvanced
}

// Codex configuration types
export interface CodexProvider {
  type?: string
  baseUrl?: string
  azureDeployment?: string
  azureApiVersion?: string
}

export interface CodexFileAccess {
  allowedDirs?: string[]
  deniedPatterns?: string[]
  readOnlyDirs?: string[]
}

export interface CodexCommandExecution {
  enabled: boolean
  allowedCommands?: string[]
  deniedCommands?: string[]
}

export interface CodexSecurity {
  networkAccess: string
  fileAccess?: CodexFileAccess
  commandExecution?: CodexCommandExecution
}

export interface CodexMCPServer {
  name: string
  command: string
  args?: string[]
  env?: Record<string, string>
}

export interface CodexMCP {
  enabled: boolean
  servers?: CodexMCPServer[]
}

export interface CodexSandbox {
  enabled: boolean
  type: string
}

export interface CodexHistory {
  enabled: boolean
  filePath?: string
  maxEntries?: number
}

export interface CodexConfig {
  model: string
  apiKey?: string
  approvalMode: string
  provider?: CodexProvider
  security?: CodexSecurity
  mcp?: CodexMCP
  sandbox?: CodexSandbox
  history?: CodexHistory
}

// Gemini configuration types
export interface GeminiAuth {
  type: string
  oauthClientId?: string
  serviceAccountPath?: string
}

export interface GeminiBehavior {
  sandbox: boolean
  autoApprove?: string[]
  yoloMode: boolean
  maxFileSize?: number
  allowedExtensions?: string[]
}

export interface GeminiInstructions {
  projectDescription?: string
  techStack?: string
  codeStyle?: string
  customRules?: string[]
  fileStructure?: string
  testingGuidelines?: string
}

export interface GeminiDisplay {
  theme: string
  syntaxHighlight: boolean
  markdownRender: boolean
}

export interface GeminiConfig {
  model: string
  apiKey?: string
  projectId?: string
  auth?: GeminiAuth
  behavior?: GeminiBehavior
  instructions?: GeminiInstructions
  display?: GeminiDisplay
}

// Store state types
interface ConfigState {
  // Current active tool
  activeTool: 'claude' | 'codex' | 'gemini'
  setActiveTool: (tool: 'claude' | 'codex' | 'gemini') => void

  // Claude config
  claudeConfig: ClaudeConfig
  setClaudeConfig: (config: ClaudeConfig) => void
  updateClaudeConfig: (updates: Partial<ClaudeConfig>) => void

  // Codex config
  codexConfig: CodexConfig
  setCodexConfig: (config: CodexConfig) => void
  updateCodexConfig: (updates: Partial<CodexConfig>) => void

  // Gemini config
  geminiConfig: GeminiConfig
  setGeminiConfig: (config: GeminiConfig) => void
  updateGeminiConfig: (updates: Partial<GeminiConfig>) => void

  // Preview
  previewContent: string
  setPreviewContent: (content: string) => void

  // Status
  status: string
  setStatus: (status: string) => void

  // Saved configs
  savedConfigs: Record<string, string[]>
  setSavedConfigs: (tool: string, configs: string[]) => void
}

export const useConfigStore = create<ConfigState>((set) => ({
  // Active tool
  activeTool: 'claude',
  setActiveTool: (tool) => set({ activeTool: tool }),

  // Claude config
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
  setClaudeConfig: (config) => set({ claudeConfig: config }),
  updateClaudeConfig: (updates) =>
    set((state) => ({
      claudeConfig: { ...state.claudeConfig, ...updates },
    })),

  // Codex config
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
  setCodexConfig: (config) => set({ codexConfig: config }),
  updateCodexConfig: (updates) =>
    set((state) => ({
      codexConfig: { ...state.codexConfig, ...updates },
    })),

  // Gemini config
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
  setGeminiConfig: (config) => set({ geminiConfig: config }),
  updateGeminiConfig: (updates) =>
    set((state) => ({
      geminiConfig: { ...state.geminiConfig, ...updates },
    })),

  // Preview
  previewContent: '',
  setPreviewContent: (content) => set({ previewContent: content }),

  // Status
  status: 'Ready',
  setStatus: (status) => set({ status: status }),

  // Saved configs
  savedConfigs: {
    claude: [],
    codex: [],
    gemini: [],
  },
  setSavedConfigs: (tool, configs) =>
    set((state) => ({
      savedConfigs: { ...state.savedConfigs, [tool]: configs },
    })),
}))
