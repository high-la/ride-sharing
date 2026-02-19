package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	h "github.com/high-la/ride-sharing/services/trip-service/internal/infrastructure/http"
	"github.com/high-la/ride-sharing/services/trip-service/internal/infrastructure/repository"
	"github.com/high-la/ride-sharing/services/trip-service/internal/service"
)

func main() {

	inmemRepo := repository.NewInmemRepositiry()

	svc := service.NewService(inmemRepo)

	// .
	mux := http.NewServeMux()

	httpHandler := h.HttpHandler{Service: svc}

	mux.HandleFunc("POST /preview", httpHandler.HandleTripPreview)

	//
	server := &http.Server{
		Addr:    ":8083",
		Handler: mux,
	}

	serverErrors := make(chan error, 1)

	go func() {

		log.Printf("server listening on %s", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("error starting server: %v", err)

	case sig := <-shutdown:
		log.Printf("server is shutitng down due to %v signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := server.Shutdown(ctx)
		if err != nil {
			log.Printf("could not stop server gracefully: %v", err)
			server.Close()
		}
	}
}
