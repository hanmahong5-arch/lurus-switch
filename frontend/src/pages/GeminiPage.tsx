import { useEffect, useState } from 'react'
import * as Tabs from '@radix-ui/react-tabs'
import * as Switch from '@radix-ui/react-switch'
import * as Select from '@radix-ui/react-select'
import { Save, Download, ChevronDown, Check, Plus, X } from 'lucide-react'
import { useConfigStore } from '../stores/configStore'
import { ConfigPreview } from '../components/ConfigPreview'
import {
  GetDefaultGeminiConfig,
  GenerateGeminiConfig,
  ExportGeminiConfig,
  SaveGeminiConfig,
  ValidateGeminiConfig,
} from '../../wailsjs/go/main/App'

const geminiModels = [
  { value: 'gemini-2.0-flash', label: 'Gemini 2.0 Flash (Latest)' },
  { value: 'gemini-2.0-pro', label: 'Gemini 2.0 Pro' },
  { value: 'gemini-1.5-pro', label: 'Gemini 1.5 Pro' },
  { value: 'gemini-1.5-flash', label: 'Gemini 1.5 Flash' },
]

const authTypes = [
  { value: 'api_key', label: 'API Key' },
  { value: 'oauth', label: 'OAuth' },
  { value: 'adc', label: 'Application Default Credentials' },
]

const themes = [
  { value: 'auto', label: 'Auto (System)' },
  { value: 'dark', label: 'Dark' },
  { value: 'light', label: 'Light' },
]

export function GeminiPage() {
  const { geminiConfig, updateGeminiConfig, setStatus } = useConfigStore()
  const [preview, setPreview] = useState('')
  const [configName, setConfigName] = useState('default')
  const [newRule, setNewRule] = useState('')

  useEffect(() => {
    GenerateGeminiConfig(geminiConfig as any)
      .then(setPreview)
      .catch((err) => console.error('Failed to generate preview:', err))
  }, [geminiConfig])

  useEffect(() => {
    GetDefaultGeminiConfig()
      .then((config) => {
        updateGeminiConfig(config as any)
      })
      .catch(console.error)
  }, [])

  const handleSave = async () => {
    try {
      setStatus('Saving...')
      await SaveGeminiConfig(configName, geminiConfig as any)
      setStatus('Saved successfully')
    } catch (err) {
      setStatus(`Error: ${err}`)
    }
  }

  const handleExport = async () => {
    try {
      setStatus('Exporting...')
      const result = await ExportGeminiConfig(geminiConfig as any)
      setStatus(`Exported: ${result?.join(', ')}`)
    } catch (err) {
      setStatus(`Error: ${err}`)
    }
  }

  const handleValidate = async () => {
    try {
      const result = await ValidateGeminiConfig(geminiConfig as any)
      if (result.valid) {
        setStatus('Configuration is valid')
      } else {
        setStatus(`Validation errors: ${result.errors?.map((e: any) => e.message).join(', ')}`)
      }
    } catch (err) {
      setStatus(`Error: ${err}`)
    }
  }

  const addRule = () => {
    if (newRule.trim()) {
      updateGeminiConfig({
        instructions: {
          ...geminiConfig.instructions,
          customRules: [...(geminiConfig.instructions?.customRules || []), newRule.trim()],
        },
      })
      setNewRule('')
    }
  }

  const removeRule = (index: number) => {
    const rules = [...(geminiConfig.instructions?.customRules || [])]
    rules.splice(index, 1)
    updateGeminiConfig({
      instructions: { ...geminiConfig.instructions, customRules: rules },
    })
  }

  return (
    <div className="flex flex-col h-full">
      <div className="p-4 border-b border-border wails-drag">
        <h2 className="text-xl font-semibold">Gemini CLI Configuration</h2>
        <p className="text-sm text-muted-foreground">Configure settings for Google Gemini CLI</p>
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
              value="auth"
              className="px-4 py-2 text-sm font-medium text-muted-foreground data-[state=active]:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary -mb-px"
            >
              Auth
            </Tabs.Trigger>
            <Tabs.Trigger
              value="behavior"
              className="px-4 py-2 text-sm font-medium text-muted-foreground data-[state=active]:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary -mb-px"
            >
              Behavior
            </Tabs.Trigger>
            <Tabs.Trigger
              value="instructions"
              className="px-4 py-2 text-sm font-medium text-muted-foreground data-[state=active]:text-foreground data-[state=active]:border-b-2 data-[state=active]:border-primary -mb-px"
            >
              Instructions
            </Tabs.Trigger>
          </Tabs.List>

          {/* Model Tab */}
          <Tabs.Content value="model" className="space-y-4">
            <div className="grid gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Model</label>
                <Select.Root
                  value={geminiConfig.model}
                  onValueChange={(value) => updateGeminiConfig({ model: value })}
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
                        {geminiModels.map((model) => (
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
                <label className="text-sm font-medium">Theme</label>
                <Select.Root
                  value={geminiConfig.display?.theme || 'auto'}
                  onValueChange={(value) =>
                    updateGeminiConfig({ display: { ...geminiConfig.display!, theme: value } })
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
                        {themes.map((theme) => (
                          <Select.Item
                            key={theme.value}
                            value={theme.value}
                            className="flex items-center gap-2 px-2 py-1.5 text-sm rounded cursor-pointer outline-none hover:bg-accent"
                          >
                            <Select.ItemText>{theme.label}</Select.ItemText>
                          </Select.Item>
                        ))}
                      </Select.Viewport>
                    </Select.Content>
                  </Select.Portal>
                </Select.Root>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium">Syntax Highlighting</label>
                  <p className="text-xs text-muted-foreground">Highlight code blocks</p>
                </div>
                <Switch.Root
                  checked={geminiConfig.display?.syntaxHighlight ?? true}
                  onCheckedChange={(checked) =>
                    updateGeminiConfig({
                      display: { ...geminiConfig.display!, syntaxHighlight: checked },
                    })
                  }
                  className="w-11 h-6 bg-muted rounded-full relative data-[state=checked]:bg-primary"
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform translate-x-0.5 data-[state=checked]:translate-x-5" />
                </Switch.Root>
              </div>
            </div>
          </Tabs.Content>

          {/* Auth Tab */}
          <Tabs.Content value="auth" className="space-y-4">
            <div className="grid gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Authentication Type</label>
                <Select.Root
                  value={geminiConfig.auth?.type || 'api_key'}
                  onValueChange={(value) =>
                    updateGeminiConfig({ auth: { ...geminiConfig.auth, type: value } })
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
                        {authTypes.map((auth) => (
                          <Select.Item
                            key={auth.value}
                            value={auth.value}
                            className="flex items-center gap-2 px-2 py-1.5 text-sm rounded cursor-pointer outline-none hover:bg-accent"
                          >
                            <Select.ItemText>{auth.label}</Select.ItemText>
                          </Select.Item>
                        ))}
                      </Select.Viewport>
                    </Select.Content>
                  </Select.Portal>
                </Select.Root>
              </div>

              {geminiConfig.auth?.type === 'api_key' && (
                <div className="space-y-2">
                  <label className="text-sm font-medium">API Key</label>
                  <input
                    type="password"
                    value={geminiConfig.apiKey || ''}
                    onChange={(e) => updateGeminiConfig({ apiKey: e.target.value })}
                    placeholder="AIza..."
                    className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input"
                  />
                </div>
              )}

              <div className="space-y-2">
                <label className="text-sm font-medium">Project ID (optional)</label>
                <input
                  type="text"
                  value={geminiConfig.projectId || ''}
                  onChange={(e) => updateGeminiConfig({ projectId: e.target.value })}
                  placeholder="my-gcp-project"
                  className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input"
                />
              </div>
            </div>
          </Tabs.Content>

          {/* Behavior Tab */}
          <Tabs.Content value="behavior" className="space-y-4">
            <div className="grid gap-4">
              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium">Sandbox Mode</label>
                  <p className="text-xs text-muted-foreground">Run in isolated environment</p>
                </div>
                <Switch.Root
                  checked={geminiConfig.behavior?.sandbox ?? false}
                  onCheckedChange={(checked) =>
                    updateGeminiConfig({
                      behavior: { ...geminiConfig.behavior!, sandbox: checked },
                    })
                  }
                  className="w-11 h-6 bg-muted rounded-full relative data-[state=checked]:bg-primary"
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform translate-x-0.5 data-[state=checked]:translate-x-5" />
                </Switch.Root>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="text-sm font-medium">YOLO Mode</label>
                  <p className="text-xs text-muted-foreground">Skip all confirmations (dangerous)</p>
                </div>
                <Switch.Root
                  checked={geminiConfig.behavior?.yoloMode ?? false}
                  onCheckedChange={(checked) =>
                    updateGeminiConfig({
                      behavior: { ...geminiConfig.behavior!, yoloMode: checked },
                    })
                  }
                  className="w-11 h-6 bg-muted rounded-full relative data-[state=checked]:bg-primary"
                >
                  <Switch.Thumb className="block w-5 h-5 bg-white rounded-full transition-transform translate-x-0.5 data-[state=checked]:translate-x-5" />
                </Switch.Root>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">Max File Size (MB)</label>
                <input
                  type="number"
                  value={(geminiConfig.behavior?.maxFileSize || 10485760) / 1024 / 1024}
                  onChange={(e) =>
                    updateGeminiConfig({
                      behavior: {
                        ...geminiConfig.behavior!,
                        maxFileSize: (parseInt(e.target.value) || 10) * 1024 * 1024,
                      },
                    })
                  }
                  className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input"
                />
              </div>
            </div>
          </Tabs.Content>

          {/* Instructions Tab */}
          <Tabs.Content value="instructions" className="space-y-4">
            <div className="grid gap-4">
              <div className="space-y-2">
                <label className="text-sm font-medium">Project Description</label>
                <textarea
                  value={geminiConfig.instructions?.projectDescription || ''}
                  onChange={(e) =>
                    updateGeminiConfig({
                      instructions: { ...geminiConfig.instructions, projectDescription: e.target.value },
                    })
                  }
                  placeholder="Describe your project..."
                  rows={3}
                  className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input resize-none"
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">Tech Stack</label>
                <textarea
                  value={geminiConfig.instructions?.techStack || ''}
                  onChange={(e) =>
                    updateGeminiConfig({
                      instructions: { ...geminiConfig.instructions, techStack: e.target.value },
                    })
                  }
                  placeholder="e.g., React, TypeScript, Node.js..."
                  rows={2}
                  className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input resize-none"
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">Code Style</label>
                <textarea
                  value={geminiConfig.instructions?.codeStyle || ''}
                  onChange={(e) =>
                    updateGeminiConfig({
                      instructions: { ...geminiConfig.instructions, codeStyle: e.target.value },
                    })
                  }
                  placeholder="Coding conventions and style guidelines..."
                  rows={2}
                  className="w-full px-3 py-2 text-sm border rounded-md bg-background border-input resize-none"
                />
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">Custom Rules</label>
                <div className="space-y-2">
                  {geminiConfig.instructions?.customRules?.map((rule, index) => (
                    <div key={index} className="flex items-center gap-2">
                      <span className="flex-1 px-3 py-2 text-sm bg-muted rounded-md">{rule}</span>
                      <button
                        onClick={() => removeRule(index)}
                        className="p-2 text-muted-foreground hover:text-foreground"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    </div>
                  ))}
                  <div className="flex items-center gap-2">
                    <input
                      type="text"
                      value={newRule}
                      onChange={(e) => setNewRule(e.target.value)}
                      onKeyDown={(e) => e.key === 'Enter' && addRule()}
                      placeholder="Add a new rule..."
                      className="flex-1 px-3 py-2 text-sm border rounded-md bg-background border-input"
                    />
                    <button
                      onClick={addRule}
                      className="p-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
                    >
                      <Plus className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </Tabs.Content>
        </Tabs.Root>

        <div className="mt-6">
          <ConfigPreview content={preview} language="markdown" />
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
