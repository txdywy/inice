package probes

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/txdywy/inice/internal/model"
)

// Streaming tests Netflix, ChatGPT, YouTube, and other streaming service
// accessibility through the proxy.
func Streaming(ctx context.Context, client *http.Client) model.StreamingResult {
	result := model.StreamingResult{
		Google:   "ERROR",
		GitHub:   "ERROR",
		Netflix:  "ERROR",
		ChatGPT:  "ERROR",
		YouTube:  "ERROR",
		Disney:   "ERROR",
		Bilibili: "ERROR",
	}

	checks := []struct {
		name string
		url  string
		set  func(string)
	}{
		{"Google", "https://www.google.com/generate_204", func(v string) { result.Google = v }},
		{"GitHub", "https://github.com/", func(v string) { result.GitHub = v }},
		{"Netflix", "https://www.netflix.com/title/80018499", func(v string) { result.Netflix = v }},
		{"ChatGPT", "https://chat.openai.com/", func(v string) { result.ChatGPT = v }},
		{"YouTube", "https://www.youtube.com/", func(v string) { result.YouTube = v }},
		{"Disney", "https://www.disneyplus.com/", func(v string) { result.Disney = v }},
		{"Bilibili", "https://www.bilibili.com/", func(v string) { result.Bilibili = v }},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, check := range checks {
		wg.Add(1)
		go func(c struct {
			name string
			url  string
			set  func(string)
		}) {
			defer wg.Done()
			status := checkStreaming(ctx, client, c.url)
			mu.Lock()
			c.set(status)
			mu.Unlock()
		}(check)
	}

	wg.Wait()

	return result
}

func checkStreaming(ctx context.Context, client *http.Client, url string) string {
	timeout := client.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	timeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(timeCtx, "HEAD", url, nil)
	if err != nil {
		return "ERROR"
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	// Disable redirect following to detect geo-block redirects
	clientNoRedirect := &http.Client{
		Transport: client.Transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	start := time.Now()
	resp, err := clientNoRedirect.Do(req)
	if err != nil {
		return "ERROR"
	}
	defer resp.Body.Close()

	latencyMs := int(time.Since(start).Milliseconds())

	switch resp.StatusCode {
	case 200, 204, 301, 302, 307, 308, 405:
		// 200/204: directly accessible
		// 301/302/307/308: redirecting (some services redirect to region-specific pages)
		// 405: Method Not Allowed (some servers reject HEAD but are otherwise reachable)
		location := resp.Header.Get("Location")
		if location != "" {
			_ = location
			return fmt.Sprintf("%dms", latencyMs)
		}
		return fmt.Sprintf("%dms", latencyMs)
	case 403:
		return "NO"
	case 451:
		return "NO"
	default:
		return fmt.Sprintf("MAYBE(%d)", resp.StatusCode)
	}
}
