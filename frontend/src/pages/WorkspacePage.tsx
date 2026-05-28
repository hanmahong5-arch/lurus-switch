import { useTranslation } from 'react-i18next'
import { BookOpen, BookMarked, Plug, FileText, Activity } from 'lucide-react'
import { useConfigStore, type WorkspaceSubTab } from '../stores/configStore'
import { TabBar } from '../components/TabBar'
import { PromptLibraryPage } from './PromptLibraryPage'
import { DocumentPage } from './DocumentPage'
import { ProcessPage } from './ProcessPage'
import { RulesMarketPage } from './RulesMarketPage'
import { McpMarketPage } from './McpMarketPage'

export function WorkspacePage() {
  const { t } = useTranslation()
  const { getSubTab, setSubTab } = useConfigStore()
  const activeTab = getSubTab('workspace', 'prompts') as WorkspaceSubTab

  const tabs = [
    { id: 'prompts', label: t('nav.prompts'), icon: BookOpen },
    { id: 'rules', label: t('nav.rules', 'Rules Market'), icon: BookMarked },
    { id: 'mcp-market', label: t('nav.mcpMarket', 'MCP Market'), icon: Plug },
    { id: 'context', label: t('nav.documents'), icon: FileText },
    { id: 'process', label: t('home.processTab'), icon: Activity },
  ]

  return (
    <div className="h-full flex flex-col overflow-hidden">
      <TabBar
        tabs={tabs}
        activeTab={activeTab}
        onTabChange={(id) => setSubTab('workspace', id)}
      />
      <div className="flex-1 overflow-hidden">
        {activeTab === 'prompts' && <PromptLibraryPage />}
        {activeTab === 'rules' && <RulesMarketPage />}
        {activeTab === 'mcp-market' && <McpMarketPage />}
        {activeTab === 'context' && <DocumentPage />}
        {activeTab === 'process' && (
          <div className="h-full flex flex-col overflow-hidden">
            <div className="flex-1 overflow-y-auto">
              <ProcessPage />
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
