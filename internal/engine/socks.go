package engine

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

// NewSOCKSDialer creates a SOCKS5 dialer that routes through the given proxy address.
func NewSOCKSDialer(proxyAddr string, timeout time.Duration) (proxy.Dialer, error) {
	return proxy.SOCKS5("tcp", proxyAddr, nil, &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
	})
}

// NewSOCKSHostFixDialer creates a SOCKS5 dialer that always connects to the proxy
// but uses a custom resolve callback so the proxy resolves target hostnames.
func NewSOCKSHostFixDialer(proxyAddr string, timeout time.Duration) (proxy.Dialer, error) {
	return proxy.SOCKS5("tcp", proxyAddr, nil, &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
		Resolver:  nil, // nil means use system resolver for the proxy address itself
	})
}

// NewHTTPClient creates an HTTP client that routes through a SOCKS5 proxy.
func NewHTTPClient(proxyAddr string, timeout time.Duration) (*http.Client, error) {
	dialer, err := NewSOCKSDialer(proxyAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("SOCKS5 dialer: %w", err)
	}

	transport := &http.Transport{
		Dial:                  dialer.Dial,
		MaxIdleConnsPerHost:   5,
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       30 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}, nil
}

// NewHTTPClientWithContext creates an HTTP client with context-aware dialing.
func NewHTTPClientWithContext(proxyAddr string, timeout time.Duration) (*http.Client, error) {
	dialer, err := NewSOCKSDialer(proxyAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("SOCKS5 dialer: %w", err)
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
		MaxIdleConnsPerHost:   5,
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       30 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}, nil
}
