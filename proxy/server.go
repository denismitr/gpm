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

// Server struct handles all the proxy related actions from serving
// incoming HTTP request to querying the proxied url,
// copying body and headers of the first response to the ResponseWriter
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

// ServeHTTP - handle HTTP request
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

// proxyResponse - copies the client's first response body and header into
// ResponseWriter object
func (s *Server) proxyResponse(w http.ResponseWriter, response *FirstResponse) {
	copyHeaders(w.Header(), response.GetHeader())
	w.WriteHeader(response.GetStatusCode())

	bytesCopied, _ := io.Copy(w, response.GetBody())
	if err := response.CloseBody(); err != nil {
		s.Logger.Printf("Can't close response body %v", err)
	}

	s.Logger.Printf("Copied %v bytes to the client", bytesCopied)
}

// processRequest - process incoming request by creating n goroutines to query the
// incoming url
func (s *Server) processRequest(r *http.Request) *FirstResponse {
	defer func() {
		s.Logger.Println("\nProcess request method exiting...")
	}()

	// set timeout
	timeout := time.Duration(waitGatewayResponseFor) * time.Second
	// create an http client with specified proxy transport
	client := NewClient("http://103.15.60.23:8080", timeout, s.Logger)
	// extract the url from the request object
	url := r.URL.String()
	// make a new buffered response channel with the capacity of max concurrent tries
	responseCh := make(chan *http.Response, concurrentTries)
	// ticker to detect a time out
	signal := time.Tick(timeout + time.Second)

	// start n goroutines to query the url
	for i := 0; i < concurrentTries; i++ {
		go func(id int) {
			defer s.Logger.Printf("\nClient request [%d] exiting...", id)
			// make a query
			response, err := client.Get(url)
			if err != nil {
				// @TODO handle errors gracefully
				s.handleGatewayFailure(url, err)
				return
			}

			// check if response is one of 2**
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				responseCh <- response
			} else {
				// @TODO handle errors gracefully
				s.handleGatewayFailure(url, fmt.Errorf("Error status %d", response.StatusCode))
			}
		}(i)
	}

	for {
		select {
		// catch a first 2** response
		case firstResponse := <-responseCh:
			s.Logger.Printf("\nReceived success response from %s", url)
			// create a valid first response object and return
			return NewValidFirstResponse(firstResponse)
		case <-signal:
			// on time out create an invalid first response and return
			return NewInvalidFirstResponse(
				fmt.Errorf("Timeout after waiting for %d seconds", waitGatewayResponseFor),
				true,
			)
		}
	}
}

// NewServer - creates a new proxy server
func NewServer(logger *log.Logger) *Server {
	server := Server{
		Logger: logger,
	}

	return &server
}
