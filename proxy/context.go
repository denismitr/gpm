package proxy

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type RequestContext struct {
	originalRequest *http.Request
	FirstResponse   chan *FirstResponse
	// a buffered response channel with the capacity of max concurrent tries
	responseCh chan *http.Response
	client     *http.Client
	logger     *log.Logger
	timeout    time.Duration

	session int64

	mu   sync.Mutex
	done bool
}

func (rc *RequestContext) processRequest() {
	// extract the url from the request object
	url := rc.originalRequest.URL.String()

	// ticker to detect a time out
	timeout := time.Tick(rc.timeout + time.Second)

	defer func() {
		rc.logger.Printf("\nProcess request [session %d] method exiting...", rc.session)
	}()

	// start n goroutines to query the url
	for i := 0; i < concurrentTries; i++ {
		go func(id int) {
			defer func() {
				rc.logger.Printf("\nClient request [%d] exiting...", id)
				// just in case something goes wrong
				if r := recover(); r != nil {
					rc.logger.Println("\nError! Recovered from ", r)
				}
			}()

			// make a query
			response, err := rc.client.Get(url)
			if err != nil {
				// @TODO handle errors gracefully
				rc.logger.Printf("\nRequest to %s failed with error %s", url, err.Error())
				return
			}

			// check if response is one of 2**
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				if !rc.done {
					rc.responseCh <- response
				}
			} else {
				// @TODO handle errors gracefully
				rc.logger.Printf("\nError status %d received from %s", response.StatusCode, url)
			}
		}(i)
	}

	for {
		select {
		// catch a first 2** response
		case firstResponse := <-rc.responseCh:
			rc.logger.Printf("\nReceived success response from %s", url)
			// create a valid first response object and return
			rc.FirstResponse <- NewValidFirstResponse(firstResponse)

			rc.close()

			return
		case <-timeout:
			// on time out create an invalid first response and return
			rc.FirstResponse <- NewInvalidFirstResponse(
				fmt.Errorf("Timeout after waiting for %d seconds", waitGatewayResponseFor),
				true,
			)

			rc.close()

			return
		}
	}
}

func (rc *RequestContext) close() {
	rc.mu.Lock()
	rc.done = true
	rc.mu.Unlock()
	close(rc.responseCh)
	close(rc.FirstResponse)
}

func NewRequestContext(originalRequest *http.Request, logger *log.Logger, session int64) *RequestContext {
	timeout := time.Duration(waitGatewayResponseFor) * time.Second

	return &RequestContext{
		FirstResponse:   make(chan *FirstResponse),
		originalRequest: originalRequest,
		responseCh:      make(chan *http.Response, concurrentTries),
		logger:          logger,
		timeout:         timeout,
		client:          NewClient(proxyUrl, proxyAuth, timeout, logger),
		session:         session,
	}
}
