package main

import (
	"fmt"
	"sync"

	"lurus-switch/internal/agent"
	"lurus-switch/internal/analytics"
	"lurus-switch/internal/appreg"
	"lurus-switch/internal/audit"
	"lurus-switch/internal/auth"
	"lurus-switch/internal/billing"
	"lurus-switch/internal/budget"
	"lurus-switch/internal/config"
	"lurus-switch/internal/db"
	"lurus-switch/internal/docmgr"
	"lurus-switch/internal/envmgr"
	"lurus-switch/internal/gateway"
	"lurus-switch/internal/installer"
	"lurus-switch/internal/mcp"
	"lurus-switch/internal/metering"
	"lurus-switch/internal/modelcatalog"
	"lurus-switch/internal/orgsync"
	"lurus-switch/internal/process"
	"lurus-switch/internal/promoter"
	"lurus-switch/internal/promptlib"
	"lurus-switch/internal/provider"
	"lurus-switch/internal/proxy"
	"lurus-switch/internal/redemption"
	"lurus-switch/internal/relay"
	"lurus-switch/internal/serverctl"
	"lurus-switch/internal/snapshot"
	"lurus-switch/internal/updater"
	"lurus-switch/internal/validator"
)

// services holds all service dependencies for the application.
// It is embedded in App so that existing field accesses (a.store, a.instMgr, etc.)
// continue to work without modification.
// Constructable independently via newServices() for isolated testing.
type services struct {
	store     *config.Store
	validator *validator.Validator
	instMgr   *installer.Manager
	proxyMgr  *proxy.ProxyManager

	// OIDC authentication session (Zitadel PKCE).
	authSession *auth.Session

	selfUpdater *updater.SelfUpdater
	npmChecker  *updater.NpmChecker

	billingMu     sync.Mutex
	billingClient *billing.Client

	processMon  *process.Monitor
	snapshotStr *snapshot.Store
	promptStr   *promptlib.Store
	mcpStr      *mcp.Store
	docMgr      *docmgr.Manager
	envMgr      *envmgr.Manager
	tracker     *analytics.Tracker

	serverMgr   *serverctl.Manager
	relayStore  *relay.Store
	promoterSvc *promoter.Service
	catalogMgr  *modelcatalog.Manager

	// User-defined provider endpoints (Wave2 EXT-1). Persisted to
	// custom-providers.json; nil-safe via the bindings.
	customProviderStore *provider.CustomStore

	// Local API gateway (replaces serverctl for new architecture).
	appRegistry *appreg.Registry
	meterStore  *metering.Store
	gatewaySrv  *gateway.Server
	budgetGuard *budget.Guard // active spend wall, wired into gateway

	// Agent fleet management (v3 龙虾管理员).
	database       *db.DB
	agentStore     *agent.Store
	agentConfigMgr *agent.ConfigManager
	agentInstMgr   *agent.InstanceManager

	// EndUser activation lifecycle (S-Xc.4 / S-Xc.5). redemptionStore is
	// the persistent activation file; redeemer issues the redemption HTTP
	// call; heartbeat is started lazily on activation and at startup.
	redemptionStore *redemption.Store
	redeemer        *redemption.Redeemer
	heartbeat       *redemption.Heartbeat

	// Append-only journal of state-mutating bindings. The capability
	// package gates writes; the audit package records them with enough
	// payload for the Undo UI to revert.
	auditJournal *audit.Journal

	// Enterprise-mode org chart store. Lazily created on first access
	// so Personal/Reseller installs don't pay the file IO cost.
	orgsyncMu sync.Mutex
	orgsync   *orgsync.Store
}

// newServices constructs all service dependencies. Initialization failures for
// optional services are collected as warnings rather than causing a fatal error.
func newServices(appDataDir, version string) (*services, []string) {
	var warnings []string

	store, err := config.NewStore()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("config store: %v", err))
	}

	proxyMgr, err := proxy.NewProxyManager()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("proxy manager: %v", err))
	}

	snapStr, err := snapshot.NewStore()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("snapshot store: %v", err))
	}

	promptStr, err := promptlib.NewStore()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("prompt store: %v", err))
	}

	mcpStr, err := mcp.NewStore()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("mcp store: %v", err))
	}

	authSess, err := auth.NewSession()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("auth session: %v", err))
	}

	tracker, err := analytics.NewTracker()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("analytics tracker: %v", err))
	}

	relayStr, err := relay.NewStore(appDataDir)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("relay store: %v", err))
	}

	appReg, err := appreg.NewRegistry(appDataDir)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("app registry: %v", err))
	}

	meterStr, err := metering.NewStore(appDataDir)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("metering store: %v", err))
	}

	// Open SQLite database for agent fleet management.
	database, err := db.Open(appDataDir)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("database: %v", err))
	}

	var agentStr *agent.Store
	if database != nil {
		agentStr = agent.NewStore(database)
	}

	agentCfgMgr, err := agent.NewConfigManager(appDataDir)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("agent config manager: %v", err))
	}

	redemptionStr, err := redemption.NewStore()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("redemption store: %v", err))
	}

	auditJ, aerr := audit.NewJournal(appDataDir)
	if aerr != nil {
		warnings = append(warnings, fmt.Sprintf("audit journal: %v", aerr))
	}

	customProvStr, cpErr := provider.NewCustomStore(appDataDir)
	if cpErr != nil {
		warnings = append(warnings, fmt.Sprintf("custom provider store: %v", cpErr))
	}

	svc := &services{
		store:          store,
		validator:      validator.NewValidator(),
		instMgr:        installer.NewManager(),
		proxyMgr:       proxyMgr,
		authSession:    authSess,
		selfUpdater:    updater.NewSelfUpdater(version),
		npmChecker:     updater.NewNpmChecker(),
		processMon:     process.NewMonitor(),
		snapshotStr:    snapStr,
		promptStr:      promptStr,
		mcpStr:         mcpStr,
		docMgr:         docmgr.NewManager(),
		envMgr:         envmgr.NewManager(),
		tracker:        tracker,
		serverMgr:      serverctl.NewManager(appDataDir),
		relayStore:     relayStr,
		catalogMgr:     modelcatalog.NewManager(appDataDir),
		appRegistry:    appReg,
		meterStore:     meterStr,
		database:       database,
		agentStore:     agentStr,
		agentConfigMgr: agentCfgMgr,
		agentInstMgr: func() *agent.InstanceManager {
			if agentStr != nil && agentCfgMgr != nil {
				return agent.NewInstanceManager(agentStr, agentCfgMgr, process.NewMonitor())
			}
			return nil
		}(),
		redemptionStore:     redemptionStr,
		redeemer:            redemption.NewRedeemer(version),
		auditJournal:        auditJ,
		customProviderStore: customProvStr,
	}
	// Gateway depends on appRegistry and meterStore, so create after the struct.
	if appReg != nil && meterStr != nil {
		svc.gatewaySrv = gateway.NewServer(appDataDir, appReg, meterStr)
		// Active Budget Wall — persisted config alongside other gateway
		// state. The guard delegates "today's tokens" to meterStore so
		// daily limits track all traffic, not just this process's session.
		guard, gErr := budget.New(
			fmt.Sprintf("%s/budget.json", appDataDir),
			func() metering.DailySummary { return meterStr.TodaySummary() },
		)
		if gErr == nil {
			svc.budgetGuard = guard
			svc.gatewaySrv.SetBudgetGuard(guard)
		} else {
			warnings = append(warnings, fmt.Sprintf("budget guard init failed: %v", gErr))
		}
	}
	svc.promoterSvc = promoter.NewService(svc.ensureBillingClient)
	return svc, warnings
}

// ensureBillingClient lazily initializes the billing client.
// Priority: OIDC session gateway token > proxy settings UserToken.
func (s *services) ensureBillingClient() (*billing.Client, error) {
	s.billingMu.Lock()
	defer s.billingMu.Unlock()

	if s.billingClient != nil {
		return s.billingClient, nil
	}

	// Prefer the gateway token from the OIDC session when available.
	if s.authSession != nil && s.authSession.HasGatewayToken() {
		endpoint := ""
		if s.proxyMgr != nil {
			endpoint = s.proxyMgr.GetSettings().APIEndpoint
		}
		if endpoint == "" {
			endpoint = "https://api.lurus.cn"
		}
		token := s.authSession.GetGatewayToken()
		s.billingClient = billing.NewClient(endpoint, "", token)
		return s.billingClient, nil
	}

	// Fall back to manual proxy UserToken.
	if s.proxyMgr == nil {
		return nil, fmt.Errorf("proxy manager not initialized")
	}
	settings := s.proxyMgr.GetSettings()
	if settings.UserToken == "" {
		return nil, fmt.Errorf("user token not configured: login with your Lurus account or paste a token in Proxy Settings")
	}
	if settings.APIEndpoint == "" {
		return nil, fmt.Errorf("API endpoint not configured")
	}
	s.billingClient = billing.NewClient(settings.APIEndpoint, settings.TenantSlug, settings.UserToken)
	return s.billingClient, nil
}

// resetBillingClient clears the cached billing client, forcing re-creation on next use.
func (s *services) resetBillingClient() {
	s.billingMu.Lock()
	defer s.billingMu.Unlock()
	s.billingClient = nil
}
