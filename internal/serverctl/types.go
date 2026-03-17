package serverctl

// ServerStatus describes the current state of the embedded gateway server.
type ServerStatus struct {
	Running  bool   `json:"running"`
	Port     int    `json:"port"`
	URL      string `json:"url"`      // "http://localhost:PORT"
	Uptime   int64  `json:"uptime"`   // seconds since start
	Version  string `json:"version"`
	BinaryOK bool   `json:"binaryOk"` // binary exists and is executable
}

// ServerConfig is the persistent configuration for the embedded gateway.
type ServerConfig struct {
	Port          int    `json:"port"`           // default 19090
	SessionSecret string `json:"session_secret"` // auto-generated 32-char random string
	AdminPassword string `json:"admin_password"` // initial root password; auto-generated on first run
	AdminToken    string `json:"admin_token"`    // obtained after first successful start
	AutoStart     bool   `json:"auto_start"`
}
