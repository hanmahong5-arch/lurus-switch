import { useEffect, useState } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import * as Switch from '@radix-ui/react-switch'
import * as Select from '@radix-ui/react-select'
import { Save, Download, ChevronDown, Check } from 'lucide-react'
import { useConfigStore } from '../stores/configStore'
import { ConfigPreview } from '../components/ConfigPreview'
import {
  GetDefaultCodexConfig,
  GenerateCodexConfig,
  ExportCodexConfig,
  SaveCodexConfig,
  ValidateCodexConfig,
} from '../../wailsjs/go/main/App'

const codexModels = [
  { value: 'o4-mini', label: 'o4-mini (Recommended)' },
  { value: 'o3', label: 'o3' },
  { value: 'gpt-4o', label: 'GPT-4o' },
  { value: 'gpt-4-turbo', label: 'GPT-4 Turbo' },
]

const approvalModes = [
  { value: 'suggest', label: 'Suggest - Requires approval for all actions' },
  { value: 'auto-edit', label: 'Auto-Edit - Auto-approve file edits' },
  { value: 'full-auto', label: 'Full-Auto - No approvals needed' },
]

const networkAccessOptions = [
  { value: 'off', label: 'Off - No network access' },
  { value: 'local', label: 'Local - localhost only' },
  { value: 'full', label: 'Full - All network access' },
]

export function CodexPage() {
  const { codexConfig, updateCodexConfig, setStatus } = useConfigStore()
  const [preview, setPreview] = useState('')
  const [configName, setConfigName] = useState('default')

  useEffect(() => {
    GenerateCodexConfig(codexConfig as any)
      .then(setPreview)
      .catch((err) => console.error('Failed to generate preview:', err))
  }, [codexConfig])

  useEffect(() => {
    GetDefaultCodexConfig()
      .then((config) => {
        updateCodexConfig(config as any)
      })
      .catch(console.error)
  }, [])

  const handleSave = async () => {
    try {
      setStatus('Saving...')
      await SaveCodexConfig(configName, codexConfig as any)
      setStatus('Saved successfully')
    } catch (err) {
      setStatus(`Error: ${err}`)
    }
  }

  const handleExport = async () => {
    try {
      setStatus('Exporting...')
      const result = await ExportCodexConfig(codexConfig as any)
      setStatus(`Exported to: ${result}`)
    } catch (err) {
      setStatus(`Error: ${err}`)
    }
  }

  const handleValidate = async () => {
    try {
      const result = await ValidateCodexConfig(codexConfig as any)
      if (result.valid) {
        setStatus('Configuration is valid')
      } else {
        setStatus(`Validation errors: ${result.errors?.map((e: any) => e.message).join(', ')}`)
      }
    } catch (err) {
      setStatus(`Error: ${err}`)
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="p-4 border-b border-border wails-drag">
        <h2 className="text-xl font-semibold">Codex Configuration</h2>
        <p className="text-sm text-muted-foreground">Configure settings for OpenAI Codex CLI</p>
      </div>

      <div className="flex-1 overflow-auto p-4">
        <Tabs.Root defaultValue="model" className="flex flex-col gap-4">
          <Tabs.List className="flex gap-1 border-b border-border">
            <Tabs.Trigger
              value="model"
              className="px-4 py-2 text-sm font-medium text-muted-foreground data-[state=active]:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary -mb-px"
            >
              Model
            </Tabs.Trigger>
            <Tabs.Trigger
              value="provider"
              className="px-4 py-2 text-sm font-medium text-muted-foreground data-[state=active]:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary -mb-px"
            >
              Provider
            </Tabs.Trigger>
            <Tabs.Trigger
              value="security"
              className="px-4 py-2 text-sm font-medium text-muted-foreground data-[state=active]:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary -mb-px"
            >
              Security
            </Tabs.Trigger>
            <Tabs.Trigger
              value="mcp"
              className="px-4 py-2 text-sm font-medium text-muted-foreground data-[state=active]:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary -mb-px"
            >
              MCP
            </Tabs.Trigger>
          </Tabs.List>

          {/* Model Tab */}
          <Tabs.Content value="model" className="space-y-4">
            <div className="grid gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Model</label>
                <Select.Root
                  value={codexConfig.model}
                  onValueChange={(value) => updateCodexConfig({ model: value })}
                >
                  <Select.Trigger className="flex items-center justify-between w-full px-3 py-2 text-sm border rounded-md bg-background border-input hover:bg-accent">
                    <Select.Value placeholder="Select model" />
                    <Select.Icon>
                      <ChevronDown className="h-4 w-4" />
                    </Select.Icon>
                  </Select.Trigger>
                  <Select.Portal>
                    <Select.Content className="bg-popover border border-border rounded-md shadow-md">
                      <Select.Viewport className="p-1">
                        {codexModels.map((model) => (
                          <Select.Item
                            key={model.value}
                            value={model.value}
                            className="flex items-center gap-2 px-2 py-1.5 text-sm rounded cursor-pointer outline-none hover:bg-accent data-[highlighted]:bg-accent"
                          >
                            <Select.ItemIndicator>
                              <Check className="h-4 w-4" />
                            </Select.ItemIndicator>
                            <Select.ItemText>{model.label}</Select.ItemText>
                          </Select.Item>
                        ))}
                      </Select.Viewport>
                    </Select.Content>
                  </Select.Portal>
                </Select.Root>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">Approval Mode</label>
                <Select.Root
                  value={codexConfig.approvalMode}
                  onValueChange={(value) => updateCodexConfig({ approvalMode: value })}
                >
                  <Select.Trigger className="flex items-center justify-between w-full px-3 py-2 text-sm border rounded-md bg-background border-input hover:bg-accent">
                    <Select.Value placeholder="Select mode" />
                    <Select.Icon>
                      <ChevronDown className="h-4 w-4" />
                    </Select.Icon>
                  </Select.Trigger>
                  <Select.Portal>
                    <Select.Content className="bg-popover border border-border rounded-md shadow-md">
                      <Select.Viewport className="p-1">
                        {approvalModes.map((mode) => (
                          <Select.Item
                            key={mode.value}
                            value={mode.value}
                            className="flex items-center gap-2 px-2 py-1.5 text-sm rounded cursor-pointer outline-none hover:bg-accent data-[highlighted]:bg-accent"
                          >
                            <Select.ItemIndicator>
                              <Check className="h-4 w-4" />
                            </Select.ItemIndicator>
                            <Select.ItemText>{mode.label}</Select.ItemText>
                          </Select.Item>
                        ))}
                      </Select.Viewport>
                    </Select.Content>
                  </Select.Portal>
                </Select.Root>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">API Key (optional)</label>
                <input
                  type="password"
                  value={codexConfig.apiKey || ''}
                  onChange={(e) => updateCodexConfig({ apiKey: e.target.value })}
                  placeholder="sk-..."
                  className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input"
                />
              </div>
            </div>
          </Tabs.Content>

          {/* Provider Tab */}
          <Tabs.Content value="provider" className="space-y-4">
            <div className="grid gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Provider Type</label>
                <Select.Root
                  value={codexConfig.provider?.type || 'openai'}
                  onValueChange={(value) =>
                    updateCodexConfig({ provider: { ...codexConfig.provider, type: value } })
                  }
                >
                  <Select.Trigger className="flex items-center justify-between w-full px-3 py-2 text-sm border rounded-md bg-background border-input hover:bg-accent">
                    <Select.Value />
                    <Select.Icon>
                      <ChevronDown className="h-4 w-4" />
                    </Select.Icon>
                  </Select.Trigger>
                  <Select.Portal>
                    <Select.Content className="bg-popover border border-border rounded-md shadow-md">
                      <Select.Viewport className="p-1">
                        {['openai', 'azure', 'openrouter', 'custom'].map((provider) => (
                          <Select.Item
                            key={provider}
                            value={provider}
                            className="flex items-center gap-2 px-2 py-1.5 text-sm rounded cursor-pointer outline-none hover:bg-accent"
                          >
                            <Select.ItemText>{provider.charAt(0).toUpperCase() + provider.slice(1)}</Select.ItemText>
                          </Select.Item>
                        ))}
                      </Select.Viewport>
                    </Select.Content>
                  </Select.Portal>
                </Select.Root>
              </div>

              {codexConfig.provider?.type === 'azure' && (
                <>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">Azure Deployment Name</label>
                    <input
                      type="text"
                      value={codexConfig.provider?.azureDeployment || ''}
                      onChange={(e) =>
                        updateCodexConfig({
                          provider: { ...codexConfig.provider, azureDeployment: e.target.value },
                        })
                      }
                      className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input"
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">Azure API Version</label>
                    <input
                      type="text"
                      value={codexConfig.provider?.azureApiVersion || '2024-02-15-preview'}
                      onChange={(e) =>
                        updateCodexConfig({
                          provider: { ...codexConfig.provider, azureApiVersion: e.target.value },
                        })
                      }
                      className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input"
                    />
                  </div>
                </>
              )}

              {(codexConfig.provider?.type === 'azure' || codexConfig.provider?.type === 'custom') && (
                <div className="space-y-2">
                  <label className="text-sm font-medium">Base URL</label>
                  <input
                    type="text"
                    value={codexConfig.provider?.baseUrl || ''}
                    onChange={(e) =>
                      updateCodexConfig({
                        provider: { ...codexConfig.provider, baseUrl: e.target.value },
                      })
                    }
                    placeholder="https://your-endpoint.openai.azure.com"
                    className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input"
                  />
                </div>
              )}
            </div>
          </Tabs.Content>

          {/* Security Tab */}
          <Tabs.Content value="security" className="space-y-4">
            <div className="grid gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Network Access</label>
                <Select.Root
                  value={codexConfig.security?.networkAccess || 'local'}
                  onValueChange={(value) =>
                    updateCodexConfig({ security: { ...codexConfig.security!, networkAccess: value } })
                  }
                >
                  <Select.Trigger className="flex items-center justify-between w-full px-3 py-2 text-sm border rounded-md bg-background border-input hover:bg-accent">
                    <Select.Value />
                    <Select.Icon>
                      <ChevronDown className="h-4 w-4" />
                    </Select.Icon>
                  </Select.Trigger>
                  <Select.Portal>
                    <Select.Content className="bg-popover border border-border rounded-md shadow-md">
                      <Select.Viewport className="p-1">
                        {networkAccessOptions.map((opt) => (
                          <Select.Item
                            key={opt.value}
                            value={opt.value}
                            className="flex items-center gap-2 px-2 py-1.5 text-sm rounded cursor-pointer outline-none hover:bg-accent"
                          >
                            <Select.ItemText>{opt.label}</Select.ItemText>
                          </Select.Item>
                        ))}
                      </Select.Viewport>
                    </Select.Content>
                  </Select.Portal>
                </Select.Root>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium">Command Execution</label>
                  <p className="text-xs text-muted-foreground">Allow running shell commands</p>
                </div>
                <Switch.Root
                  checked={codexConfig.security?.commandExecution?.enabled ?? true}
                  onCheckedChange={(checked) =>
                    updateCodexConfig({
                      security: {
                        ...codexConfig.security!,
                        commandExecution: { ...codexConfig.security?.commandExecution, enabled: checked },
                      },
                    })
                  }
                  className="w-11 h-6 bg-muted rounded-full relative data-[state=checked]:bg-primary"
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform translate-x-0.5 data-[state=checked]:translate-x-5" />
                </Switch.Root>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium">Enable Sandbox</label>
                  <p className="text-xs text-muted-foreground">Isolate command execution</p>
                </div>
                <Switch.Root
                  checked={codexConfig.sandbox?.enabled ?? true}
                  onCheckedChange={(checked) =>
                    updateCodexConfig({
                      sandbox: { ...codexConfig.sandbox!, enabled: checked },
                    })
                  }
                  className="w-11 h-6 bg-muted rounded-full relative data-[state=checked]:bg-primary"
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform translate-x-0.5 data-[state=checked]:translate-x-5" />
                </Switch.Root>
              </div>
            </div>
          </Tabs.Content>

          {/* MCP Tab */}
          <Tabs.Content value="mcp" className="space-y-4">
            <div className="grid gap-4">
              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium">Enable MCP</label>
                  <p className="text-xs text-muted-foreground">Model Context Protocol servers</p>
                </div>
                <Switch.Root
                  checked={codexConfig.mcp?.enabled ?? false}
                  onCheckedChange={(checked) =>
                    updateCodexConfig({
                      mcp: { ...codexConfig.mcp!, enabled: checked },
                    })
                  }
                  className="w-11 h-6 bg-muted rounded-full relative data-[state=checked]:bg-primary"
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform translate-x-0.5 data-[state=checked]:translate-x-5" />
                </Switch.Root>
              </div>

              {codexConfig.mcp?.enabled && (
                <div className="p-4 bg-muted/50 rounded-md">
                  <p className="text-sm text-muted-foreground">
                    MCP server configuration will be added in a future update.
                  </p>
                </div>
              )}
            </div>
          </Tabs.Content>
        </Tabs.Root>

        <div className="mt-6">
          <ConfigPreview content={preview} language="toml" />
        </div>
      </div>

      <div className="p-4 border-t border-border flex items-center gap-2">
        <input
          type="text"
          value={configName}
          onChange={(e) => setConfigName(e.target.value)}
          placeholder="Config name"
          className="px-3 py-2 text-sm border rounded-md bg-background border-input w-40"
        />
        <button
          onClick={handleSave}
          className="flex items-center gap-2 px-4 py-2 text-sm font-medium bg-secondary text-secondary-foreground rounded-md hover:bg-secondary/80"
        >
          <Save className="h-4 w-4" />
          Save Config
        </button>
        <button
          onClick={handleExport}
          className="flex items-center gap-2 px-4 py-2 text-sm font-medium bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
        >
          <Download className="h-4 w-4" />
          Export
        </button>
        <button
          onClick={handleValidate}
          className="flex items-center gap-2 px-4 py-2 text-sm font-medium border border-input rounded-md hover:bg-accent"
        >
          Validate
        </button>
      </div>
    </div>
  )
}
