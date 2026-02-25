package main

import (
	"log"
	"net/http"

	"github.com/high-la/ride-sharing/services/api-gateway/grpc_clients"
	"github.com/high-la/ride-sharing/shared/contracts"
	"github.com/high-la/ride-sharing/shared/messaging"
	"github.com/high-la/ride-sharing/shared/proto/driver"
)

var (
	connManager = messaging.NewConnectionManager()
)

func handleRidersWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {

	conn, err := connManager.Upgrade(w, r)
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

	// Add connection to manager
	connManager.Add(userID, conn)
	defer connManager.Remove(userID)

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

func handleDriversWebSocket(w http.ResponseWriter, r *http.Request, rb *messaging.RabbitMQ) {

	conn, err := connManager.Upgrade(w, r)
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

	// Add connection to manager
	connManager.Add(userID, conn)

	driverService, err := grpc_clients.NewDriverServiceClient()
	if err != nil {
		log.Fatal(err)
	}

	ctx := r.Context()

	// Closing connections
	defer func() {
		connManager.Remove(userID)

		driverService.Client.UnRegisterDriver(ctx, &driver.RegisterDriverRequest{
			DriverID:    userID,
			PackageSlug: packageSlug,
		})

		driverService.Close()

		log.Panicln("Driver unregistered: ", userID)
	}()

	driverData, err := driverService.Client.RegisterDriver(ctx, &driver.RegisterDriverRequest{
		DriverID:    userID,
		PackageSlug: packageSlug,
	})
	if err != nil {
		log.Printf("Error registering driver: %v", err)
		return
	}

	err = connManager.SendMessage(userID, contracts.WSMessage{
		Type: contracts.DriverCmdRegister,
		Data: driverData.Driver,
	})
	if err != nil {
		log.Printf("error sending message: %v", err)
		return
	}

	// Initialize queue consumers
	queues := []string{
		messaging.DriverCmdTripRequestQueue,
	}

	for _, q := range queues {
		consumer := messaging.NewQueueConsumer(rb, connManager, q)

		if err := consumer.Start(); err != nil {
			log.Printf("Failed to start consumer for queue: %s err: %v", q, err)
		}
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
