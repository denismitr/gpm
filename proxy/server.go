package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

// Server struct handles all the proxy related actions from serving
// incoming HTTP request to querying the proxied url,
// copying body and headers of the first response to the ResponseWriter
type Server struct {
	logger Logger

	// stores unique session number
	session int64
}

type contextKey string

func (c contextKey) String() string {
	return "proxy context key " + string(c)
}

var (
	responseKey = contextKey("response")
	sessionKey  = contextKey("session")
)

// ProxyGET - middleware that will perform multiplexing
// and will place response object in to the context
func (s *Server) ProxyGetRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			s.logger.Printf("\nMultiplexer GET middleware exiting session [%d]...", s.session)

			if rec := recover(); rec != nil {
				msg := fmt.Sprintf("Internal server error occurred. Recovered from %v", rec)
				http.Error(w, msg, http.StatusInternalServerError)
			}
		}()

		_, err := ParseURLArgument(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// create new context
		requestContext, err := NewMultiplexer(r, s.logger, atomic.AddInt64(&s.session, 1))
		if err != nil {
			s.logger.Println(err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		// process the request and get the first good response
		// if one actually arrives
		go requestContext.processRequest()

		response := <-requestContext.FirstResponse
		requestContext.SafeClose()

		s.logger.Printf("Done. Response for session %d received.", s.session)

		ctx := context.WithValue(r.Context(), responseKey, response)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ProxyGetResponse - handle HTTP GET request
func (s *Server) ProxyGetResponse(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			msg := fmt.Sprintf("Internal error occurred. Recovered from %v", rec)
			http.Error(w, msg, http.StatusBadGateway)
		}
	}()

	ctx := r.Context()
	response, ok := ctx.Value(responseKey).(*FirstResponse)
	if !ok {
		http.Error(w, "Multiplexer failed to deliver any response", 502)
		return
	}

	s.proxyResponse(w, response)
}

// proxyResponse - copies the client's first response body and header into
// ResponseWriter object
func (s *Server) proxyResponse(w http.ResponseWriter, response *FirstResponse) {
	// check if response is valid
	if !response.IsValid() {
		s.logger.Println(response.GetError())
		http.Error(w, response.GetError().Error(), http.StatusBadGateway)
		return
	}

	// copy all the headers
	copyHeaders(w.Header(), response.GetHeader())
	// copy status code
	w.WriteHeader(response.GetStatusCode())

	// copy body
	bytesCopied, _ := io.Copy(w, response.GetBody())
	// close body
	if err := response.CloseBody(); err != nil {
		s.logger.Printf("Can't close response body %v", err)
	}

	s.logger.Printf("Copied %v bytes to the client. All done.", bytesCopied)
}

// NewServer - creates a new proxy server
func NewServer(logger Logger) *Server {
	server := Server{
		logger: logger,
	}

	return &server
}
