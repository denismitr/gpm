package proxy

import (
	"io"
	"net/http"
)

type FirstResponse struct {
	err      error
	Response *http.Response
	timedOut bool
}

func (fr *FirstResponse) IsValid() bool {
	return fr.Response != nil &&
		fr.err == nil &&
		fr.Response.StatusCode >= 200 &&
		fr.Response.StatusCode < 300
}

func (fr *FirstResponse) GetError() error {
	return fr.err
}

func (fr *FirstResponse) HasTimedOut() bool {
	return fr.timedOut
}

func (fr *FirstResponse) GetStatusCode() int {
	return fr.Response.StatusCode
}

func (fr *FirstResponse) GetBody() io.ReadCloser {
	return fr.Response.Body
}

func (fr *FirstResponse) CloseBody() error {
	return fr.Response.Body.Close()
}

func (fr *FirstResponse) GetHeader() http.Header {
	return fr.Response.Header
}

func NewValidFirstResponse(response *http.Response) *FirstResponse {
	return &FirstResponse{
		Response: response,
		err:      nil,
		timedOut: false,
	}
}

func NewInvalidFirstResponse(err error, timedOut bool) *FirstResponse {
	return &FirstResponse{
		Response: nil,
		err:      err,
		timedOut: timedOut,
	}
}
