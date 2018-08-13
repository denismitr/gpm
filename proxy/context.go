package proxy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RequestContext - orchestrates making HTTP requests to the requested URL
type RequestContext struct {
	// holds original request that came from the end user
	originalRequest *http.Request

	// channel for passing the first response from the multiple requests
	FirstResponse chan *FirstResponse
	// a buffered response channel with the capacity of max concurrent tries
	responseCh chan *http.Response
	// channel to indicate when response got received and we are done
	doneCh chan struct{}
	// ticker to detect a time out
	timeoutCh <-chan time.Time

	client    *http.Client
	transport *http.Transport
	logger    *log.Logger
	timeout   time.Duration

	session      int64
	context      context.Context
	canelContext context.CancelFunc

	// mutexes
	mu   sync.Mutex
	once sync.Once
	// indicates whether response was received
	done bool
}

func (rc *RequestContext) processRequest() {
	defer func() {
		rc.logger.Printf("\nProcess request [context %d] method exiting...", rc.session)
	}()

	// start n goroutines to query the url
	for i := 1; i <= concurrentTries; i++ {
		go rc.makeRequest(i)
	}

	for {
		select {
		// catch a first 2** response
		case firstResponse := <-rc.responseCh:
			rc.markAsDone()

			// create a valid first response object and return
			rc.FirstResponse <- NewValidFirstResponse(firstResponse)

			return
		case <-rc.timeoutCh:
			rc.markAsDone()

			// on time out create an invalid first response and return
			rc.FirstResponse <- NewInvalidFirstResponse(
				fmt.Errorf("Timeout on session [%d] after waiting for %d seconds", rc.session, waitGatewayResponseFor),
				true,
			)

			return
		}
	}
}

func (rc *RequestContext) markAsDone() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.done = true
}

func (rc *RequestContext) IsDone() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.done
}

// SafeClose - Safely close all the channels
func (rc *RequestContext) SafeClose() {
	rc.once.Do(func() {
		close(rc.responseCh)
		close(rc.FirstResponse)
		rc.canelContext()
		rc.logger.Printf("Request context [%d] in now closed", rc.session)
	})
}

func (rc *RequestContext) createRequest() *http.Request {
	req, _ := http.NewRequest(rc.resolveMethod(), rc.resolveUrl(), nil)

	return req.WithContext(rc.context)
}

func (rc *RequestContext) makeRequest(index int) {
	defer func() {
		rc.logger.Printf("\nClient request [%d:%d] exiting...", rc.session, index)
		// just in case something goes wrong
		if r := recover(); r != nil {
			rc.logger.Println("\nError! Recovered from ", r)
		}
	}()

	req := rc.createRequest()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				rc.logger.Println("\nError! Recovered from ", r)
			}
		}()

		// make a query
		response, err := rc.client.Do(req)
		if err != nil {
			if strings.Contains(err.Error(), "context canceled") {
				rc.logger.Printf("\nRequest to %s within session [%d] got cancelled", req.URL, rc.session)
			} else {
				rc.logger.Printf("\nRequest to %s failed with error [%s]", req.URL, err.Error())
			}
			return
		}

		// check if response is one of 2**
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			if !rc.IsDone() {
				rc.responseCh <- response
				close(rc.doneCh)
				return
			}
			rc.logger.Printf("\nResponse to request to %s already received", req.URL)
		} else {
			rc.logger.Printf("\nError status %d received from %s", response.StatusCode, req.URL)
		}

		response.Body.Close()
	}()

	select {
	case <-rc.doneCh:
		rc.logger.Printf("Job on session [%d] is done. Cancelling remaining requests", rc.session)
		rc.transport.CancelRequest(req)
		return
	case <-rc.context.Done():
		rc.logger.Printf("Context on session [%d] is done. Cancelling remaining requests", rc.session)
		rc.transport.CancelRequest(req)
		return
	case <-rc.timeoutCh:
		rc.logger.Printf("Timeout received on session [%d]. Cancelling remaining requests", rc.session)
		rc.transport.CancelRequest(req)
		return
	}
}

func (rc *RequestContext) resolveUrl() string {
	return rc.originalRequest.URL.String()
}

func (rc *RequestContext) resolveMethod() string {
	return rc.originalRequest.Method
}

// NewRequestContext - create new request context
func NewRequestContext(originalRequest *http.Request, logger *log.Logger, session int64) *RequestContext {
	timeout := time.Duration(waitGatewayResponseFor) * time.Second
	transport := &http.Transport{}
	ctx, cancel := context.WithTimeout(originalRequest.Context(), timeout)

	return &RequestContext{
		FirstResponse:   make(chan *FirstResponse),
		originalRequest: originalRequest,
		context:         ctx,
		canelContext:    cancel,
		responseCh:      make(chan *http.Response, 1),
		doneCh:          make(chan struct{}),
		logger:          logger,
		timeoutCh:       time.Tick(timeout + time.Second),
		timeout:         timeout,
		transport:       transport,
		client:          NewClient(transport),
		session:         session,
	}
}
