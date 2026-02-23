package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

var GrpcAddr = ":9092"

func main() {

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

	service := NewService()

	// Starting the gRPC server
	grpcServer := grpcserver.NewServer()
	NewGrpcHandler(grpcServer, service)

	log.Printf("starting gRPC server Driver service on port %s", lis.Addr().String())

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
