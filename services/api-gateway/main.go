package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/high-la/ride-sharing/shared/env"
)

var (
	httpAddr = env.GetString("HTTP_ADDR", ":8081")
)

func main() {
	log.Println("Starting API Gateway!")

	// Create a new HTTP request multiplexer (mux) to route incoming requests to handlers.
	// Using a custom mux is preferred over http.DefaultServeMux for better control and testing.
	mux := http.NewServeMux()

	mux.HandleFunc("POST /trip/preview", handleTripPreview)

	//
	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	serverErrors := make(chan error, 1)

	go func() {

		log.Printf("server listening on %s", httpAddr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("error starting the server : %v", err)
	case sig := <-shutdown:
		log.Printf("server is shutdown due to the %v signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := server.Shutdown(ctx)
		if err != nil {
			log.Printf("could not stop server gracefully: %v", err)
			server.Close()
		}
	}
}
