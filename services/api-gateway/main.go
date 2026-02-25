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
	"github.com/high-la/ride-sharing/shared/messaging"
)

var (
	httpAddr    = env.GetString("HTTP_ADDR", ":8081")
	rabbitMqURI = env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
)

func main() {
	log.Println("Starting API Gateway!")

	// Create a new HTTP request multiplexer (mux) to route incoming requests to handlers.
	// Using a custom mux is preferred over http.DefaultServeMux for better control and testing.
	mux := http.NewServeMux()

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("starting RabbitMQ connection")

	// mux.HandleFunc("POST /trip/preview", enableCORS(handleTripPreview))
	// mux.HandleFunc("POST /trip/start", enableCORS(handleTripStart))

	// // WebSockets (no CORS needed normally)
	// mux.HandleFunc("/ws/drivers", handleDriversWebSocket)
	// mux.HandleFunc("/ws/riders", handleRidersWebSocket)

	mux.HandleFunc("POST /trip/preview", handleTripPreview)
	mux.HandleFunc("POST /trip/start", handleTripStart)
	mux.HandleFunc("/ws/drivers", func(w http.ResponseWriter, r *http.Request) {
		handleDriversWebSocket(w, r, rabbitmq)
	})
	mux.HandleFunc("/ws/riders", func(w http.ResponseWriter, r *http.Request) {
		handleRidersWebSocket(w, r, rabbitmq)
	})

	//
	server := &http.Server{
		Addr: httpAddr,
		// Handler: mux,
		Handler: enableCORS(mux),
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
