// Package topology composes a single architecture-snapshot view of all the
// runtime entities Switch coordinates — CLI tools, the local gateway, the
// optional BYO upstream proxy, the Lurus Hub, the upstream model providers,
// the OIDC session, and (in EndUser mode) the activation token.
//
// The Compose function is pure: callers gather inputs via existing probes
// (gateway.Status, installer.DetectAll, connectivity.Report, auth.AuthState,
// redemption.Status, etc.) and pass them in. Compose decides node ordering,
// status classification, and the mode-specific edge graph. Frontend renders
// the result as a clickable diagram; every red/yellow node carries a
// NavPage/SubTab pointing at the page that can fix it, and (when applicable)
// a FixAction naming the Wails binding to call inline.
//
// Design note: every branch state a Chinese-network user hits — DNS poison,
// TLS RST, proxy down, npm registry blocked, OIDC token expired, activation
// revoked — must end up as a labelled node or edge here so the home view
// can answer "what's broken and what do I click" without a toast.
package topology

import (
	"sort"
	"time"
)

// NodeStatus classifies an entity's health.
type NodeStatus string

const (
	StatusOK            NodeStatus = "ok"
	StatusDegraded      NodeStatus = "degraded"
	StatusDown          NodeStatus = "down"
	StatusUnknown       NodeStatus = "unknown"
	StatusNotConfigured NodeStatus = "notconfigured"
)

// NodeKind identifies the role of a node so the frontend can pick an icon
// and apply the right click-handler.
type NodeKind string

const (
	KindTool     NodeKind = "tool"
	KindGateway  NodeKind = "gateway"
	KindProxy    NodeKind = "proxy"
	KindHub      NodeKind = "hub"
	KindProvider NodeKind = "provider"
	KindAuth     NodeKind = "auth"
	KindMCP      NodeKind = "mcp"
)

// CredType labels what kind of credential rides on an edge. Frontend
// translates the value to a small chip.
type CredType string

const (
	CredNone       CredType = ""
	CredEnv        CredType = "env"        // CLI reads ANTHROPIC_API_KEY / OPENAI_API_KEY env
	CredConfig     CredType = "config"     // CLI reads ~/.claude/settings.json etc.
	CredOIDC       CredType = "oidc"       // Zitadel ID token
	CredAPIKey     CredType = "apikey"     // Bearer token to upstream
	CredActivation CredType = "activation" // EndUser activation code → token
	CredStdio      CredType = "stdio"      // MCP subprocess pipe
)

// Node is one entity on the topology canvas.
type Node struct {
	ID        string     `json:"id"`
	Kind      NodeKind   `json:"kind"`
	Label     string     `json:"label"`
	Status    NodeStatus `json:"status"`
	Detail    string     `json:"detail,omitempty"`
	Hint      string     `json:"hint,omitempty"`
	NavPage   string     `json:"navPage,omitempty"`
	NavSubTab string     `json:"navSubTab,omitempty"`
	FixAction string     `json:"fixAction,omitempty"`
	LatencyMs int64      `json:"latencyMs,omitempty"`
	Badge     string     `json:"badge,omitempty"`
	// Highlight marks the node as currently in-use (e.g. the active model)
	// so the UI can render an emphasised border without flipping its status.
	Highlight bool `json:"highlight,omitempty"`
}

// Edge connects two nodes and carries credential + status metadata.
type Edge struct {
	From       string     `json:"from"`
	To         string     `json:"to"`
	Credential CredType   `json:"credential,omitempty"`
	Status     NodeStatus `json:"status,omitempty"`
	Label      string     `json:"label,omitempty"`
}

// Summary is the one-glance roll-up surfaced above the topology.
type Summary struct {
	OK            int    `json:"ok"`
	Degraded      int    `json:"degraded"`
	Down          int    `json:"down"`
	NotConfigured int    `json:"notconfigured"`
	Unknown       int    `json:"unknown"`
	Headline      string `json:"headline"`
}

// Snapshot is the entire scene the frontend renders.
type Snapshot struct {
	Mode        string    `json:"mode"`
	GeneratedAt time.Time `json:"generatedAt"`
	Nodes       []Node    `json:"nodes"`
	Edges       []Edge    `json:"edges"`
	Summary     Summary   `json:"summary"`
}

// ToolInput describes one detected CLI tool.
type ToolInput struct {
	Name       string
	Installed  bool
	Version    string
	Path       string
	Health     string // "green" | "yellow" | "red" — from toolhealth.CheckAll
	Update     bool   // updateAvailable
	ComingSoon bool   // manifest marks status="coming-soon" — install path not yet hosted
}

// GatewayInput captures local gateway status.
type GatewayInput struct {
	Running       bool
	Port          int
	URL           string
	Uptime        int64
	TotalRequests int64
	UpstreamURL   string // configured upstream API endpoint
}

// ProxyInput describes the BYO upstream proxy.
type ProxyInput struct {
	Configured bool
	Enabled    bool
	URL        string  // user-facing proxy URL, e.g. socks5://127.0.0.1:1080
	Reachable  *bool   // tri-state: nil = untested, otherwise true/false
	LatencyMs  int64
	Error      string
}

// HubInput is the Lurus Hub reachability slot.
type HubInput struct {
	URL       string
	Reachable bool
	LatencyMs int64
	Error     string
}

// ProviderInput is one row of the connectivity doctor matrix.
type ProviderInput struct {
	ID            string
	Label         string
	DNSOK         bool
	DirectOK      bool
	DirectMs      int64
	UpstreamOK    bool
	UpstreamMs    int64
	UpstreamTried bool
	Error         string
}

// AuthInput captures OIDC session state.
type AuthInput struct {
	LoggedIn        bool
	HasGatewayToken bool
	UserEmail       string
	ExpiresAt       string
}

// ActivationInput captures EndUser activation lifecycle.
type ActivationInput struct {
	State       string // "unactivated" | "active" | "stale" | "revoked" | "device_mismatch"
	HubURL      string
	TenantSlug  string
	ExpiresAt   time.Time
	LastBeat    time.Time
}

// ComposeInput bundles everything Compose needs.
type ComposeInput struct {
	Mode         string
	Tools        []ToolInput
	Gateway      GatewayInput
	Proxy        ProxyInput
	Hub          HubInput
	Providers    []ProviderInput
	Auth         AuthInput
	Activation   ActivationInput // only meaningful in EndUser mode
	CurrentModel string          // for highlighting the active provider
}

// Compose turns a ComposeInput into a renderable Snapshot. Pure function —
// no I/O, no goroutines. The probe orchestration runs in the binding layer.
func Compose(in ComposeInput, now time.Time) Snapshot {
	snap := Snapshot{
		Mode:        in.Mode,
		GeneratedAt: now,
		Nodes:       []Node{},
		Edges:       []Edge{},
	}

	addToolNodes(&snap, in.Tools)
	addGatewayNode(&snap, in.Gateway)
	addProxyNode(&snap, in.Proxy)
	addAuthNode(&snap, in.Mode, in.Auth, in.Activation)
	addHubNode(&snap, in.Mode, in.Hub)
	addProviderNodes(&snap, in.Providers, in.CurrentModel, in.Proxy.Enabled)

	addToolToGatewayEdges(&snap, in.Tools, in.Gateway.Running)
	addGatewayToAuthEdge(&snap, in.Mode, in.Auth, in.Activation)
	addGatewayToProxyEdge(&snap, in.Proxy, in.Gateway.Running)
	addProxyOrGatewayToHubEdge(&snap, in.Proxy, in.Hub, in.Gateway.Running)
	addHubToProviderEdges(&snap, in.Providers, in.Hub, in.Proxy.Enabled)

	snap.Summary = rollUp(snap.Nodes)
	return snap
}

func addToolNodes(snap *Snapshot, tools []ToolInput) {
	for _, t := range tools {
		n := Node{
			ID:        "tool:" + t.Name,
			Kind:      KindTool,
			Label:     toolDisplay(t.Name),
			NavPage:   "tools",
			NavSubTab: t.Name,
		}
		switch {
		case !t.Installed && t.ComingSoon:
			// "敬请期待" — manifest knows about this tool but no installable
			// artifact is hosted yet. Deliberately omit FixAction so the
			// action bar doesn't render a 404-bound install button.
			n.Status = StatusNotConfigured
			n.Detail = "敬请期待"
			n.Hint = "Lurus 上架后将自动出现安装按钮"
			n.Badge = "敬请期待"
		case !t.Installed:
			n.Status = StatusNotConfigured
			n.Detail = "未安装"
			n.Hint = "点击安装"
			n.FixAction = "install-tool:" + t.Name
		case t.Health == "red":
			n.Status = StatusDown
			n.Detail = "配置异常"
			n.Hint = "查看修复建议"
			n.FixAction = "fix-tool:" + t.Name
			n.Badge = t.Version
		case t.Health == "yellow" || t.Update:
			n.Status = StatusDegraded
			if t.Update {
				n.Detail = "有新版本"
				n.Hint = "点击更新"
				n.FixAction = "update-tool:" + t.Name
			} else {
				n.Detail = "配置可优化"
				n.Hint = "查看建议"
			}
			n.Badge = t.Version
		default:
			n.Status = StatusOK
			n.Detail = "已安装并配置"
			n.Badge = t.Version
		}
		snap.Nodes = append(snap.Nodes, n)
	}
}

func addGatewayNode(snap *Snapshot, g GatewayInput) {
	n := Node{
		ID:      "gateway",
		Kind:    KindGateway,
		Label:   "本地网关 / Local Gateway",
		NavPage: "gateway",
	}
	switch {
	case g.Running:
		n.Status = StatusOK
		n.Detail = g.URL
		n.Badge = formatPort(g.Port)
	case g.Port == 0:
		n.Status = StatusNotConfigured
		n.Detail = "未配置端口"
		n.Hint = "前往网关设置"
	default:
		n.Status = StatusDown
		n.Detail = "已停止"
		n.Hint = "点击启动网关"
		n.FixAction = "start-gateway"
		n.Badge = formatPort(g.Port)
	}
	snap.Nodes = append(snap.Nodes, n)
}

func addProxyNode(snap *Snapshot, p ProxyInput) {
	n := Node{
		ID:      "proxy",
		Kind:    KindProxy,
		Label:   "上游代理 / Upstream Proxy",
		NavPage: "settings",
		NavSubTab: "proxy",
	}
	switch {
	case !p.Configured:
		n.Status = StatusNotConfigured
		n.Detail = "未配置（可选）"
		n.Hint = "中国大陆网络可在此自带代理"
	case !p.Enabled:
		n.Status = StatusNotConfigured
		n.Detail = "已配置但未启用"
		n.Hint = "前往设置启用"
	case p.Reachable != nil && !*p.Reachable:
		n.Status = StatusDown
		n.Detail = "代理不可达"
		if p.Error != "" {
			n.Detail = trimErr(p.Error)
		}
		n.Hint = "检查本地代理是否在运行"
		n.Badge = p.URL
	case p.Reachable != nil && *p.Reachable:
		n.Status = StatusOK
		n.Detail = p.URL
		n.LatencyMs = p.LatencyMs
	default:
		n.Status = StatusUnknown
		n.Detail = p.URL
	}
	snap.Nodes = append(snap.Nodes, n)
}

func addAuthNode(snap *Snapshot, mode string, auth AuthInput, act ActivationInput) {
	if mode == "enduser" {
		n := Node{
			ID:        "auth",
			Kind:      KindAuth,
			Label:     "激活令牌 / Activation",
			NavPage:   "settings",
			NavSubTab: "activation",
		}
		switch act.State {
		case "active":
			n.Status = StatusOK
			n.Detail = "已激活"
			n.Badge = act.TenantSlug
		case "stale":
			n.Status = StatusDegraded
			n.Detail = "心跳暂时失败，仍可使用"
			n.Hint = "稍后重试"
		case "revoked":
			n.Status = StatusDown
			n.Detail = "激活已被吊销"
			n.Hint = "联系客服或重新激活"
			n.FixAction = "reactivate"
		case "device_mismatch":
			n.Status = StatusDown
			n.Detail = "设备指纹不匹配"
			n.Hint = "请联系发行方解绑设备"
		default:
			n.Status = StatusNotConfigured
			n.Detail = "尚未激活"
			n.Hint = "输入激活码"
			n.FixAction = "reactivate"
		}
		snap.Nodes = append(snap.Nodes, n)
		return
	}
	// Personal / Reseller / Enterprise: OIDC session
	n := Node{
		ID:        "auth",
		Kind:      KindAuth,
		Label:     "登录会话 / OIDC",
		NavPage:   "account",
	}
	switch {
	case !auth.LoggedIn:
		n.Status = StatusNotConfigured
		n.Detail = "未登录"
		n.Hint = "点击登录"
		n.FixAction = "login"
	case !auth.HasGatewayToken:
		n.Status = StatusDegraded
		n.Detail = "已登录，网关令牌未签发"
		n.Hint = "重新登录以重签"
		n.FixAction = "login"
		n.Badge = auth.UserEmail
	default:
		n.Status = StatusOK
		n.Detail = auth.UserEmail
		n.Badge = "OIDC"
	}
	snap.Nodes = append(snap.Nodes, n)
}

func addHubNode(snap *Snapshot, mode string, h HubInput) {
	label := "Lurus Hub"
	if mode == "reseller" {
		label = "经销商 Hub / Reseller Hub"
	} else if mode == "enduser" {
		label = "白标 Hub / White-label Hub"
	}
	n := Node{
		ID:      "hub",
		Kind:    KindHub,
		Label:   label,
		NavPage: "settings",
		NavSubTab: "hub",
	}
	switch {
	case h.URL == "":
		n.Status = StatusNotConfigured
		n.Detail = "未配置 Hub URL"
		n.Hint = "前往设置填写"
	case h.Reachable:
		n.Status = StatusOK
		n.Detail = h.URL
		n.LatencyMs = h.LatencyMs
	default:
		n.Status = StatusDown
		n.Detail = h.URL
		if h.Error != "" {
			n.Hint = trimErr(h.Error)
		} else {
			n.Hint = "Hub 不可达 — 检查网络或上游代理"
		}
	}
	snap.Nodes = append(snap.Nodes, n)
}

// providerOrder fixes the canvas order so the home page doesn't jitter
// between snapshot polls.
var providerOrder = []string{"anthropic", "openai", "gemini", "deepseek", "github", "npm"}

func addProviderNodes(snap *Snapshot, providers []ProviderInput, currentModel string, proxyEnabled bool) {
	byID := make(map[string]ProviderInput, len(providers))
	for _, p := range providers {
		byID[p.ID] = p
	}
	for _, id := range providerOrder {
		p, ok := byID[id]
		if !ok {
			continue
		}
		n := Node{
			ID:    "provider:" + p.ID,
			Kind:  KindProvider,
			Label: p.Label,
			NavPage: "settings",
			NavSubTab: "models",
		}
		direct := p.DirectOK
		via := p.UpstreamOK && p.UpstreamTried
		switch {
		case direct:
			n.Status = StatusOK
			n.Detail = "可直连"
			n.LatencyMs = p.DirectMs
		case via:
			n.Status = StatusDegraded
			n.Detail = "需经上游代理"
			n.LatencyMs = p.UpstreamMs
			if !proxyEnabled {
				n.Hint = "启用上游代理以走此路径"
			}
		case !p.DNSOK:
			n.Status = StatusDown
			n.Detail = "DNS 解析失败"
			n.Hint = "切换 DNS 或启用上游代理"
		default:
			n.Status = StatusDown
			n.Detail = "不可达"
			n.Hint = "启用上游代理或改走 Hub 中转"
		}
		// Highlight the in-use provider derived from the current model.
		if currentModel != "" && providerForModel(currentModel) == p.ID {
			n.Highlight = true
		}
		snap.Nodes = append(snap.Nodes, n)
	}
}

// addToolToGatewayEdges draws one edge per installed tool to the gateway,
// labelling the credential the CLI uses to find the gateway.
func addToolToGatewayEdges(snap *Snapshot, tools []ToolInput, gatewayRunning bool) {
	for _, t := range tools {
		if !t.Installed {
			continue
		}
		e := Edge{
			From:       "tool:" + t.Name,
			To:         "gateway",
			Credential: CredConfig,
			Label:      configFileFor(t.Name),
		}
		if gatewayRunning {
			e.Status = StatusOK
		} else {
			e.Status = StatusDown
		}
		snap.Edges = append(snap.Edges, e)
	}
}

func addGatewayToAuthEdge(snap *Snapshot, mode string, auth AuthInput, act ActivationInput) {
	e := Edge{From: "gateway", To: "auth"}
	if mode == "enduser" {
		e.Credential = CredActivation
		if act.State == "active" || act.State == "stale" {
			e.Status = StatusOK
		} else {
			e.Status = StatusDown
		}
	} else {
		e.Credential = CredOIDC
		if auth.LoggedIn && auth.HasGatewayToken {
			e.Status = StatusOK
		} else if auth.LoggedIn {
			e.Status = StatusDegraded
		} else {
			e.Status = StatusDown
		}
	}
	snap.Edges = append(snap.Edges, e)
}

func addGatewayToProxyEdge(snap *Snapshot, p ProxyInput, gatewayRunning bool) {
	if !p.Configured {
		return
	}
	e := Edge{
		From:       "gateway",
		To:         "proxy",
		Credential: CredNone,
		Label:      "可选 / optional",
	}
	switch {
	case !p.Enabled:
		e.Status = StatusNotConfigured
	case p.Reachable != nil && !*p.Reachable:
		e.Status = StatusDown
	case gatewayRunning && p.Reachable != nil && *p.Reachable:
		e.Status = StatusOK
	default:
		e.Status = StatusUnknown
	}
	snap.Edges = append(snap.Edges, e)
}

func addProxyOrGatewayToHubEdge(snap *Snapshot, p ProxyInput, h HubInput, gatewayRunning bool) {
	from := "gateway"
	label := "HTTPS 直连"
	if p.Configured && p.Enabled {
		from = "proxy"
		label = "经上游代理"
	}
	e := Edge{
		From:       from,
		To:         "hub",
		Credential: CredAPIKey,
		Label:      label,
	}
	switch {
	case !gatewayRunning:
		e.Status = StatusDown
	case h.Reachable:
		e.Status = StatusOK
	default:
		e.Status = StatusDown
	}
	snap.Edges = append(snap.Edges, e)
}

func addHubToProviderEdges(snap *Snapshot, providers []ProviderInput, h HubInput, proxyEnabled bool) {
	for _, id := range providerOrder {
		var p ProviderInput
		found := false
		for _, pp := range providers {
			if pp.ID == id {
				p = pp
				found = true
				break
			}
		}
		if !found {
			continue
		}
		e := Edge{
			From:       "hub",
			To:         "provider:" + p.ID,
			Credential: CredAPIKey,
		}
		switch {
		case p.DirectOK || (p.UpstreamTried && p.UpstreamOK):
			e.Status = StatusOK
		case !h.Reachable:
			e.Status = StatusDown
		default:
			e.Status = StatusDown
		}
		snap.Edges = append(snap.Edges, e)
	}
	_ = proxyEnabled
}

func rollUp(nodes []Node) Summary {
	s := Summary{}
	for _, n := range nodes {
		switch n.Status {
		case StatusOK:
			s.OK++
		case StatusDegraded:
			s.Degraded++
		case StatusDown:
			s.Down++
		case StatusNotConfigured:
			s.NotConfigured++
		default:
			s.Unknown++
		}
	}
	s.Headline = headline(s, nodes)
	return s
}

// headline picks the single most-useful sentence to put above the canvas.
// Priority: anything down > anything degraded > tools missing > all-ok.
func headline(s Summary, nodes []Node) string {
	// Find first down node for a concrete pointer.
	var firstDown *Node
	for i := range nodes {
		if nodes[i].Status == StatusDown {
			firstDown = &nodes[i]
			break
		}
	}
	if firstDown != nil {
		return firstDown.Label + " 不可用：" + firstDown.Hint
	}
	if s.Degraded > 0 {
		return "部分组件降级运行 — 点击黄色节点查看建议"
	}
	if s.NotConfigured > 0 {
		return "首次使用：完成未配置项让 AI CLI 跑通"
	}
	return "所有组件正常 — AI CLI 可直接使用"
}

// providerForModel maps a model ID to its upstream provider for highlighting.
// Conservative — unknown IDs return "" (no highlight) rather than guessing.
func providerForModel(model string) string {
	switch {
	case startsWith(model, "claude"):
		return "anthropic"
	case startsWith(model, "gpt"), startsWith(model, "o1"), startsWith(model, "o3"):
		return "openai"
	case startsWith(model, "gemini"):
		return "gemini"
	case startsWith(model, "deepseek"):
		return "deepseek"
	}
	return ""
}

func startsWith(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}

func toolDisplay(name string) string {
	switch name {
	case "claude":
		return "Claude Code"
	case "codex":
		return "Codex CLI"
	case "gemini":
		return "Gemini CLI"
	case "picoclaw":
		return "PicoClaw"
	case "nullclaw":
		return "NullClaw"
	case "zeroclaw":
		return "ZeroClaw"
	case "openclaw":
		return "OpenClaw"
	}
	return name
}

func configFileFor(name string) string {
	switch name {
	case "claude":
		return "~/.claude/settings.json"
	case "codex":
		return "~/.codex/config.toml"
	case "gemini":
		return "~/.gemini/settings.json"
	case "picoclaw":
		return "~/.picoclaw/config.json"
	case "nullclaw":
		return "~/.nullclaw/config.json"
	case "zeroclaw":
		return "~/.zeroclaw/config.json"
	case "openclaw":
		return "~/.openclaw/config.json"
	}
	return ""
}

func formatPort(p int) string {
	if p <= 0 {
		return ""
	}
	return ":" + itoa(p)
}

func trimErr(s string) string {
	const max = 80
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// SortNodesForRender stabilises node order across snapshots so SVG positions
// don't jitter between polls. Tools first (by Name), then gateway/proxy/auth,
// then hub, then providers.
func SortNodesForRender(nodes []Node) {
	rank := map[NodeKind]int{
		KindTool:     1,
		KindGateway:  2,
		KindProxy:    3,
		KindAuth:     4,
		KindHub:      5,
		KindMCP:      6,
		KindProvider: 7,
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		ri, rj := rank[nodes[i].Kind], rank[nodes[j].Kind]
		if ri != rj {
			return ri < rj
		}
		return nodes[i].ID < nodes[j].ID
	})
}
