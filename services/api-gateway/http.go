package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/high-la/ride-sharing/shared/contracts"
)

func handleTripPreview(w http.ResponseWriter, r *http.Request) {

	var reqBody previewTripRequest

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "failed to parse JSON data", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	// simple validation
	if reqBody.UserID == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}

	// TODO: Call trip service
	jsonBody, _ := json.Marshal(reqBody)
	reader := bytes.NewReader(jsonBody)

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
