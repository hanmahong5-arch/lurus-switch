import './style.css'
import { Sidebar } from './components/Sidebar'
import { StatusBar } from './components/StatusBar'
import { DashboardPage } from './pages/DashboardPage'
import { ToolConfigPage } from './pages/ToolConfigPage'
import { BillingPage } from './pages/BillingPage'
import { useConfigStore } from './stores/configStore'

function App() {
  const { activeTool } = useConfigStore()

  const renderPage = () => {
    switch (activeTool) {
      case 'dashboard':
        return <DashboardPage />
      case 'claude':
      case 'codex':
      case 'gemini':
      case 'picoclaw':
        return <ToolConfigPage />
      case 'billing':
        return <BillingPage />
      default:
        return <DashboardPage />
    }
  }

  return (
    <div className="flex flex-col h-screen bg-background text-foreground">
      <div className="flex flex-1 overflow-hidden">
        <Sidebar />
        <main className="flex-1 overflow-hidden">
          {renderPage()}
        </main>
      </div>
      <StatusBar />
    </div>
  )
}

export default App
