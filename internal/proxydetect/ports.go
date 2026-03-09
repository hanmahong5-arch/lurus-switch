package proxydetect

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// probeTimeout is the TCP dial timeout per port
const probeTimeout = 1 * time.Second

// commonPort describes a well-known proxy port to probe
type commonPort struct {
	port   int
	source string // human-readable source name
	typ    string // "http" | "socks5"
}

var commonPorts = []commonPort{
	{7890, "clash", "http"},
	{7891, "clash", "socks5"},
	{10808, "v2ray", "http"},
	{10809, "v2ray", "socks5"},
	{1080, "socks", "socks5"},
}

// detectCommonPorts probes localhost for well-known proxy ports concurrently
func detectCommonPorts() []DetectedProxy {
	type indexedResult struct {
		idx   int
		proxy *DetectedProxy
	}

	var wg sync.WaitGroup
	ch := make(chan indexedResult, len(commonPorts))

	for i, cp := range commonPorts {
		wg.Add(1)
		go func(idx int, cp commonPort) {
			defer wg.Done()
			addr := fmt.Sprintf("127.0.0.1:%d", cp.port)
			conn, err := net.DialTimeout("tcp", addr, probeTimeout)
			if err != nil {
				ch <- indexedResult{idx: idx}
				return
			}
			_ = conn.Close()

			scheme := "http"
			if cp.typ == "socks5" {
				scheme = "socks5"
			}
			ch <- indexedResult{idx: idx, proxy: &DetectedProxy{
				Source: cp.source,
				Host:   "127.0.0.1",
				Port:   cp.port,
				Type:   cp.typ,
				URL:    fmt.Sprintf("%s://127.0.0.1:%d", scheme, cp.port),
			}}
		}(i, cp)
	}

	// Close channel after all goroutines complete
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Collect results preserving original order
	ordered := make([]*DetectedProxy, len(commonPorts))
	for r := range ch {
		ordered[r.idx] = r.proxy
	}

	var results []DetectedProxy
	for _, p := range ordered {
		if p != nil {
			results = append(results, *p)
		}
	}
	return results
}
