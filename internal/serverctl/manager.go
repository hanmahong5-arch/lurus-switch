package serverctl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

const (
	defaultPort      = 19090
	healthCheckPath  = "/api/status"
	startupTimeout   = 30 * time.Second
	shutdownTimeout  = 10 * time.Second
	configFileName   = "config.json"
	envFileName      = ".env"
	secretAlphabet   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	secretLength     = 32
)

// Manager controls the lifecycle of the embedded gateway server process.
type Manager struct {
	mu         sync.Mutex
	appDataDir string
	serverDir  string
	cfg        ServerConfig
	cmd        *exec.Cmd
	startTime  time.Time
	cancel     context.CancelFunc
}

// NewManager creates a Manager rooted at appDataDir.
func NewManager(appDataDir string) *Manager {
	serverDir := filepath.Join(appDataDir, serverSubDir)
	m := &Manager{
		appDataDir: appDataDir,
		serverDir:  serverDir,
	}
	m.cfg = m.loadConfig()
	return m
}

// Status returns the current server status without blocking.
func (m *Manager) Status() ServerStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	binaryPath := detectBinary(m.appDataDir)
	binaryOK := binaryPath != ""

	if m.cmd == nil || m.cmd.Process == nil {
		return ServerStatus{
			Running:  false,
			Port:     m.cfg.Port,
			URL:      "",
			Uptime:   0,
			BinaryOK: binaryOK,
		}
	}

	// Check the process is still alive (non-blocking).
	running := m.isProcessAlive()
	url := ""
	uptime := int64(0)
	if running {
		url = fmt.Sprintf("http://localhost:%d", m.cfg.Port)
		uptime = int64(time.Since(m.startTime).Seconds())
	}

	return ServerStatus{
		Running:  running,
		Port:     m.cfg.Port,
		URL:      url,
		Uptime:   uptime,
		BinaryOK: binaryOK,
	}
}

// Start launches the gateway server process.
// It writes the .env config file, executes the binary, and polls /api/status
// until the server is healthy or the timeout expires.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.isProcessAlive() {
		return nil // already running
	}

	binaryPath := detectBinary(m.appDataDir)
	if binaryPath == "" {
		return fmt.Errorf("gateway binary not found; call EnsureBinary first")
	}

	if err := m.writeEnvFile(); err != nil {
		return fmt.Errorf("write env file: %w", err)
	}

	procCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(procCtx, binaryPath)
	cmd.Dir = m.serverDir
	cmd.Env = append(os.Environ(), m.buildEnvSlice()...)

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start gateway process: %w", err)
	}

	m.cmd = cmd
	m.cancel = cancel
	m.startTime = time.Now()

	// Poll health endpoint in background; do not hold the mutex.
	m.mu.Unlock()
	err := m.waitHealthy(ctx, startupTimeout)
	m.mu.Lock()

	if err != nil {
		// Kill the process if it didn't become healthy.
		m.stopLocked()
		return fmt.Errorf("gateway did not become healthy: %w", err)
	}

	// Obtain admin token.
	if token, loginErr := m.loginAsAdmin(); loginErr == nil && token != "" {
		m.cfg.AdminToken = token
		_ = m.saveConfig(m.cfg)
	}

	return nil
}

// Stop sends SIGTERM to the server process and waits for it to exit.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopLocked()
}

// stopLocked stops the server; caller must hold m.mu.
func (m *Manager) stopLocked() error {
	if m.cmd == nil || m.cmd.Process == nil {
		return nil
	}
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	done := make(chan error, 1)
	go func() { done <- m.cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(shutdownTimeout):
		_ = m.cmd.Process.Kill()
		<-done
	}

	m.cmd = nil
	m.startTime = time.Time{}
	return nil
}

// GetConfig returns the current server configuration.
func (m *Manager) GetConfig() ServerConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg
}

// SaveConfig persists a new configuration to disk.
// If the port changed and the server is running, the change only takes effect
// after the next restart.
func (m *Manager) SaveConfig(cfg ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cfg.SessionSecret == "" {
		cfg.SessionSecret = m.cfg.SessionSecret
	}
	cfg.AdminToken = m.cfg.AdminToken
	m.cfg = cfg
	return m.saveConfig(cfg)
}

// EnsureBinary checks for the binary and downloads it if missing.
// progress(downloaded, total) is called periodically with byte counts.
func (m *Manager) EnsureBinary(ctx context.Context, progress func(downloaded, total int64)) error {
	if detectBinary(m.appDataDir) != "" {
		return nil
	}
	dest := defaultBinaryPath(m.appDataDir)
	return downloadBinary(ctx, dest, progress)
}

// GetAdminToken returns the stored admin token.
func (m *Manager) GetAdminToken() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg.AdminToken
}

// GetURL returns the base URL of the running server or "" if not running.
func (m *Manager) GetURL() string {
	s := m.Status()
	return s.URL
}

// --- internal helpers ---

func (m *Manager) isProcessAlive() bool {
	if m.cmd == nil || m.cmd.Process == nil {
		return false
	}
	// os.FindProcess always succeeds on Unix; we check via signal(0).
	err := m.cmd.Process.Signal(nil)
	return err == nil
}

func (m *Manager) waitHealthy(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://localhost:%d%s", m.cfg.Port, healthCheckPath)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timed out after %s", timeout)
}

// loginAsAdmin calls the newapi root login endpoint and returns a Bearer token.
func (m *Manager) loginAsAdmin() (string, error) {
	url := fmt.Sprintf("http://localhost:%d/api/user/login", m.cfg.Port)
	body, _ := json.Marshal(map[string]string{
		"username": "root",
		"password": "123456",
	})

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Data.Token, nil
}

// writeEnvFile writes the .env file used by the server process.
func (m *Manager) writeEnvFile() error {
	if err := os.MkdirAll(m.serverDir, 0o755); err != nil {
		return err
	}
	content := m.buildEnvContent()
	return os.WriteFile(filepath.Join(m.serverDir, envFileName), []byte(content), 0o600)
}

func (m *Manager) buildEnvContent() string {
	dbPath := filepath.Join(m.serverDir, "newapi.db")
	return fmt.Sprintf(
		"SQL_DSN=local\nSQLITE_PATH=%s\nSESSION_SECRET=%s\nPORT=%d\nGIN_MODE=release\nNODE_TYPE=master\nALLOWED_ORIGINS=wails://localhost,http://localhost:34115\n",
		dbPath, m.cfg.SessionSecret, m.cfg.Port,
	)
}

func (m *Manager) buildEnvSlice() []string {
	dbPath := filepath.Join(m.serverDir, "newapi.db")
	return []string{
		"SQL_DSN=local",
		fmt.Sprintf("SQLITE_PATH=%s", dbPath),
		fmt.Sprintf("SESSION_SECRET=%s", m.cfg.SessionSecret),
		fmt.Sprintf("PORT=%d", m.cfg.Port),
		"GIN_MODE=release",
		"NODE_TYPE=master",
		"ALLOWED_ORIGINS=wails://localhost,http://localhost:34115",
	}
}

// loadConfig reads config.json from the server directory; returns defaults if missing.
func (m *Manager) loadConfig() ServerConfig {
	configPath := filepath.Join(m.serverDir, configFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return m.defaultConfig()
	}
	var cfg ServerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return m.defaultConfig()
	}
	if cfg.Port == 0 {
		cfg.Port = defaultPort
	}
	if cfg.SessionSecret == "" {
		cfg.SessionSecret = generateSecret()
		_ = m.saveConfig(cfg)
	}
	return cfg
}

func (m *Manager) saveConfig(cfg ServerConfig) error {
	if err := os.MkdirAll(m.serverDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.serverDir, configFileName), data, 0o600)
}

func (m *Manager) defaultConfig() ServerConfig {
	return ServerConfig{
		Port:          defaultPort,
		SessionSecret: generateSecret(),
		AutoStart:     false,
	}
}

// generateSecret produces a random 32-character alphanumeric string.
func generateSecret() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, secretLength)
	for i := range b {
		b[i] = secretAlphabet[r.Intn(len(secretAlphabet))]
	}
	return string(b)
}
