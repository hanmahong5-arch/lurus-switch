package main

import (
	"fmt"
	"sync"

	"lurus-switch/internal/analytics"
	"lurus-switch/internal/billing"
	"lurus-switch/internal/config"
	"lurus-switch/internal/docmgr"
	"lurus-switch/internal/envmgr"
	"lurus-switch/internal/installer"
	"lurus-switch/internal/mcp"
	"lurus-switch/internal/process"
	"lurus-switch/internal/promoter"
	"lurus-switch/internal/promptlib"
	"lurus-switch/internal/proxy"
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

	tracker, err := analytics.NewTracker()
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("analytics tracker: %v", err))
	}

	relayStr, err := relay.NewStore(appDataDir)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("relay store: %v", err))
	}

	svc := &services{
		store:       store,
		validator:   validator.NewValidator(),
		instMgr:     installer.NewManager(),
		proxyMgr:    proxyMgr,
		selfUpdater: updater.NewSelfUpdater(version),
		npmChecker:  updater.NewNpmChecker(),
		processMon:  process.NewMonitor(),
		snapshotStr: snapStr,
		promptStr:   promptStr,
		mcpStr:      mcpStr,
		docMgr:      docmgr.NewManager(),
		envMgr:      envmgr.NewManager(),
		tracker:     tracker,
		serverMgr:   serverctl.NewManager(appDataDir),
		relayStore:  relayStr,
	}
	svc.promoterSvc = promoter.NewService(svc.ensureBillingClient)
	return svc, warnings
}

// ensureBillingClient lazily initializes the billing client from proxy settings.
func (s *services) ensureBillingClient() (*billing.Client, error) {
	s.billingMu.Lock()
	defer s.billingMu.Unlock()

	if s.billingClient != nil {
		return s.billingClient, nil
	}
	if s.proxyMgr == nil {
		return nil, fmt.Errorf("proxy manager not initialized")
	}
	settings := s.proxyMgr.GetSettings()
	if settings.UserToken == "" {
		return nil, fmt.Errorf("user token not configured: paste your token in Proxy Settings")
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
