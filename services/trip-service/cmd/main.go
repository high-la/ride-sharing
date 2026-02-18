package main

import (
	"log"
	"net/http"

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

	err := server.ListenAndServe()
	if err != nil {
		log.Printf("HTTP server error: %v", err)
	}
}
