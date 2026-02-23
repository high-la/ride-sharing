package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/high-la/ride-sharing/services/trip-service/internal/infrastructure/events"
	"github.com/high-la/ride-sharing/services/trip-service/internal/infrastructure/grpc"
	"github.com/high-la/ride-sharing/services/trip-service/internal/infrastructure/repository"
	"github.com/high-la/ride-sharing/services/trip-service/internal/service"
	"github.com/high-la/ride-sharing/shared/env"
	"github.com/high-la/ride-sharing/shared/messaging"
	grpcserver "google.golang.org/grpc"
)

var GrpcAddr = ":9093"

func main() {

	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

	inmemRepo := repository.NewInmemRepositiry()
	svc := service.NewService(inmemRepo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigChannel := make(chan os.Signal, 1)
		signal.Notify(sigChannel, os.Interrupt, syscall.SIGTERM)
		<-sigChannel
		cancel()
	}()

	//.
	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("starting RabbitMQ connection")

	publisher := events.NewTripEventPublisher(rabbitmq)

	// Starting the gRPC server
	grpcServer := grpcserver.NewServer()
	grpc.NewGRPCHandler(grpcServer, svc, publisher)

	log.Printf("starting gRPC server Trip service on port %s", lis.Addr().String())

	go func() {

		err := grpcServer.Serve(lis)
		if err != nil {
			log.Printf("failed to serve: %v", err)
			cancel()
		}
	}()

	// Wait for the shutdown signal
	<-ctx.Done()
	log.Println("shutting down the server...")
	grpcServer.GracefulStop()
}
