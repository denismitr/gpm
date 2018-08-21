package proxy

import (
	"bufio"
	"math/rand"
	"os"
	"strings"
	"time"
)

// List - list of available proxies
type List struct {
	Filename string
	list     []string
}

// Load the contents of the proxy.list
func (l *List) Load() {
	f, err := os.Open(l.Filename)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		l.Add(scanner.Text())
	}
}

// Add proxy to the list of proxies
func (l *List) Add(proxy string) {
	if !strings.Contains(proxy, "http") {
		proxy = "http://" + proxy
	}

	l.list = append(l.list, proxy)
}

// Rand - get random proxy from list
func (l *List) Rand() string {
	s := rand.NewSource(time.Now().Unix())
	r := rand.New(s)
	return l.list[r.Intn(len(l.list))]
}

// Count the available proxies in the list
func (l *List) Count() int {
	return len(l.list)
}

// NewList make new list of proxies
func NewList() *List {
	filename := os.Getenv("GPM_PROXY_LIST")
	if filename == "" {
		filename = "proxy.list"
	}

	return &List{
		list:     make([]string, 0),
		Filename: filename,
	}
}
