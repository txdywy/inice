package probes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/txdywy/inice/internal/model"
)

const (
	cloudflareTrace = "https://www.cloudflare.com/cdn-cgi/trace"
	ipSBGeoIP       = "https://api.ip.sb/geoip"
)

type ipSBResponse struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	City    string `json:"city"`
	ISP     string `json:"isp"`
	ASN     int    `json:"asn"`
}

// ExitIP detects the proxy's exit IP through Cloudflare trace and ip.sb.
func ExitIP(ctx context.Context, client *http.Client) model.IPInfo {
	info := model.IPInfo{}

	// Cloudflare trace (primary)
	cfInfo := fetchCloudflareTrace(ctx, client)
	if cfInfo.IP != "" {
		info = cfInfo
		info.Source = "cloudflare"
	}

	// ip.sb (supplementary, for geo/ISP)
	timeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeCtx, "GET", ipSBGeoIP, nil)
	if err != nil {
		return info
	}

	resp, err := client.Do(req)
	if err != nil {
		return info
	}
	defer resp.Body.Close()

	var geo ipSBResponse
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return info
	}

	// Merge: Cloudflare IP is authoritative, ip.sb provides richer geo
	if info.IP == "" {
		info.IP = geo.IP
	}
	info.Country = geo.Country
	info.City = geo.City
	info.ISP = geo.ISP
	info.ASN = fmt.Sprintf("AS%d", geo.ASN)
	info.Source = "cloudflare+ip.sb"

	return info
}

func fetchCloudflareTrace(ctx context.Context, client *http.Client) model.IPInfo {
	timeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(timeCtx, "GET", cloudflareTrace, nil)
	if err != nil {
		return model.IPInfo{}
	}

	resp, err := client.Do(req)
	if err != nil {
		return model.IPInfo{}
	}
	defer resp.Body.Close()

	// Parse plaintext trace output:
	// fl=xxx
	// ip=1.2.3.4
	// colo=LAX
	// loc=US
	var info model.IPInfo

	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	for _, line := range strings.Split(string(buf[:n]), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "ip="):
			info.IP = strings.TrimPrefix(line, "ip=")
		case strings.HasPrefix(line, "colo="):
			info.Colo = strings.TrimPrefix(line, "colo=")
		case strings.HasPrefix(line, "loc="):
			info.Country = strings.TrimPrefix(line, "loc=")
		}
	}

	return info
}
