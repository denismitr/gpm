package main

import (
	"context"
	"gpm/proxy"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

const defaultPort = ":8081"

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	logger := log.New(os.Stdout, "", 0)
	port := resolvePort()
	proxyServer := proxy.NewServer(logger)
	handler := &http.Server{Addr: port, Handler: proxyServer}

	go func() {
		log.Printf("Listening on port %s", port)

		if err := handler.ListenAndServe(); err != nil {
			logger.Fatal(err)
		}
	}()

	<-stop

	logger.Println("\nShutting down the server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	handler.Shutdown(ctx)

	logger.Println("Server gracefully stopped")
}

func resolvePort() string {
	addr := ":" + os.Getenv("GPM_PORT")
	if addr == ":" {
		addr = defaultPort
	}

	return addr
}
