package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/high-la/ride-sharing/shared/contracts"
	"github.com/high-la/ride-sharing/shared/util"
)

var upgrader = websocket.Upgrader{

	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleRidersWebSocket(w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()

	userID := r.URL.Query().Get("userID")
	if userID == "" {
		log.Println("no user ID provided")
		return
	}

	// read message
	for {

		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("error reading message: %v", err)
			break
		}

		log.Printf("received message: %s", message)

	}
}

func handleDriversWebSocket(w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	defer conn.Close()

	userID := r.URL.Query().Get("userID")
	if userID == "" {
		log.Println("no user ID provided")
		return
	}

	packageSlug := r.URL.Query().Get("packageSlug")
	if packageSlug == "" {
		log.Println("no package slug provided")
		return
	}

	// .
	type Driver struct {
		Id             string `json:"id"`
		Name           string `json:"name"`
		ProfilePicture string `json:"profilePicture"`
		CarPlate       string `json:"carPlate"`
		PackageSlug    string `json:"packageSlug"`
	}

	msg := contracts.WSMessage{
		Type: "driver.cmd.register",
		Data: Driver{
			Id:             userID,
			Name:           "Haile",
			ProfilePicture: util.GetRandomAvatar(1),
			CarPlate:       "ABC123",
			PackageSlug:    packageSlug,
		},
	}

	err = conn.WriteJSON(msg)
	if err != nil {
		log.Printf("error sending message: %v", err)
		return
	}

	// read message
	for {

		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("error reading message: %v", err)
			break
		}

		log.Printf("received message: %s", message)

	}
}
