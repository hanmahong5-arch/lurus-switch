import { Server } from 'lucide-react'
import { useConfigStore } from '../stores/configStore'
import { useGatewayStore } from '../stores/gatewayStore'

interface Props {
  children: React.ReactNode
}

export function GatewayRequiredGuard({ children }: Props) {
  const running = useGatewayStore((s) => s.status?.running ?? false)
  const { setActiveTool } = useConfigStore()

  if (!running) {
    return (
      <div className="h-full flex flex-col items-center justify-center gap-4 text-center p-8">
        <div className="p-3 rounded-full bg-muted">
          <Server className="h-8 w-8 text-muted-foreground" />
        </div>
        <div>
          <h3 className="text-sm font-semibold">网关服务未运行</h3>
          <p className="text-xs text-muted-foreground mt-1">访问此功能需要先启动网关服务</p>
        </div>
        <button
          onClick={() => setActiveTool('gateway')}
          className="flex items-center gap-1.5 px-4 py-2 rounded-md text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
        >
          <Server className="h-4 w-4" />
          前往启动
        </button>
      </div>
    )
  }

  return <>{children}</>
}
