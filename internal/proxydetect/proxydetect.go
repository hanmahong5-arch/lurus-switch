package proxydetect

// DetectedProxy represents a proxy found on the system
type DetectedProxy struct {
	Source string `json:"source"` // "env", "clash", "v2ray", "system", "socks"
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Type   string `json:"type"` // "http" | "socks5"
	URL    string `json:"url"`  // e.g. "http://127.0.0.1:7890"
}

// DetectAll runs all detection methods and returns discovered proxies (deduplicated)
func DetectAll() []DetectedProxy {
	var results []DetectedProxy
	seen := make(map[string]bool)

	add := func(proxies []DetectedProxy) {
		for _, p := range proxies {
			if !seen[p.URL] {
				seen[p.URL] = true
				results = append(results, p)
			}
		}
	}

	add(detectEnvVars())
	add(detectSystemProxy())
	add(detectCommonPorts())

	return results
}
