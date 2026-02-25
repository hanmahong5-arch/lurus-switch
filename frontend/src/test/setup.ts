import '@testing-library/jest-dom'
import { vi } from 'vitest'

// Mock Wails runtime
vi.mock('../../wailsjs/runtime/runtime', () => ({
  WindowSetTitle: vi.fn(),
  WindowMinimise: vi.fn(),
  WindowMaximise: vi.fn(),
  WindowUnmaximise: vi.fn(),
  WindowClose: vi.fn(),
  WindowShow: vi.fn(),
  WindowHide: vi.fn(),
  WindowCenter: vi.fn(),
  WindowSetSize: vi.fn(),
  WindowSetPosition: vi.fn(),
  WindowSetMinSize: vi.fn(),
  WindowSetMaxSize: vi.fn(),
  WindowToggleMaximise: vi.fn(),
  WindowFullscreen: vi.fn(),
  WindowUnfullscreen: vi.fn(),
  WindowSetBackgroundColour: vi.fn(),
  WindowReload: vi.fn(),
  WindowReloadApp: vi.fn(),
  EventsOn: vi.fn(),
  EventsOff: vi.fn(),
  EventsOnce: vi.fn(),
  EventsOnMultiple: vi.fn(),
  EventsEmit: vi.fn(),
  LogDebug: vi.fn(),
  LogInfo: vi.fn(),
  LogWarning: vi.fn(),
  LogError: vi.fn(),
  LogFatal: vi.fn(),
  LogPrint: vi.fn(),
  LogTrace: vi.fn(),
  BrowserOpenURL: vi.fn(),
  Environment: vi.fn(() => ({
    buildType: 'dev',
    platform: 'windows',
    arch: 'amd64',
  })),
  Quit: vi.fn(),
  Hide: vi.fn(),
  Show: vi.fn(),
  ClipboardGetText: vi.fn(() => Promise.resolve('')),
  ClipboardSetText: vi.fn(() => Promise.resolve(true)),
}))

// Mock Wails Go bindings
vi.mock('../../wailsjs/go/main/App', () => ({
  GetDefaultClaudeConfig: vi.fn(() => Promise.resolve({
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
  })),
  GetDefaultCodexConfig: vi.fn(() => Promise.resolve({
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
  })),
  GetDefaultGeminiConfig: vi.fn(() => Promise.resolve({
    model: 'gemini-2.0-flash',
    auth: { type: 'api_key' },
    behavior: {
      sandbox: false,
      yoloMode: false,
      maxFileSize: 10485760,
    },
    instructions: { customRules: [] },
    display: {
      theme: 'auto',
      syntaxHighlight: true,
      markdownRender: true,
    },
  })),
  GenerateClaudeConfig: vi.fn(() => Promise.resolve('{"model":"claude-sonnet-4"}')),
  GenerateCodexConfig: vi.fn(() => Promise.resolve('[main]\nmodel = "o4-mini"')),
  GenerateGeminiConfig: vi.fn(() => Promise.resolve('# GEMINI.md\n\nConfig file')),
  ValidateClaudeConfig: vi.fn(() => Promise.resolve({ valid: true, errors: [] })),
  ValidateCodexConfig: vi.fn(() => Promise.resolve({ valid: true, errors: [] })),
  ValidateGeminiConfig: vi.fn(() => Promise.resolve({ valid: true, errors: [] })),
  SaveClaudeConfig: vi.fn(() => Promise.resolve()),
  SaveCodexConfig: vi.fn(() => Promise.resolve()),
  SaveGeminiConfig: vi.fn(() => Promise.resolve()),
  LoadClaudeConfig: vi.fn(() => Promise.resolve(null)),
  LoadCodexConfig: vi.fn(() => Promise.resolve(null)),
  LoadGeminiConfig: vi.fn(() => Promise.resolve(null)),
  ListClaudeConfigs: vi.fn(() => Promise.resolve([])),
  ListCodexConfigs: vi.fn(() => Promise.resolve([])),
  ListGeminiConfigs: vi.fn(() => Promise.resolve([])),
  DeleteClaudeConfig: vi.fn(() => Promise.resolve()),
  DeleteCodexConfig: vi.fn(() => Promise.resolve()),
  DeleteGeminiConfig: vi.fn(() => Promise.resolve()),
  ExportClaudeConfig: vi.fn(() => Promise.resolve()),
  ExportCodexConfig: vi.fn(() => Promise.resolve()),
  ExportGeminiConfig: vi.fn(() => Promise.resolve()),
  PackageClaudeConfig: vi.fn(() => Promise.resolve()),
  DownloadCodexBinary: vi.fn(() => Promise.resolve('')),
  GetConfigDir: vi.fn(() => Promise.resolve('')),
  OpenConfigDir: vi.fn(() => Promise.resolve()),
  CheckBunInstalled: vi.fn(() => Promise.resolve(true)),
  CheckNodeInstalled: vi.fn(() => Promise.resolve(true)),
}))

// Mock Monaco Editor
vi.mock('@monaco-editor/react', () => ({
  default: vi.fn(({ value, language }: { value: string; language: string }) => {
    const element = document.createElement('pre')
    element.setAttribute('data-testid', 'monaco-editor')
    element.setAttribute('data-language', language)
    element.textContent = value
    return element
  }),
}))

// Reset mocks before each test
beforeEach(() => {
  vi.clearAllMocks()
})
