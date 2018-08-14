package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
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
	// a channel for passing errors
	errorCh chan error
	// channel to indicate when response got received and we are done
	doneCh chan struct{}
	// since go does not support checking for whether channel is closed this flag is used along with doneCh
	done bool
	// ticker to detect a time out
	timeoutCh <-chan time.Time

	client          *http.Client
	transport       *http.Transport
	proxyAuth       string
	logger          *log.Logger
	timeout         time.Duration
	concurrentTries int

	session      int64
	context      context.Context
	canelContext context.CancelFunc

	errors []error

	// mutexes
	mu   sync.Mutex
	once sync.Once

	// Timestamps
	startedAt time.Time
}

func (rc *RequestContext) processRequest() {
	defer func() {
		rc.logger.Printf("\nProcess request [context %d] method exiting...", rc.session)
	}()

	// mark the begining of request processing
	rc.startedAt = time.Now().UTC()

	// start n goroutines to query the url
	for i := 1; i <= rc.concurrentTries; i++ {
		go rc.multiplexer(i)
	}

	for {
		select {
		// catch a first 2** response
		case firstResponse := <-rc.responseCh:
			rc.finish()

			// create a valid first response object and return
			rc.FirstResponse <- NewValidFirstResponse(firstResponse, rc.GetElapsedTime())

			return
		case newErr := <-rc.errorCh:
			rc.addError(newErr)

			if rc.GetErrorsCount() > rc.concurrentTries {
				rc.finish()
				rc.FirstResponse <- NewInvalidFirstResponse(
					fmt.Errorf("all %d requests on session [%d] failed with errors", rc.concurrentTries, rc.session),
					false,
					rc.GetElapsedTime())

				return
			}
		case <-rc.timeoutCh:
			rc.finish()

			// on time out create an invalid first response and return
			rc.FirstResponse <- NewInvalidFirstResponse(
				fmt.Errorf("all requests on session [%d] failed with timeout after waiting for %.3f seconds", rc.session, rc.timeout.Seconds()),
				true,
				rc.GetElapsedTime())

			return
		}
	}
}

// finish - closes the doneCh and marks done flag as true
// after that RequestContext should not perform any action
// all outgoing requests should be canceled
func (rc *RequestContext) finish() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.done = true
	close(rc.doneCh)

	rc.logger.Printf("List of errors for session [%d]: %v", rc.session, rc.errors)
}

// IsDone - checks whether RequestContext is done with it's activity
func (rc *RequestContext) IsDone() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.done
}

// GetErrors - retrieves final error message
func (rc *RequestContext) GetErrors() []error {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.errors
}

// GetErrorsCount - Get a count of errors that have occured in the multiplexer
func (rc *RequestContext) GetErrorsCount() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return len(rc.errors)
}

func (rc *RequestContext) addError(err error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.errors = append(rc.errors, err)
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

// GetElapsedTime - get time elapsed since the processing started
func (rc *RequestContext) GetElapsedTime() time.Duration {
	return time.Now().UTC().Sub(rc.startedAt)
}

func (rc *RequestContext) createRequest() *http.Request {
	req, _ := http.NewRequest(rc.resolveMethod(), rc.resolveURL(), nil)

	// dump the request to the console
	dump, _ := httputil.DumpRequest(req, false)
	fmt.Println(string(dump))

	return req.WithContext(rc.context)
}

func (rc *RequestContext) multiplexer(index int) {
	defer func() {
		rc.logger.Printf("\nClient request [%d:%d] exiting...", rc.session, index)
		// just in case something goes wrong
		if r := recover(); r != nil {
			rc.errorOccurred(
				fmt.Errorf("\nError! Recovered from %v", r))
		}
	}()

	req := rc.createRequest()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				rc.errorOccurred(
					fmt.Errorf("\nError! Recovered from %v", r))
			}
		}()

		// make a query
		response, err := rc.client.Do(req)
		if err != nil {
			// we don't want to register an error when context has timed out
			// for timout there is a specialized handler
			if strings.Contains(err.Error(), "context") || strings.Contains(err.Error(), "canceled") {
				rc.logger.Printf("\nRequest to %s within session [%d] got cancelled", req.URL, rc.session)
			} else {
				rc.errorOccurred(
					fmt.Errorf("\nRequest to %s failed with error [%s]", req.URL, err.Error()))
			}

			return
		}

		// check if response is one of 2**
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			if !rc.IsDone() {
				rc.responseCh <- response
				return
			}
			rc.logger.Printf("\nResponse to request to %s already received", req.URL)
		} else {
			rc.errorOccurred(
				fmt.Errorf(
					"\nError status %d received from %s on session [%d]",
					response.StatusCode,
					req.URL,
					rc.session))
		}

		response.Body.Close()
	}()

	select {
	case <-rc.doneCh:
		rc.logger.Printf("Multiplexer on session [%d] is done. Cancelling remaining requests", rc.session)
		rc.transport.CancelRequest(req)
		return
	case <-rc.context.Done():
		rc.logger.Printf("Context on session [%d] was cancelled. Cancelling remaining requests", rc.session)
		rc.transport.CancelRequest(req)
		return
	case <-rc.timeoutCh:
		rc.logger.Printf("Timeout received on session [%d]. Cancelling remaining requests", rc.session)
		rc.transport.CancelRequest(req)
		return
	}
}

func (rc *RequestContext) errorOccurred(err error) {
	rc.logger.Println(err)
	rc.errorCh <- err
}

func (rc *RequestContext) resolveURL() string {
	return rc.originalRequest.URL.String()
}

func (rc *RequestContext) resolveMethod() string {
	return rc.originalRequest.Method
}

// NewRequestContext - create new request context
func NewRequestContext(originalRequest *http.Request, logger *log.Logger, session int64) (*RequestContext, error) {
	// it is better to initialize proxy here, so that
	// if env variables change service does not have to get restarted
	// proxyURL, err := url.Parse(getProxyStr())
	// if err != nil {
	// 	return nil, err
	// }

	tlsClientSkipVerify := &tls.Config{InsecureSkipVerify: true}

	timeout := time.Duration(getMaxTimeout()) * time.Second
	transport := &http.Transport{TLSClientConfig: tlsClientSkipVerify}
	// transport.Proxy = http.ProxyURL(proxyURL)
	ctx, cancel := context.WithTimeout(originalRequest.Context(), timeout)

	cuncurrentTries := getConcurrentTries()

	return &RequestContext{
		FirstResponse:   make(chan *FirstResponse),
		originalRequest: originalRequest,
		context:         ctx,
		canelContext:    cancel,
		// to prevent race condition the response channel must be of size cuncurrentTries
		responseCh:      make(chan *http.Response, cuncurrentTries),
		doneCh:          make(chan struct{}),
		errorCh:         make(chan error),
		logger:          logger,
		timeoutCh:       time.Tick(timeout + time.Second),
		timeout:         timeout,
		transport:       transport,
		client:          NewClient(transport),
		concurrentTries: cuncurrentTries,
		session:         session,
		proxyAuth:       getProxyAuth(),
	}, nil
}
