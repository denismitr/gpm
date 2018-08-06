package proxy

import (
	"log"
	"net/http"
	"net/url"
	"time"
)

func NewClient(httpProxy string, timeout time.Duration, logger *log.Logger) *http.Client {
	proxyURL, err := url.Parse(httpProxy)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Printf("Proxy url for this request: %s", proxyURL)
	client := &http.Client{Transport: &http.Transport{}, Timeout: timeout}
	// client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)

	return client
}
