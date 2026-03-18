package gy

import (
	"context"
	"net/http"
	"sync"
	"time"
)

const statusTimeout = 5 * time.Second

// BuiltinProducts returns the three built-in GY suite product definitions.
func BuiltinProducts() []GYProduct {
	return []GYProduct{
		{
			ID:          "lucrum",
			Name:        "Lucrum AI 交易助手",
			Description: "AI 量化交易分析平台",
			Kind:        KindWeb,
			LaunchURL:   "https://gushen.lurus.cn",
		},
		{
			ID:          "creator",
			Name:        "Lurus Creator",
			Description: "AI 内容创作工厂（桌面应用）",
			Kind:        KindDesktop,
			DownloadURL: "https://github.com/lurus-dev/lurus-creator/releases/latest",
		},
		{
			ID:          "memorus",
			Name:        "Lurus Memorus",
			Description: "AI 记忆引擎（后台服务）",
			Kind:        KindService,
			ServiceURL:  "https://memorus.lurus.cn",
		},
	}
}

// CheckStatus concurrently probes each product and returns status results.
func CheckStatus(ctx context.Context, products []GYProduct) []GYStatus {
	results := make([]GYStatus, len(products))
	var wg sync.WaitGroup

	for i, p := range products {
		wg.Add(1)
		go func(idx int, product GYProduct) {
			defer wg.Done()
			st := GYStatus{ProductID: product.ID}

			switch product.Kind {
			case KindWeb:
				st.LatencyMs, st.Available = pingURL(ctx, product.LaunchURL)
			case KindService:
				st.LatencyMs, st.Available = pingURL(ctx, product.ServiceURL)
			case KindDesktop:
				path, err := FindCreatorExe()
				if err == nil && path != "" {
					st.Available = true
					st.Version = "installed"
				}
			}
			results[idx] = st
		}(i, p)
	}

	wg.Wait()
	return results
}

func pingURL(ctx context.Context, url string) (int64, bool) {
	if url == "" {
		return -1, false
	}
	client := &http.Client{
		Timeout: statusTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return -1, false
	}
	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return -1, false
	}
	resp.Body.Close()
	return latency, resp.StatusCode < 500
}
