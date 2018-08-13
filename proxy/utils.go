package proxy

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

func copyHeaders(dest, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dest.Add(key, value)
		}
	}
}

func getProxyStr() string {
	proxyURL := os.Getenv("GPM_PROXY_URL")
	if proxyURL == "" {
		proxyURL = "http://103.15.60.23:8080"
	}

	proxyURL = strings.Replace(proxyURL, "http://", "", 1)
	proxyURL = strings.Replace(proxyURL, "https://", "", 1)

	proxyAuth := getProxyAuth()

	if proxyAuth == "" {
		return fmt.Sprintf("http://%s", proxyURL)
	}

	return fmt.Sprintf("http://%s:@%s", proxyAuth, proxyURL)
}

func getProxyAuth() string {
	proxyAuth := os.Getenv("GPM_PROXY_AUTH")
	if proxyAuth == "" {
		proxyAuth = "" // @TODO or not TODO
	}
	return proxyAuth
}
