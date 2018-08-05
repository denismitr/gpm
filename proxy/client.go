package proxy

import (
	"log"
	"net/http"
	"net/url"
)

func NewClient(httpProxy string, logger *log.Logger) *http.Client {
	proxyURL, err := url.Parse(httpProxy)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Printf("Proxy url for this request: %s", proxyURL)
	client := &http.Client{Transport: &http.Transport{}}
	// client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)

	return client
}
