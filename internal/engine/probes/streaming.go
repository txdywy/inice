package probes

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/txdywy/inice/internal/model"
)

// Streaming tests Netflix, ChatGPT, YouTube, and other streaming service
// accessibility through the proxy.
func Streaming(ctx context.Context, client *http.Client) model.StreamingResult {
	result := model.StreamingResult{
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
		{"Netflix", "https://www.netflix.com/title/80018499", func(v string) { result.Netflix = v }},
		{"ChatGPT", "https://chat.openai.com/", func(v string) { result.ChatGPT = v }},
		{"YouTube", "https://www.youtube.com/", func(v string) { result.YouTube = v }},
		{"Disney", "https://www.disneyplus.com/", func(v string) { result.Disney = v }},
		{"Bilibili", "https://www.bilibili.com/", func(v string) { result.Bilibili = v }},
	}

	for _, check := range checks {
		status := checkStreaming(ctx, client, check.url)
		check.set(status)
	}

	return result
}

func checkStreaming(ctx context.Context, client *http.Client, url string) string {
	timeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeCtx, "HEAD", url, nil)
	if err != nil {
		return "ERROR"
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	// Disable redirect following to detect geo-block redirects
	clientNoRedirect := &http.Client{
		Transport: client.Transport,
		Timeout:   5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := clientNoRedirect.Do(req)
	if err != nil {
		return "ERROR"
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200, 204, 301, 302:
		// 200/204: directly accessible
		// 301/302: redirecting (some services redirect to region-specific pages)
		location := resp.Header.Get("Location")
		if location != "" {
			_ = location
			return "YES"
		}
		return "YES"
	case 403:
		return "NO"
	case 451:
		return "NO"
	default:
		return fmt.Sprintf("MAYBE(%d)", resp.StatusCode)
	}
}
