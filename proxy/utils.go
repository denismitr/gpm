package proxy

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func copyHeaders(dest, src http.Header) {
	src.Del("Content-Length")

	for key, values := range src {
		for _, value := range values {
			dest.Add(key, value)
		}
	}
}

func getProxyStr() string {
	proxyURL := os.Getenv("GPM_PROXY_URL")
	if proxyURL == "" {
		return ""
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

func getConcurrentTries() int {
	concurrentTries, err := strconv.Atoi(os.Getenv("GPM_CONCURRENT_TRIES"))
	if err != nil {
		concurrentTries = 3
	}

	return concurrentTries
}

// GetMaxTimeout - get maximum timeout from env
func GetMaxTimeout() int {
	maxTimeout, err := strconv.Atoi(os.Getenv("GPM_MAX_TIMEOUT"))
	if err != nil {
		maxTimeout = 10 // seconds
	}

	return maxTimeout
}

// ParseURLParam retrieves and parse URL param from request query
func ParseURLParam(r *http.Request) (string, error) {
	u, err := ExtractQueryParam(r, "url")
	if err != nil {
		return "", err
	}

	if u == "" {
		return "", errors.New("[url] query param is empty")
	}

	decoded, err := base64.StdEncoding.DecodeString(u)
	if err == nil {
		u = string(decoded)
	}

	regx := regexp.MustCompile(`^(?:http(s)?:\/\/)?[\w.-]+(?:\.[\w\.-]+)+[\w\-\._~:/?#[\]@!\$&'\(\)\*\+,;=.]+$`)

	if !regx.MatchString(u) {
		return "", errors.New("passed url value does not match a valid url pattern")
	}

	return u, nil
}

// ExtractQueryParam - extract query param from request query
func ExtractQueryParam(r *http.Request, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("empty query param given")
	}

	keys, ok := r.URL.Query()[key]
	if !ok || len(keys) < 1 {
		return "", fmt.Errorf("no [%s] query param found", key)
	}

	return keys[0], nil
}

// PrintMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
