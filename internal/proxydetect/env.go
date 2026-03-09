package proxydetect

import (
	"net/url"
	"os"
	"strconv"
	"strings"
)

// envVarNames lists proxy-related environment variables (uppercase + lowercase)
var envVarNames = []string{
	"HTTP_PROXY", "http_proxy",
	"HTTPS_PROXY", "https_proxy",
	"ALL_PROXY", "all_proxy",
}

// detectEnvVars checks HTTP_PROXY/HTTPS_PROXY/ALL_PROXY and their lowercase variants
func detectEnvVars() []DetectedProxy {
	seen := make(map[string]bool)
	var results []DetectedProxy

	for _, name := range envVarNames {
		val := os.Getenv(name)
		if val == "" {
			continue
		}
		p, ok := parseProxyURL(val, "env")
		if !ok || seen[p.URL] {
			continue
		}
		seen[p.URL] = true
		results = append(results, p)
	}
	return results
}

// parseProxyURL parses a proxy URL string into a DetectedProxy
func parseProxyURL(raw, source string) (DetectedProxy, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DetectedProxy{}, false
	}

	// Add scheme if missing so url.Parse works
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return DetectedProxy{}, false
	}

	host := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		return DetectedProxy{}, false
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return DetectedProxy{}, false
	}

	proxyType := "http"
	scheme := strings.ToLower(u.Scheme)
	if scheme == "socks5" || scheme == "socks5h" || scheme == "socks" {
		proxyType = "socks5"
	}

	// Reconstruct a clean URL
	cleanURL := proxyType + "://" + host + ":" + portStr
	if proxyType == "http" && (scheme == "http" || scheme == "https") {
		cleanURL = scheme + "://" + host + ":" + portStr
	}

	return DetectedProxy{
		Source: source,
		Host:   host,
		Port:   port,
		Type:   proxyType,
		URL:    cleanURL,
	}, true
}
