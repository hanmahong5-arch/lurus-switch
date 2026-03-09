//go:build windows

package proxydetect

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	internetSettingsKey = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`
)

// detectSystemProxy reads the Windows registry Internet Settings for proxy config
func detectSystemProxy() []DetectedProxy {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKey, registry.QUERY_VALUE)
	if err != nil {
		return nil
	}
	defer k.Close()

	enabled, _, err := k.GetIntegerValue("ProxyEnable")
	if err != nil || enabled == 0 {
		return nil
	}

	server, _, err := k.GetStringValue("ProxyServer")
	if err != nil || server == "" {
		return nil
	}

	return parseWindowsProxyServer(server)
}

// parseWindowsProxyServer parses the Windows ProxyServer registry value.
// It can be either "host:port" or "http=host:port;https=host:port;socks=host:port"
func parseWindowsProxyServer(server string) []DetectedProxy {
	server = strings.TrimSpace(server)
	if server == "" {
		return nil
	}

	var results []DetectedProxy
	seen := make(map[string]bool)

	// Multi-protocol format: "http=1.2.3.4:80;https=1.2.3.4:443;socks=1.2.3.4:1080"
	if strings.Contains(server, "=") {
		parts := strings.Split(server, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			eqIdx := strings.Index(part, "=")
			if eqIdx < 0 {
				continue
			}
			proto := strings.ToLower(strings.TrimSpace(part[:eqIdx]))
			addr := strings.TrimSpace(part[eqIdx+1:])

			proxyType := "http"
			if proto == "socks" || proto == "socks5" {
				proxyType = "socks5"
			}

			url := fmt.Sprintf("%s://%s", proxyType, addr)
			p, ok := parseProxyURL(url, "system")
			if ok && !seen[p.URL] {
				seen[p.URL] = true
				results = append(results, p)
			}
		}
		return results
	}

	// Simple format: "host:port"
	url := "http://" + server
	p, ok := parseProxyURL(url, "system")
	if ok {
		results = append(results, p)
	}
	return results
}
