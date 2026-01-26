import './style.css'
import { Sidebar } from './components/Sidebar'
import { StatusBar } from './components/StatusBar'
import { ClaudePage } from './pages/ClaudePage'
import { CodexPage } from './pages/CodexPage'
import { GeminiPage } from './pages/GeminiPage'
import { useConfigStore } from './stores/configStore'

function App() {
  const { activeTool } = useConfigStore()

  const renderPage = () => {
    switch (activeTool) {
      case 'claude':
        return <ClaudePage />
      case 'codex':
        return <CodexPage />
      case 'gemini':
        return <GeminiPage />
      default:
        return <ClaudePage />
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
