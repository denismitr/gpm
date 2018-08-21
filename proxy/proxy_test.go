package proxy

import (
	"strings"
	"testing"
)

func TestProxyList(t *testing.T) {
	list := NewList()
	list.Filename = "../proxy.list.example"

	list.Load()

	if list.Count() != 3 {
		t.Errorf("Invalid list length, expected 3 got %d", list.Count())
	}

	randProxy := list.Rand()
	if !strings.Contains(randProxy, "http://127.0.0.1:8") {
		t.Errorf("Expected a valid http address with some port, but got %s", randProxy)
	}
}
