package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/high-la/ride-sharing/services/api-gateway/grpc_clients"
	"github.com/high-la/ride-sharing/shared/contracts"
)

func handleTripStart(w http.ResponseWriter, r *http.Request) {

	var reqBody startTripRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&reqBody)
	if err != nil {
		http.Error(w, "failed to parse JSON data api gateway", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	// Creating new connection is a trade-off
	// Why new connection is to be created for each connection
	// because if a service is down, to not block the whole application
	// so creating a new client for each connection
	tripService, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	defer tripService.Close()

	trip, err := tripService.Client.CreateTrip(r.Context(), reqBody.toProto())
	if err != nil {
		log.Printf("failed to start a trip: %v", err)
		http.Error(w, "failed to start a trip", http.StatusInternalServerError)
		return
	}

	response := contracts.APIResponse{Data: trip}

	writeJSON(w, http.StatusCreated, response)
}

func handleTripPreview(w http.ResponseWriter, r *http.Request) {

	var reqBody previewTripRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&reqBody)
	if err != nil {
		http.Error(w, "failed to parse JSON data api gateway", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	// simple validation
	if reqBody.UserID == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}

	// Creating new connection is a trade-off
	// Why new connection is to be created for each connection
	// because if a service is down, to not block the whole application
	// so creating a new client for each connection
	tripService, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	defer tripService.Close()

	// Calling trip service
	tripPreview, err := tripService.Client.PreviewTrip(r.Context(), reqBody.toProto())
	if err != nil {
		log.Printf("failed to preview a trip: %v", err)
		http.Error(w, "failed to preview trip", http.StatusInternalServerError)
		return
	}

	// .
	response := contracts.APIResponse{Data: tripPreview}

	writeJSON(w, http.StatusCreated, response)
}
