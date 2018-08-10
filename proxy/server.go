package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var concurrentTries int
var waitGatewayResponseFor int

func init() {
	var err error

	concurrentTries, err = strconv.Atoi(os.Getenv("GPM_CONCURRENT_TRIES"))
	if err != nil {
		concurrentTries = 3
	}

	waitGatewayResponseFor, err = strconv.Atoi(os.Getenv("GPM_WAIT_GATEWAY_RESPONSE_FOR"))
	if err != nil {
		waitGatewayResponseFor = 6 // seconds
	}
}

type Server struct {
	Logger *log.Logger
}

type FirstValidResponse struct {
	Err      error
	Response *http.Response
	TimedOut bool
}

func (s *Server) handleFailure(w http.ResponseWriter, url string, err error) {
	s.Logger.Printf("\nRequest to %s failed.", url)
	s.Logger.Println(err)

	w.WriteHeader(http.StatusBadGateway)
	w.Write([]byte("\r\nBad gateway"))
}

func (s *Server) handleGatewayFailure(url string, err error) {
	s.Logger.Printf("\nRequest to %s failed.", url)
	s.Logger.Println(err)
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

	firstValidResponse := s.processRequest(r)

	if firstValidResponse.Err != nil {
		s.Logger.Println(firstValidResponse.Err)
		return
	}

	if firstValidResponse.Response.StatusCode >= 200 && firstValidResponse.Response.StatusCode < 300 {
		s.Logger.Println("Sending success response")

		copyHeaders(w.Header(), firstValidResponse.Response.Header)
		w.WriteHeader(firstValidResponse.Response.StatusCode)

		bytesCopied, _ := io.Copy(w, firstValidResponse.Response.Body)
		if err := firstValidResponse.Response.Body.Close(); err != nil {
			s.Logger.Printf("Can't close response body %v", err)
		}

		s.Logger.Printf("Copied %v bytes to the client", bytesCopied)
	} else {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("\r\nBad gateway"))
	}

	s.Logger.Println("Done. Response delivered...")
}

func (s *Server) processRequest(r *http.Request) *FirstValidResponse {
	defer func() {
		s.Logger.Println("\nProcess request method exiting...")
	}()

	timeout := time.Duration(waitGatewayResponseFor) * time.Second
	client := NewClient("https://proxy.crawlera.com:8010", timeout, s.Logger)
	url := r.URL.String()
	responseCh := make(chan *http.Response, concurrentTries)
	signal := time.Tick(timeout + time.Second)

	for i := 0; i < concurrentTries; i++ {
		go func(id int) {
			defer s.Logger.Printf("\nClient request [%d] exiting...", id)
			response, err := client.Get(url)
			if err != nil {
				s.handleGatewayFailure(url, err)
				return
			}

			if response.StatusCode == http.StatusOK {
				responseCh <- response
			} else {
				s.handleGatewayFailure(url, fmt.Errorf("Error status %d", response.StatusCode))
			}
		}(i)
	}

	for {
		select {
		case firstResponse := <-responseCh:
			s.Logger.Printf("\nReceived success response from %s", url)
			return &FirstValidResponse{
				Response: firstResponse,
				Err:      nil,
				TimedOut: false,
			}
		case <-signal:
			return &FirstValidResponse{
				Response: nil,
				Err:      fmt.Errorf("Timeout after waiting for %d seconds", waitGatewayResponseFor),
				TimedOut: true,
			}
		}
	}
}

func NewServer(logger *log.Logger) *Server {
	server := Server{
		Logger: logger,
	}

	return &server
}
