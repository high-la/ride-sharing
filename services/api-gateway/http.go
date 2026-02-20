package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/high-la/ride-sharing/services/api-gateway/grpc_clients"
	"github.com/high-la/ride-sharing/shared/contracts"
)

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

	// Calling trip service
	jsonBody, _ := json.Marshal(reqBody)
	reader := bytes.NewReader(jsonBody)

	tripService, err := grpc_clients.NewTripServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	defer tripService.Close()

	resp, err := http.Post("http://trip-service:8083/preview", "application/json", reader)
	if err != nil {
		log.Println(err)
		return
	}

	defer resp.Body.Close()

	var respBody any
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		http.Error(w, "failed to parse JSON data from trip service", http.StatusBadRequest)
		return
	}

	// .
	response := contracts.APIResponse{Data: respBody}

	writeJSON(w, http.StatusCreated, response)
}
