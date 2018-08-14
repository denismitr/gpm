package proxy

import (
	"io"
	"net/http"
	"time"
)

// FirstResponse - first response received from the multiplexer
// or an error if no proxy requests were successful
type FirstResponse struct {
	// response from the multiplexer
	Response *http.Response
	// was the response created on timeout
	timedOut bool
	// time that it took to process the request and create the response
	elapsed time.Duration
	// an error created in the request context
	err error
}

// IsValid - checks if response is valid
func (fr *FirstResponse) IsValid() bool {
	return fr.Response != nil &&
		fr.err == nil &&
		fr.Response.StatusCode >= 200 &&
		fr.Response.StatusCode < 300
}

// GetElapsedSeconds - get time elapsed since the request processing started
func (fr *FirstResponse) GetElapsedSeconds() float64 {
	return fr.elapsed.Seconds()
}

// GetError - get error object from the response
func (fr *FirstResponse) GetError() error {
	return fr.err
}

// HasTimedOut - checks if response has timed out
func (fr *FirstResponse) HasTimedOut() bool {
	return fr.timedOut
}

// GetStatusCode - get response status code
func (fr *FirstResponse) GetStatusCode() int {
	return fr.Response.StatusCode
}

// GetBody - get response body
func (fr *FirstResponse) GetBody() io.ReadCloser {
	return fr.Response.Body
}

// CloseBody - close response body
func (fr *FirstResponse) CloseBody() error {
	return fr.Response.Body.Close()
}

// GetHeader - get response header struct
func (fr *FirstResponse) GetHeader() http.Header {
	return fr.Response.Header
}

func NewValidFirstResponse(response *http.Response, elapsed time.Duration) *FirstResponse {
	return &FirstResponse{
		Response: response,
		err:      nil,
		timedOut: false,
		elapsed:  elapsed,
	}
}

func NewInvalidFirstResponse(err error, timedOut bool, elapsed time.Duration) *FirstResponse {
	return &FirstResponse{
		Response: nil,
		err:      err,
		timedOut: timedOut,
		elapsed:  elapsed,
	}
}
