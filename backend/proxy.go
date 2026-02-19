package backend

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// BuildProxyTransport creates an http.Transport configured for the given proxy URL.
// Supported schemes: http://, https://, socks5://.
// Returns nil if proxyURLStr is empty (caller should use default transport).
func BuildProxyTransport(proxyURLStr string) (*http.Transport, error) {
	if proxyURLStr == "" {
		return nil, nil
	}

	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	switch proxyURL.Scheme {
	case "socks5", "socks5h":
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
		}
		return &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		}, nil
	case "http", "https":
		return &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported proxy scheme %q — use http, https, or socks5", proxyURL.Scheme)
	}
}

// BuildProxyClient creates an http.Client using the given proxy URL and timeout.
// proxyURLStr may be empty for a plain client with no proxy.
func BuildProxyClient(proxyURLStr string, timeout time.Duration) (*http.Client, error) {
	transport, err := BuildProxyTransport(proxyURLStr)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: timeout}
	if transport != nil {
		client.Transport = transport
	}
	return client, nil
}
