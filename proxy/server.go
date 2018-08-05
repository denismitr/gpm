package proxy

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Server struct {
	Logger *log.Logger
}

func (s *Server) handleFailure(w http.ResponseWriter, url string, err error) {
	s.Logger.Printf("\nRequest to %s failed.", url)
	s.Logger.Println(err)

	w.WriteHeader(http.StatusBadGateway)
	w.Write([]byte("\r\nBad gateway"))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Logger.Printf("Headers: %v\n", r.Header)
	s.Logger.Printf("Host: %v\n", r.Host)
	s.Logger.Printf("Scheme: %v\n", r.URL.Scheme)
	s.Logger.Printf("Protocol: %v\n", r.Proto)
	s.Logger.Printf("Method: %v\n", r.Method)
	s.Logger.Printf("Remote ADDR: %v\n", r.RemoteAddr)
	s.Logger.Printf("Referer: %v\n", r.Referer())
	s.Logger.Printf("Request Cookies: %v\n", r.Cookies())
	s.Logger.Printf("URL: %v\n", r.URL.String())
	s.Logger.Printf("Request URI: %v\n", r.URL.RequestURI())
	s.Logger.Printf("Query: %v\n", r.URL.Query())
	s.Logger.Printf("Path: %v\n", r.URL.Path)
	s.Logger.Printf("Body: %v\n", r.Body)

	client := NewClient("https://proxy.crawlera.com:8010", s.Logger)
	url := r.URL.String()

	response, err := client.Get(url)
	if err != nil {
		s.handleFailure(w, url, err)
		return
	}

	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			s.handleFailure(w, url, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	} else {
		s.handleFailure(w, url, fmt.Errorf("Error status %d", response.StatusCode))
	}
}

func NewServer(logger *log.Logger) *Server {
	server := Server{
		Logger: logger,
	}

	return &server
}
