package main

import (
	"gpm/proxy"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const defaultPort = ":8081"

func main() {
	// get max timeout from env
	timeout := getMaxTimeout()
	logger := log.New(os.Stdout, "", log.LstdFlags)
	server := proxy.NewServer(logger)

	// initialize new router
	r := chi.NewRouter()

	// midlleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and all further
	// processing should be stopped.
	r.Use(middleware.Timeout(time.Duration(timeout) * time.Second))

	r.Route("/get", func(r chi.Router) {
		// this middleware will perform multiplexing
		// and pass response through the context
		r.Use(server.ProxyGetRequest)
		r.Get("/", server.ProxyGetResponse)
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	port := resolvePort()
	// proxyServer := proxy.NewServer(logger)
	// handler := &http.Server{Addr: port, Handler: proxyServer}

	go func() {
		log.Printf("Listening on port %s", port)

		if err := http.ListenAndServe(port, r); err != nil {
			logger.Fatal(err)
		}
	}()

	<-stop

	logger.Println("\nShutting down the server...")

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	// handler.Shutdown(ctx)

	logger.Println("Server gracefully stopped")
}

func resolvePort() string {
	addr := ":" + os.Getenv("GPM_PORT")
	if addr == ":" {
		addr = defaultPort
	}

	return addr
}

func getMaxTimeout() int {
	maxTimeout, err := strconv.Atoi(os.Getenv("GPM_MAX_TIMEOUT"))
	if err != nil {
		maxTimeout = 10 // seconds
	}

	return maxTimeout
}
