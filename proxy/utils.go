package proxy

import "net/http"

func copyHeaders(dest, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dest.Add(key, value)
		}
	}
}
