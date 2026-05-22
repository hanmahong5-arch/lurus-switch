import { useState } from 'react'
import {
  Activity, AlertTriangle, Boxes, Cpu, Database, FileText, Inbox, Moon,
  Plus, RefreshCw, Save, Settings, Sun, Trash2, X, Zap,
} from 'lucide-react'
import {
  ActionRail, Button, Card, EmptyState, KpiCard, Modal, PageShell,
  SectionHeader, Tabs, TabsContent, TerminalChip,
} from '../components/ui'

const SPARK_24H = [12, 14, 18, 22, 19, 24, 28, 31, 33, 29, 35, 41, 38, 44, 47, 52, 58, 61, 55, 60, 63, 71, 68, 72]
const SPARK_DOWN = [88, 82, 79, 74, 71, 68, 65, 63, 61, 58, 55, 53, 51, 49, 48, 46, 44, 42, 41, 39, 38, 36, 35, 33]

export function DesignSystemPage() {
  const [theme, setTheme] = useState<'dark' | 'light'>(() =>
    document.documentElement.classList.contains('dark') ? 'dark' : 'light',
  )
  const [tab, setTab] = useState('overview')
  const [pillTab, setPillTab] = useState('analyze')
  const [modalOpen, setModalOpen] = useState(false)

  const cycleTheme = () => {
    const next = theme === 'dark' ? 'light' : 'dark'
    document.documentElement.classList.toggle('dark', next === 'dark')
    setTheme(next)
  }

  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="sticky top-0 z-10 border-b border-rule-strong bg-background/95 backdrop-blur">
        <div className="flex items-center justify-between px-6 py-3">
          <div className="flex items-center gap-3">
            <span className="font-mono text-[10px] uppercase tracking-[0.18em] text-muted-foreground">
              [ SWITCH · DESIGN SYSTEM v2 ]
            </span>
            <TerminalChip tone="ok">primitives loaded · 10</TerminalChip>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" icon={theme === 'dark' ? <Sun className="h-3.5 w-3.5" /> : <Moon className="h-3.5 w-3.5" />} onClick={cycleTheme}>
              {theme}
            </Button>
            <Button variant="ghost" size="sm" onClick={() => { window.location.hash = '' }}>exit</Button>
          </div>
        </div>
      </div>

      <PageShell>
        {/* KpiCard */}
        <section className="space-y-3">
          <SectionHeader title="kpi card" action={<TerminalChip tone="info">4 variants</TerminalChip>} />
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            <KpiCard label="audience" value="12,847" delta={{ value: 18.3, label: '7d' }} sparkline={SPARK_24H} icon={Activity} />
            <KpiCard label="latency" value="184ms" delta={{ value: -4.2, label: 'p95' }} sparkline={SPARK_DOWN} icon={Zap} />
            <KpiCard label="errors" value="0.02%" delta={{ value: -0.8 }} icon={AlertTriangle} accent />
            <KpiCard label="agents" value={23} icon={Cpu} />
          </div>
        </section>

        {/* Button */}
        <section className="space-y-3">
          <SectionHeader title="button" />
          <Card variant="default" className="p-4 space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <Button variant="primary">primary</Button>
              <Button variant="secondary">secondary</Button>
              <Button variant="ghost">ghost</Button>
              <Button variant="danger">danger</Button>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Button size="sm">sm</Button>
              <Button size="md">md</Button>
              <Button size="lg">lg</Button>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Button icon={<Plus className="h-3.5 w-3.5" />}>icon-left</Button>
              <Button variant="secondary" icon={<RefreshCw className="h-3.5 w-3.5" />} iconPos="right">icon-right</Button>
              <Button loading>loading</Button>
              <Button disabled>disabled</Button>
            </div>
          </Card>
        </section>

        {/* Card */}
        <section className="space-y-3">
          <SectionHeader title="card" />
          <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
            <Card variant="default" className="p-3">
              <div className="text-xs font-medium mb-1">default</div>
              <p className="text-[11px] text-muted-foreground">bg-card/40 + border</p>
            </Card>
            <Card variant="elevated" className="p-3">
              <div className="text-xs font-medium mb-1">elevated</div>
              <p className="text-[11px] text-muted-foreground">card-elevated + shadow</p>
            </Card>
            <Card variant="recessed" className="p-3">
              <div className="text-xs font-medium mb-1">recessed</div>
              <p className="text-[11px] text-muted-foreground">card-recessed inset</p>
            </Card>
            <Card variant="elevated" glow className="p-3">
              <div className="text-xs font-medium mb-1 text-primary">glow / selected</div>
              <p className="text-[11px] text-muted-foreground">ring + shadow-glow-orange</p>
            </Card>
          </div>
        </section>

        {/* SectionHeader */}
        <section className="space-y-3">
          <SectionHeader title="section header" />
          <Card variant="default" className="p-4 space-y-3">
            <SectionHeader title="live usage" />
            <SectionHeader
              title="audit events"
              action={<Button variant="ghost" size="sm" icon={<RefreshCw className="h-3 w-3" />}>refresh</Button>}
            />
            <SectionHeader title="legacy plain" mono={false} />
          </Card>
        </section>

        {/* TerminalChip */}
        <section className="space-y-3">
          <SectionHeader title="terminal chip" />
          <Card variant="default" className="p-4 flex flex-wrap gap-3">
            <TerminalChip>idle · 0 events</TerminalChip>
            <TerminalChip tone="ok">healthy · 12 agents</TerminalChip>
            <TerminalChip tone="warn" icon="▾">latency p95 · 480ms</TerminalChip>
            <TerminalChip tone="err" icon="▪">api · 503</TerminalChip>
            <TerminalChip tone="info" icon="·">branch · main</TerminalChip>
          </Card>
        </section>

        {/* Tabs */}
        <section className="space-y-3">
          <SectionHeader title="tabs" />
          <Card variant="default" className="p-4 space-y-4">
            <Tabs
              tabs={[
                { value: 'overview',  label: 'Overview',  icon: Boxes },
                { value: 'audience',  label: 'Audience',  icon: Activity },
                { value: 'research',  label: 'Research',  icon: FileText },
                { value: 'data',      label: 'Data',      icon: Database, disabled: true },
              ]}
              value={tab}
              onValueChange={setTab}
            >
              <TabsContent value="overview" className="mt-3 text-xs text-muted-foreground">overview body</TabsContent>
              <TabsContent value="audience" className="mt-3 text-xs text-muted-foreground">audience body</TabsContent>
              <TabsContent value="research" className="mt-3 text-xs text-muted-foreground">research body</TabsContent>
            </Tabs>
            <Tabs
              variant="pill"
              tabs={[
                { value: 'analyze',  label: 'Analyze' },
                { value: 'compose',  label: 'Compose' },
                { value: 'release',  label: 'Release' },
              ]}
              value={pillTab}
              onValueChange={setPillTab}
            >
              <TabsContent value={pillTab} className="mt-3 text-xs text-muted-foreground">{pillTab} body</TabsContent>
            </Tabs>
          </Card>
        </section>

        {/* EmptyState */}
        <section className="space-y-3">
          <SectionHeader title="empty state" />
          <Card variant="default">
            <EmptyState
              icon={Inbox}
              title="No agents yet"
              hint="Start by selecting one of the templates below or import from a deeplink."
              action={<Button icon={<Plus className="h-3.5 w-3.5" />}>Create Agent</Button>}
            />
          </Card>
        </section>

        {/* Modal */}
        <section className="space-y-3">
          <SectionHeader title="modal" />
          <Card variant="default" className="p-4">
            <Button onClick={() => setModalOpen(true)} icon={<Settings className="h-3.5 w-3.5" />}>open modal</Button>
            <Modal
              open={modalOpen}
              onClose={() => setModalOpen(false)}
              title="[ MODAL DEMO ]"
              icon={Settings}
              size="md"
              footer={
                <>
                  <Button variant="ghost" size="sm" onClick={() => setModalOpen(false)}>cancel</Button>
                  <Button size="sm" icon={<Save className="h-3.5 w-3.5" />} onClick={() => setModalOpen(false)}>save</Button>
                </>
              }
            >
              <p className="text-xs text-muted-foreground leading-relaxed">
                Radix Dialog + framer-motion enter / exit. Header / body / footer slots.
              </p>
              <TerminalChip tone="ok" className="mt-3">backdrop-blur · animated</TerminalChip>
            </Modal>
          </Card>
        </section>

        {/* ActionRail */}
        <section className="space-y-3">
          <SectionHeader title="action rail" />
          <Card variant="default" className="p-4 relative">
            <p className="text-xs text-muted-foreground mb-4">Sticky bottom rail — left slot for secondary, right slot for primary.</p>
            <ActionRail
              sticky={false}
              left={
                <>
                  <Button variant="ghost" size="sm" icon={<X className="h-3 w-3" />}>cancel</Button>
                  <Button variant="ghost" size="sm" icon={<Trash2 className="h-3 w-3" />}>discard</Button>
                </>
              }
              right={
                <>
                  <Button variant="secondary" size="sm">run diagnostics</Button>
                  <Button size="sm" icon={<Save className="h-3.5 w-3.5" />}>save & activate</Button>
                </>
              }
            />
          </Card>
        </section>
      </PageShell>
    </div>
  )
}
