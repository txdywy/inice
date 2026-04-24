package probes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

)

const ipAPIURL = "http://ip-api.com/json/%s"

type ipAPIResponse struct {
	Status    string  `json:"status"`
	Country   string  `json:"country"`
	City      string  `json:"city"`
	ISP       string  `json:"isp"`
	ASN       string  `json:"as"`
	Org       string  `json:"org"`
	Hosting   bool    `json:"hosting"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
}

// Purity checks if the exit IP is a datacenter/hosting IP.
func Purity(ctx context.Context, client *http.Client, ip string) (bool, error) {
	if ip == "" {
		return false, fmt.Errorf("no IP to check")
	}

	timeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	url := fmt.Sprintf(ipAPIURL, ip)
	req, err := http.NewRequestWithContext(timeCtx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("decode ip-api response: %w", err)
	}

	if result.Status != "success" {
		return false, fmt.Errorf("ip-api failed: %s", result.Status)
	}

	return result.Hosting, nil
}

// hostingKeywords are common indicators of datacenter/hosting IPs.
var hostingKeywords = []string{
	"DIGITALOCEAN", "VULTR", "HETZNER", "LINODE", "AWS", "AMAZON",
	"GOOGLE", "GOOGLE CLOUD", "MICROSOFT", "AZURE", "ORACLE", "ALIBABA",
	"TENCENT", "HUAWEI", "OVH", "ONLINE", "CHOOPA", "LEASEWEB",
	"DATACAMP", "M247", "IT7", "BUYVM", "RACKNICE", "DMIT",
	"BANDWAGON", "HOSTHATCH", "NETCUP", "CLOUDFLARE", "FASTLY",
	"AKAMAI", "DATACENTER", "HOSTING", "SERVER", "CLOUD",
}

// classifyISP checks ISP/org name against known hosting providers.
func classifyISP(isp string) bool {
	for _, kw := range hostingKeywords {
		if len(isp) >= len(kw) {
			for i := 0; i <= len(isp)-len(kw); i++ {
				if equalFold(isp[i:i+len(kw)], kw) {
					return true
				}
			}
		}
	}
	return false
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'a' && ca <= 'z' {
			ca -= 32
		}
		if cb >= 'a' && cb <= 'z' {
			cb -= 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
