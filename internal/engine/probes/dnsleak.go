package probes

import (
	"context"
	"net"
	"time"

	"github.com/txdywy/inice/internal/model"
)

const dnsLeakTestDomain = "dnsleaktest.com"

// DNSLeak checks if DNS queries leak outside the proxy by comparing
// resolution through the proxy vs. direct resolution.
func DNSLeak(ctx context.Context, proxyDialer proxyDialerFunc) model.DNSLeakResult {
	result := model.DNSLeakResult{}

	// Resolve through proxy by dialing a TCP connection to a DNS server
	timeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	proxyIPs := resolveViaProxy(timeCtx, proxyDialer, dnsLeakTestDomain)
	result.ProxyResolvedIPs = proxyIPs

	// Resolve directly
	directIPs := resolveDirect(dnsLeakTestDomain)
	result.DirectResolvedIPs = directIPs

	// Check for leaks: if proxy resolves the same IPs as direct DNS
	leakCount := 0
	for _, pip := range proxyIPs {
		for _, dip := range directIPs {
			if pip == dip {
				leakCount++
			}
		}
	}

	result.LeakCount = leakCount
	result.LeakDetected = leakCount > 0 && len(proxyIPs) > 0 && len(directIPs) > 0

	return result
}

type proxyDialerFunc func(network, addr string) (net.Conn, error)

func resolveViaProxy(ctx context.Context, dialer proxyDialerFunc, domain string) []string {
	// Use the proxy to connect to a DNS server and send a query
	// For simplicity, resolve by connecting to the domain through the proxy
	// and checking what IP the proxy resolved it to
	var ips []string

	// Try connecting to port 80 of the domain through proxy
	conn, err := dialer("tcp", net.JoinHostPort(domain, "80"))
	if err != nil {
		return ips
	}
	remoteAddr := conn.RemoteAddr()
	conn.Close()

	if tcpAddr, ok := remoteAddr.(*net.TCPAddr); ok {
		ips = append(ips, tcpAddr.IP.String())
	}

	return ips
}

func resolveDirect(domain string) []string {
	var ips []string

	// Try direct resolution
	addrs, err := net.LookupHost(domain)
	if err != nil {
		return ips
	}

	for _, addr := range addrs {
		ips = append(ips, addr)
	}

	// Rate limit: wait a bit between resolutions
	time.Sleep(100 * time.Millisecond)

	return ips
}
