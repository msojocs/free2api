package httputil

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

type ClientConfig struct {
	ProxyURL string
	Timeout  time.Duration
}

func NewClient(cfg *ClientConfig) *http.Client {
	transport := &http.Transport{}
	if cfg != nil && cfg.ProxyURL != "" {
		proxyURL, err := url.Parse(cfg.ProxyURL)
		if err != nil {
			log.Printf("httputil: invalid proxy URL %q: %v", cfg.ProxyURL, err)
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	timeout := 30 * time.Second
	if cfg != nil && cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}
	return &http.Client{Transport: transport, Timeout: timeout}
}

func NewProxyClient(host, port, username, password, protocol string) *http.Client {
	var proxyURL string
	if username != "" && password != "" {
		proxyURL = fmt.Sprintf("%s://%s:%s@%s:%s", protocol, username, password, host, port)
	} else {
		proxyURL = fmt.Sprintf("%s://%s:%s", protocol, host, port)
	}
	return NewClient(&ClientConfig{ProxyURL: proxyURL})
}

func Get(ctx context.Context, client *http.Client, targetURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}
