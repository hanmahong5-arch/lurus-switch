import { useTranslation } from 'react-i18next'
import { useConfigStore, type ToolsSubTab } from '../stores/configStore'
import { useDashboardStore } from '../stores/dashboardStore'
import { TabBar } from '../components/TabBar'
import { ToolConfigPage } from './ToolConfigPage'
import { MCPServerManager } from '../components/mcp/MCPServerManager'

const TOOL_TABS: { id: ToolsSubTab; label: string }[] = [
  { id: 'claude', label: 'Claude' },
  { id: 'codex', label: 'Codex' },
  { id: 'gemini', label: 'Gemini' },
  { id: 'picoclaw', label: 'PicoClaw' },
  { id: 'nullclaw', label: 'NullClaw' },
  { id: 'zeroclaw', label: 'ZeroClaw' },
  { id: 'openclaw', label: 'OpenClaw' },
  { id: 'mcp', label: 'MCP' },
]

export function NewToolsPage() {
  const { t } = useTranslation()
  const { getSubTab, setSubTab, setLastActiveTool } = useConfigStore()
  const { tools } = useDashboardStore()
  const activeTab = getSubTab('tools', 'claude') as ToolsSubTab

  // Filter: show only installed tools + mcp, or show all if none installed
  const installedTools = TOOL_TABS.filter(tab =>
    tab.id === 'mcp' || tools[tab.id]?.installed
  )
  const visibleTabs = installedTools.length > 1 ? installedTools : TOOL_TABS

  const handleTabChange = (id: string) => {
    setSubTab('tools', id)
    if (id !== 'mcp' && id !== 'snapshots') {
      setLastActiveTool(id as ToolsSubTab)
    }
  }

  return (
    <div className="h-full flex flex-col overflow-hidden">
      <TabBar
        tabs={visibleTabs.map(tab => ({ id: tab.id, label: tab.label }))}
        activeTab={activeTab}
        onTabChange={handleTabChange}
      />
      <div className="flex-1 overflow-hidden">
        {activeTab === 'mcp' ? (
          <div className="h-full overflow-y-auto p-6">
            <MCPServerManager />
          </div>
        ) : (
          <ToolConfigPage />
        )}
      </div>
    </div>
  )
}
