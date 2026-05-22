import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Zap, RotateCcw } from 'lucide-react'
import { useRelayStore } from '../../stores/relayStore'
import { ResetRelayCircuit } from '../../../wailsjs/go/main/App'

// CircuitStateChip displays the breaker state for a single endpoint and
// exposes a one-click "Reset Circuit" affordance when it's open.
// Polls every 5s while mounted so the RelayPage column stays live.
export function CircuitStateChip({ endpointID }: { endpointID: string }) {
  const { t } = useTranslation()
  const circuitState = useRelayStore((s) => s.circuitState[endpointID])
  const pollCircuitState = useRelayStore((s) => s.pollCircuitState)

  useEffect(() => {
    void pollCircuitState()
    const h = setInterval(() => { void pollCircuitState() }, 5000)
    return () => clearInterval(h)
  }, [pollCircuitState])

  if (!circuitState || circuitState.status === 'closed') {
    return (
      <span className="inline-flex items-center gap-1 text-[11px] text-emerald-500">
        <Zap className="h-3 w-3" />
        {t('relay.circuit.closed', 'Healthy')}
      </span>
    )
  }
  const isOpen = circuitState.status === 'open'
  const isHalf = circuitState.status === 'half_open'
  const reset = async () => {
    try { await ResetRelayCircuit(endpointID) } catch { /* swallow; poll will surface */ }
    void pollCircuitState()
  }
  return (
    <span className="inline-flex items-center gap-1">
      <span className={
        'inline-flex items-center gap-1 text-[11px] ' +
        (isOpen ? 'text-red-500' : 'text-amber-500')
      }>
        <Zap className="h-3 w-3" />
        {isOpen
          ? t('relay.circuit.open', 'Circuit open')
          : isHalf ? t('relay.circuit.halfOpen', 'Probing')
          : circuitState.status}
        {circuitState.consecutiveFailures > 0 && (
          <span className="text-muted-foreground">× {circuitState.consecutiveFailures}</span>
        )}
      </span>
      {isOpen && (
        <button
          onClick={reset}
          title={t('relay.circuit.reset', 'Reset')}
          className="p-0.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground"
        >
          <RotateCcw className="h-3 w-3" />
        </button>
      )}
    </span>
  )
}
