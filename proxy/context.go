package proxy

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

type RequestContext struct {
	originalRequest *http.Request
	firstResponse   *FirstResponse
	// a buffered response channel with the capacity of max concurrent tries
	responseCh chan *http.Response
	client     *http.Client
	logger     *log.Logger
	timeout    time.Duration

	session int64
}

func (rc *RequestContext) processRequest() *FirstResponse {
	// extract the url from the request object
	url := rc.originalRequest.URL.String()

	// ticker to detect a time out
	signal := time.Tick(rc.timeout + time.Second)

	defer func() {
		rc.logger.Println("\nProcess request method exiting...")
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
				rc.responseCh <- response
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

func NewRequestContext(originalRequest *http.Request, logger *log.Logger, session int64) *RequestContext {
	timeout := time.Duration(waitGatewayResponseFor) * time.Second

	return &RequestContext{
		originalRequest: originalRequest,
		responseCh:      make(chan *http.Response, concurrentTries),
		logger:          logger,
		timeout:         timeout,
		client:          NewClient(proxyUrl, proxyAuth, timeout, logger),
		session:         session,
	}
}
