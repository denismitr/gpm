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

	response := s.processRequest(r)

	if !response.IsValid() {
		s.Logger.Println(response.GetError())
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("\r\nBad gateway"))
		return
	} else {
		s.Logger.Println("Sending success response")

		s.proxyResponse(w, response)
	}

	s.Logger.Println("Done. Response delivered...")
}

func (s *Server) proxyResponse(w http.ResponseWriter, response *FirstResponse) {
	copyHeaders(w.Header(), response.GetHeader())
	w.WriteHeader(response.GetStatusCode())

	bytesCopied, _ := io.Copy(w, response.GetBody())
	if err := response.CloseBody(); err != nil {
		s.Logger.Printf("Can't close response body %v", err)
	}

	s.Logger.Printf("Copied %v bytes to the client", bytesCopied)
}

func (s *Server) processRequest(r *http.Request) *FirstResponse {
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
			return NewValidFirstResponse(firstResponse)
		case <-signal:
			return NewInvalidFirstResponse(
				fmt.Errorf("Timeout after waiting for %d seconds", waitGatewayResponseFor),
				true,
			)
		}
	}
}

func NewServer(logger *log.Logger) *Server {
	server := Server{
		Logger: logger,
	}

	return &server
}
