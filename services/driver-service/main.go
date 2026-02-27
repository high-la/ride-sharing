package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/high-la/ride-sharing/shared/env"
	"github.com/high-la/ride-sharing/shared/messaging"
	"github.com/high-la/ride-sharing/shared/tracing"
	grpcserver "google.golang.org/grpc"
)

var GrpcAddr = ":9092"

func main() {

	// Initialize Tracing
	tracerCfg := tracing.Config{
		ServiceName:    "driver-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://jaeger:14268/api/traces"),
	}

	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize the tracer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer sh(ctx)

	rabbitMqURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")

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

	// RabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(rabbitMqURI)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("starting RabbitMQ connection")

	// Starting the gRPC server
	grpcServer := grpcserver.NewServer(tracing.WithTracingInterceptors()...)
	NewGrpcHandler(grpcServer, service)

	consumer := NewTripConsumer(rabbitmq, service)
	go func() {

		if err := consumer.Listen(); err != nil {
			log.Fatalf("Failed to listen to the message: %v", err)
		}
	}()

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
