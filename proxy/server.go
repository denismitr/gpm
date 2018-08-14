package proxy

import (
	"io"
	"log"
	"net/http"
	"sync/atomic"
)

// Server struct handles all the proxy related actions from serving
// incoming HTTP request to querying the proxied url,
// copying body and headers of the first response to the ResponseWriter
type Server struct {
	Logger *log.Logger

	// stores unique session number
	session int64
}

// ServeHTTP - handle HTTP request
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		s.Logger.Printf("\nServe HTTP exiting session [%d]...", s.session)
		if r := recover(); r != nil {
			s.Logger.Printf("\nError occurred during Serve HTTP! Recovered from %v", r)
		}
	}()

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

	// create new context
	requestContext, err := NewRequestContext(r, s.Logger, atomic.AddInt64(&s.session, 1))
	if err != nil {
		s.Logger.Println(err)
		// return bad gateway if no valid response arrived
		// @TODO maybe use some other format and/or status code
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("\r\nBad gateway"))
		return
	}

	// process the request and get the first good response
	// if one actually arrives
	go requestContext.processRequest()

	response := <-requestContext.FirstResponse

	// check if response is valid
	if !response.IsValid() {
		s.Logger.Println(response.GetError())
		// return bad gateway if no valid response arrived
		// @TODO maybe use some other format and/or status code
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("\r\n" + response.GetError().Error()))
		return
	}

	s.Logger.Printf(
		"Sending success response from session %d after %.3f seconds of processing", s.session, response.GetElapsedSeconds())

	// copy response with headers to ResponseWriter
	s.proxyResponse(w, response)
	requestContext.SafeClose()

	s.Logger.Printf(
		"Done. Response delivered. Session [%d] is now closed...", s.session)
}

// proxyResponse - copies the client's first response body and header into
// ResponseWriter object
func (s *Server) proxyResponse(w http.ResponseWriter, response *FirstResponse) {
	// copy all the headers
	copyHeaders(w.Header(), response.GetHeader())
	// copy status code
	w.WriteHeader(response.GetStatusCode())

	// copy body
	bytesCopied, _ := io.Copy(w, response.GetBody())
	// close body
	if err := response.CloseBody(); err != nil {
		s.Logger.Printf("Can't close response body %v", err)
	}

	s.Logger.Printf("Copied %v bytes to the client", bytesCopied)
}

// NewServer - creates a new proxy server
func NewServer(logger *log.Logger) *Server {
	server := Server{
		Logger: logger,
	}

	return &server
}
