package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"
)

const firstRequest = 1

// Multiplexer - orchestrates making HTTP requests to the requested URL
type Multiplexer struct {
	// holds original request that came from the end user
	originalRequest *http.Request

	// URL that should be requested
	destinationURL string
	// HTTP Method that should be used
	method string

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

	// logger
	logger Logger
	// proxy list
	proxyList *List
	// max timeout
	timeout time.Duration
	// number of concurrent request that multiplexer method should produce
	concurrentTries int

	// unique session identifier
	// it can be used not only to dintinguish processes
	// but potentially to link request context with server Serve HTTP
	session      int64
	context      context.Context
	canelContext context.CancelFunc

	// List of errors
	errors []error

	// mutexes
	doneMu  sync.Mutex
	errorMu sync.Mutex
	once    sync.Once

	// Timestamps
	startedAt time.Time
}

func (m *Multiplexer) processRequest() {
	defer func() {
		m.logger.Printf("\nProcess request [context %d] method exiting...", m.session)
	}()

	// mark the begining of request processing
	m.startedAt = time.Now().UTC()

	// start n goroutines to query the url
	for i := 1; i <= m.concurrentTries; i++ {
		go m.multiplex(i)
	}

	for {
		select {
		// catch a first 2** response
		case firstResponse := <-m.responseCh:
			m.finish()

			// create a valid first response object and return
			m.FirstResponse <- NewValidFirstResponse(firstResponse, m.GetElapsedTime())

			return
		case newErr := <-m.errorCh:
			m.addError(newErr)

			if m.AllErrored() {
				m.finish()
				m.FirstResponse <- NewInvalidFirstResponse(m.GetFirstError(), false, m.GetElapsedTime())
				return
			}
		case <-m.timeoutCh:
			m.finish()

			// on time out create an invalid first response and return
			m.FirstResponse <- NewInvalidFirstResponse(
				fmt.Errorf("all requests to %s failed with timeout after waiting for %.3f seconds", m.destinationURL, m.timeout.Seconds()),
				true,
				m.GetElapsedTime())

			return
		}
	}
}

// finish - closes the doneCh and marks done flag as true
// after that RequestContext should not perform any action
// all outgoing requests should be canceled
func (m *Multiplexer) finish() {
	m.doneMu.Lock()
	defer m.doneMu.Unlock()
	m.done = true
	close(m.doneCh)

	m.logger.Printf("List of errors for session [%d]: %v", m.session, m.errors)
}

// IsDone - checks whether RequestContext is done with it's activity
func (m *Multiplexer) IsDone() bool {
	m.doneMu.Lock()
	defer m.doneMu.Unlock()
	return m.done
}

// GetErrors - retrieves final error message
func (m *Multiplexer) GetErrors() []error {
	m.errorMu.Lock()
	defer m.errorMu.Unlock()
	return m.errors
}

// AllErrored - checks if all requests have failed
func (m *Multiplexer) AllErrored() bool {
	return m.GetErrorsCount() >= m.concurrentTries
}

// GetFirstError get an error that was mostly or excludively encountered during requests
// otherwise just return that all requests failed
func (m *Multiplexer) GetFirstError() error {
	return m.errors[0]
	// if m.AllErrored() {
	// 	for _, err := range m.errors {
	// 		for _, anotherErr := range m.errors {
	// 			if err.Error() != anotherErr.Error() {
	// 				return fmt.Errorf("all requests to %s have failed", m.destinationURL)
	// 			}
	// 		}
	// 	}

	// 	return m.errors[0]
	// }

	return fmt.Errorf("all requests to %s have failed", m.destinationURL)
}

// GetErrorsCount - Get a count of errors that have occured in the multiplexer
func (m *Multiplexer) GetErrorsCount() int {
	m.errorMu.Lock()
	defer m.errorMu.Unlock()
	return len(m.errors)
}

func (m *Multiplexer) addError(err error) {
	m.errorMu.Lock()
	defer m.errorMu.Unlock()
	m.errors = append(m.errors, err)
}

// SafeClose - Safely close all the channels
func (m *Multiplexer) SafeClose() {
	m.once.Do(func() {
		close(m.responseCh)
		close(m.FirstResponse)
		close(m.errorCh)
		m.canelContext()
		m.logger.Printf("Request context [%d] in now closed", m.session)
	})

	// PrintMemUsage()
}

// GetElapsedTime - get time elapsed since the processing started
func (m *Multiplexer) GetElapsedTime() time.Duration {
	return time.Now().UTC().Sub(m.startedAt)
}

func (m *Multiplexer) createRequest() *http.Request {
	req, _ := http.NewRequest(m.method, m.destinationURL, nil)

	// dump the request to the console
	dump, _ := httputil.DumpRequest(req, false)
	fmt.Println(string(dump))

	return req.WithContext(m.context)
}

func (m *Multiplexer) multiplex(index int) {
	defer func() {
		m.logger.Printf("\nClient request [%d:%d] exiting...", m.session, index)
		// just in case something goes wrong
		if r := recover(); r != nil {
			m.logger.Printf("\nError! Recovered from %v", r)
		}
	}()

	var transport *http.Transport
	var err error

	if index == firstRequest {
		transport = NewTransport()
	} else {
		transport, err = NewProxiedTransport(m.proxyList.Rand())
		if err != nil {
			m.logger.Printf("Could not create proxied transport, falls back to default one")
		}
	}

	// create a new client
	client := NewClient(transport)
	// create a new request
	req := m.createRequest()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.logger.Printf("\nError! Recovered from %v", r)
			}
		}()

		// make a query
		response, err := client.Do(req)
		if err != nil {
			// we don't want to register an error when context has timed out
			// for any timout error there is a specialized handler
			if strings.Contains(err.Error(), "context") || strings.Contains(err.Error(), "canceled") {
				m.logger.Printf("\nRequest to %s within session [%d] got cancelled", req.URL, m.session)
			} else {
				// will save error in errors list
				m.errorOccurred(
					fmt.Errorf("\nRequest to %s failed with error [%s]", req.URL, err.Error()))
			}

			return
		}

		// check if response is one of 2**
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			if !m.IsDone() {
				m.responseCh <- response
				return
			}
			m.logger.Printf("\nResponse to request to %s already received", req.URL)
		} else {
			m.errorOccurred(fmt.Errorf("error status %d received from %s", response.StatusCode, req.URL))
		}

		// close response body of any response that was not passed to the channel
		response.Body.Close()
	}()

	select {
	case <-m.doneCh:
		m.logger.Printf("\nMultiplexer on session [%d] is done. Cancelling remaining requests", m.session)
		transport.CancelRequest(req)
		return
	case <-m.context.Done():
		m.logger.Printf("\nContext on session [%d] was cancelled. Cancelling remaining requests", m.session)
		transport.CancelRequest(req)
		return
	}
}

func (m *Multiplexer) errorOccurred(err error) {
	m.logger.Println(err)
	m.errorCh <- err
}

// NewMultiplexer - create new request context
func NewMultiplexer(
	originalRequest *http.Request,
	method string,
	destinationURL string,
	logger Logger,
	proxyList *List,
	session int64,
) (*Multiplexer, error) {
	// how many concurrent requests should be sent to destination URL
	cuncurrentTries := getConcurrentTries()
	timeout := time.Duration(GetMaxTimeout()) * time.Second

	// get context from the original request
	ctx, cancel := context.WithCancel(originalRequest.Context())

	return &Multiplexer{
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
		concurrentTries: cuncurrentTries,
		session:         session,
		destinationURL:  destinationURL,
		method:          method,
		proxyList:       proxyList,
	}, nil
}
