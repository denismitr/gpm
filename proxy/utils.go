package proxy

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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
		proxyURL = "http://103.15.60.23:8080" // default proxy
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

func ParseURLArgument(r *http.Request) (string, error) {
	query := r.URL.Query()

	list, ok := query["url"]
	if !ok {
		return "", errors.New("No [url] query param found")
	}

	if len(list) < 1 || list[0] == "" {
		return "", errors.New("[url] query param is empty")
	}

	u := list[0]

	regx := regexp.MustCompile(`^(?:http(s)?:\/\/)?[\w.-]+(?:\.[\w\.-]+)+[\w\-\._~:/?#[\]@!\$&'\(\)\*\+,;=.]+$`)

	if !regx.MatchString(u) {
		return "", errors.New("Passed url value does not match a valid url pattern")
	}

	return u, nil
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
