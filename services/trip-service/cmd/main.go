package main

import (
	"context"
	"log"
	"time"

	"github.com/high-la/ride-sharing/services/trip-service/internal/domain"
	"github.com/high-la/ride-sharing/services/trip-service/internal/infrastructure/repository"
	"github.com/high-la/ride-sharing/services/trip-service/internal/service"
)

func main() {

	ctx := context.Background()

	inmemRepo := repository.NewInmemRepositiry()

	svc := service.NewService(inmemRepo)

	fare := &domain.RideFareModel{
		UserID: "42",
	}

	t, err := svc.CreateTrip(ctx, fare)
	if err != nil {
		log.Println(err)
	}

	log.Println(t)

	// keep the program running for now
	for {
		time.Sleep(time.Second)
	}
}
