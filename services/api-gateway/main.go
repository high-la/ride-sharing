package main

import (
	"log"
	"net/http"

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

	err := server.ListenAndServe()
	if err != nil {
		log.Printf("HTTP server error: %v", err)
	}
}
