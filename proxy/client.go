package proxy

import (
	"net/http"
)

// NewClient - creates new client with proxy transport
func NewClient(transport *http.Transport) *http.Client {
	client := &http.Client{Transport: transport}

	return client
}
