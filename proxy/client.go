package proxy

import (
	"crypto/tls"
	"net/http"
	"net/url"
)

// NewClient - creates new http client
func NewClient(transport *http.Transport) *http.Client {
	return &http.Client{Transport: transport}
}

// NewTransport - creates new transport
func NewTransport() *http.Transport {
	tlsClientSkipVerify := &tls.Config{InsecureSkipVerify: true}
	return &http.Transport{TLSClientConfig: tlsClientSkipVerify}
}

// NewProxiedTransport - creates new proxied transport
func NewProxiedTransport(proxy string) (*http.Transport, error) {
	transport := NewTransport()

	proxyURL, err := url.Parse(proxy)
	if err != nil {
		return transport, err
	}

	transport.Proxy = http.ProxyURL(proxyURL)

	return transport, nil
}
