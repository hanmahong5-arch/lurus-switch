import { useEffect, useState } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import * as Switch from '@radix-ui/react-switch'
import * as Select from '@radix-ui/react-select'
import { Save, Download, Package, ChevronDown, Check } from 'lucide-react'
import { cn } from '../lib/utils'
import { useConfigStore } from '../stores/configStore'
import { ConfigPreview } from '../components/ConfigPreview'
import {
  GetDefaultClaudeConfig,
  GenerateClaudeConfig,
  ExportClaudeConfig,
  SaveClaudeConfig,
  ValidateClaudeConfig,
} from '../../wailsjs/go/main/App'

const claudeModels = [
  { value: 'claude-sonnet-4-20250514', label: 'Claude Sonnet 4 (Latest)' },
  { value: 'claude-opus-4-20250514', label: 'Claude Opus 4' },
  { value: 'claude-3-5-sonnet-20241022', label: 'Claude 3.5 Sonnet' },
  { value: 'claude-3-5-haiku-20241022', label: 'Claude 3.5 Haiku' },
  { value: 'claude-3-opus-20240229', label: 'Claude 3 Opus' },
]

export function ClaudePage() {
  const { claudeConfig, updateClaudeConfig, setStatus } = useConfigStore()
  const [preview, setPreview] = useState('')
  const [configName, setConfigName] = useState('default')

  // Generate preview whenever config changes
  useEffect(() => {
    GenerateClaudeConfig(claudeConfig as any)
      .then(setPreview)
      .catch((err) => console.error('Failed to generate preview:', err))
  }, [claudeConfig])

  // Load default config on mount
  useEffect(() => {
    GetDefaultClaudeConfig()
      .then((config) => {
        updateClaudeConfig(config as any)
      })
      .catch(console.error)
  }, [])

  const handleSave = async () => {
    try {
      setStatus('Saving...')
      await SaveClaudeConfig(configName, claudeConfig as any)
      setStatus('Saved successfully')
    } catch (err) {
      setStatus(`Error: ${err}`)
    }
  }

  const handleExport = async () => {
    try {
      setStatus('Exporting...')
      const result = await ExportClaudeConfig(claudeConfig as any)
      setStatus(`Exported to: ${result}`)
    } catch (err) {
      setStatus(`Error: ${err}`)
    }
  }

  const handleValidate = async () => {
    try {
      const result = await ValidateClaudeConfig(claudeConfig as any)
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
      {/* Header */}
      <div className="p-4 border-b border-border wails-drag">
        <h2 className="text-xl font-semibold">Claude Code Configuration</h2>
        <p className="text-sm text-muted-foreground">Configure settings for Claude Code CLI</p>
      </div>

      {/* Content */}
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
              value="permissions"
              className="px-4 py-2 text-sm font-medium text-muted-foreground data-[state=active]:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary -mb-px"
            >
              Permissions
            </Tabs.Trigger>
            <Tabs.Trigger
              value="sandbox"
              className="px-4 py-2 text-sm font-medium text-muted-foreground data-[state=active]:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary -mb-px"
            >
              Sandbox
            </Tabs.Trigger>
            <Tabs.Trigger
              value="advanced"
              className="px-4 py-2 text-sm font-medium text-muted-foreground data-[state=active]:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary -mb-px"
            >
              Advanced
            </Tabs.Trigger>
          </Tabs.List>

          {/* Model Tab */}
          <Tabs.Content value="model" className="space-y-4">
            <div className="grid gap-4">
              {/* Model Selection */}
              <div className="space-y-2">
                <label className="text-sm font-medium">Model</label>
                <Select.Root
                  value={claudeConfig.model}
                  onValueChange={(value) => updateClaudeConfig({ model: value })}
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
                        {claudeModels.map((model) => (
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

              {/* Max Tokens */}
              <div className="space-y-2">
                <label className="text-sm font-medium">Max Tokens</label>
                <input
                  type="number"
                  value={claudeConfig.maxTokens || 8192}
                  onChange={(e) => updateClaudeConfig({ maxTokens: parseInt(e.target.value) || 8192 })}
                  className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input"
                />
              </div>

              {/* API Key */}
              <div className="space-y-2">
                <label className="text-sm font-medium">API Key (optional)</label>
                <input
                  type="password"
                  value={claudeConfig.apiKey || ''}
                  onChange={(e) => updateClaudeConfig({ apiKey: e.target.value })}
                  placeholder="sk-ant-..."
                  className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input"
                />
                <p className="text-xs text-muted-foreground">Leave empty to use environment variable</p>
              </div>

              {/* Custom Instructions */}
              <div className="space-y-2">
                <label className="text-sm font-medium">Custom Instructions</label>
                <textarea
                  value={claudeConfig.customInstructions || ''}
                  onChange={(e) => updateClaudeConfig({ customInstructions: e.target.value })}
                  placeholder="Enter custom instructions for Claude..."
                  rows={4}
                  className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input resize-none"
                />
              </div>
            </div>
          </Tabs.Content>

          {/* Permissions Tab */}
          <Tabs.Content value="permissions" className="space-y-4">
            <div className="grid gap-4">
              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium">Allow Bash Commands</label>
                  <p className="text-xs text-muted-foreground">Execute shell commands</p>
                </div>
                <Switch.Root
                  checked={claudeConfig.permissions?.allowBash ?? true}
                  onCheckedChange={(checked) =>
                    updateClaudeConfig({
                      permissions: { ...claudeConfig.permissions, allowBash: checked },
                    })
                  }
                  className="w-11 h-6 bg-muted rounded-full relative data-[state=checked]:bg-primary"
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform translate-x-0.5 data-[state=checked]:translate-x-5" />
                </Switch.Root>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium">Allow File Read</label>
                  <p className="text-xs text-muted-foreground">Read files from disk</p>
                </div>
                <Switch.Root
                  checked={claudeConfig.permissions?.allowRead ?? true}
                  onCheckedChange={(checked) =>
                    updateClaudeConfig({
                      permissions: { ...claudeConfig.permissions, allowRead: checked },
                    })
                  }
                  className="w-11 h-6 bg-muted rounded-full relative data-[state=checked]:bg-primary"
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform translate-x-0.5 data-[state=checked]:translate-x-5" />
                </Switch.Root>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium">Allow File Write</label>
                  <p className="text-xs text-muted-foreground">Write and modify files</p>
                </div>
                <Switch.Root
                  checked={claudeConfig.permissions?.allowWrite ?? true}
                  onCheckedChange={(checked) =>
                    updateClaudeConfig({
                      permissions: { ...claudeConfig.permissions, allowWrite: checked },
                    })
                  }
                  className="w-11 h-6 bg-muted rounded-full relative data-[state=checked]:bg-primary"
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform translate-x-0.5 data-[state=checked]:translate-x-5" />
                </Switch.Root>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium">Allow Web Fetch</label>
                  <p className="text-xs text-muted-foreground">Make HTTP requests</p>
                </div>
                <Switch.Root
                  checked={claudeConfig.permissions?.allowWebFetch ?? false}
                  onCheckedChange={(checked) =>
                    updateClaudeConfig({
                      permissions: { ...claudeConfig.permissions, allowWebFetch: checked },
                    })
                  }
                  className="w-11 h-6 bg-muted rounded-full relative data-[state=checked]:bg-primary"
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform translate-x-0.5 data-[state=checked]:translate-x-5" />
                </Switch.Root>
              </div>
            </div>
          </Tabs.Content>

          {/* Sandbox Tab */}
          <Tabs.Content value="sandbox" className="space-y-4">
            <div className="grid gap-4">
              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium">Enable Sandbox</label>
                  <p className="text-xs text-muted-foreground">Run commands in isolated environment</p>
                </div>
                <Switch.Root
                  checked={claudeConfig.sandbox?.enabled ?? false}
                  onCheckedChange={(checked) =>
                    updateClaudeConfig({
                      sandbox: { ...claudeConfig.sandbox, enabled: checked },
                    })
                  }
                  className="w-11 h-6 bg-muted rounded-full relative data-[state=checked]:bg-primary"
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform translate-x-0.5 data-[state=checked]:translate-x-5" />
                </Switch.Root>
              </div>

              {claudeConfig.sandbox?.enabled && (
                <>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">Sandbox Type</label>
                    <Select.Root
                      value={claudeConfig.sandbox?.type || 'docker'}
                      onValueChange={(value) =>
                        updateClaudeConfig({
                          sandbox: { ...claudeConfig.sandbox, type: value },
                        })
                      }
                    >
                      <Select.Trigger className="flex items-center justify-between w-full px-3 py-2 text-sm border rounded-md bg-background border-input">
                        <Select.Value />
                        <Select.Icon>
                          <ChevronDown className="h-4 w-4" />
                        </Select.Icon>
                      </Select.Trigger>
                      <Select.Portal>
                        <Select.Content className="bg-popover border border-border rounded-md shadow-md">
                          <Select.Viewport className="p-1">
                            <Select.Item
                              value="docker"
                              className="flex items-center gap-2 px-2 py-1.5 text-sm rounded cursor-pointer outline-none hover:bg-accent"
                            >
                              <Select.ItemText>Docker</Select.ItemText>
                            </Select.Item>
                            <Select.Item
                              value="wsl"
                              className="flex items-center gap-2 px-2 py-1.5 text-sm rounded cursor-pointer outline-none hover:bg-accent"
                            >
                              <Select.ItemText>WSL</Select.ItemText>
                            </Select.Item>
                          </Select.Viewport>
                        </Select.Content>
                      </Select.Portal>
                    </Select.Root>
                  </div>

                  {claudeConfig.sandbox?.type === 'docker' && (
                    <div className="space-y-2">
                      <label className="text-sm font-medium">Docker Image</label>
                      <input
                        type="text"
                        value={claudeConfig.sandbox?.dockerImage || ''}
                        onChange={(e) =>
                          updateClaudeConfig({
                            sandbox: { ...claudeConfig.sandbox, dockerImage: e.target.value },
                          })
                        }
                        placeholder="ubuntu:latest"
                        className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input"
                      />
                    </div>
                  )}
                </>
              )}
            </div>
          </Tabs.Content>

          {/* Advanced Tab */}
          <Tabs.Content value="advanced" className="space-y-4">
            <div className="grid gap-4">
              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium">Verbose Logging</label>
                  <p className="text-xs text-muted-foreground">Enable detailed logs</p>
                </div>
                <Switch.Root
                  checked={claudeConfig.advanced?.verbose ?? false}
                  onCheckedChange={(checked) =>
                    updateClaudeConfig({
                      advanced: { ...claudeConfig.advanced, verbose: checked },
                    })
                  }
                  className="w-11 h-6 bg-muted rounded-full relative data-[state=checked]:bg-primary"
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform translate-x-0.5 data-[state=checked]:translate-x-5" />
                </Switch.Root>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium">Disable Telemetry</label>
                  <p className="text-xs text-muted-foreground">Opt out of usage analytics</p>
                </div>
                <Switch.Root
                  checked={claudeConfig.advanced?.disableTelemetry ?? false}
                  onCheckedChange={(checked) =>
                    updateClaudeConfig({
                      advanced: { ...claudeConfig.advanced, disableTelemetry: checked },
                    })
                  }
                  className="w-11 h-6 bg-muted rounded-full relative data-[state=checked]:bg-primary"
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform translate-x-0.5 data-[state=checked]:translate-x-5" />
                </Switch.Root>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">Timeout (seconds)</label>
                <input
                  type="number"
                  value={claudeConfig.advanced?.timeout || 300}
                  onChange={(e) =>
                    updateClaudeConfig({
                      advanced: { ...claudeConfig.advanced, timeout: parseInt(e.target.value) || 300 },
                    })
                  }
                  className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input"
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">Custom API Endpoint</label>
                <input
                  type="text"
                  value={claudeConfig.advanced?.apiEndpoint || ''}
                  onChange={(e) =>
                    updateClaudeConfig({
                      advanced: { ...claudeConfig.advanced, apiEndpoint: e.target.value },
                    })
                  }
                  placeholder="https://api.anthropic.com"
                  className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input"
                />
              </div>
            </div>
          </Tabs.Content>
        </Tabs.Root>

        {/* Preview */}
        <div className="mt-6">
          <ConfigPreview content={preview} language="json" />
        </div>
      </div>

      {/* Actions */}
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
