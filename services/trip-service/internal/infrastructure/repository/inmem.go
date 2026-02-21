package repository

import (
	"context"

	"github.com/high-la/ride-sharing/services/trip-service/internal/domain"
)

type inmemRepository struct {
	trips     map[string]*domain.TripModel
	rideFares map[string]*domain.RideFareModel
}

func NewInmemRepositiry() *inmemRepository {

	return &inmemRepository{
		trips:     make(map[string]*domain.TripModel),     // create empty map
		rideFares: make(map[string]*domain.RideFareModel), // create empty map
	}
}

func (r *inmemRepository) CreateTrip(ctx context.Context, trip *domain.TripModel) (*domain.TripModel, error) {

	r.trips[trip.ID.Hex()] = trip //.Hex is a trick to change mongo object type to string

	return trip, nil
}

func (r *inmemRepository) SaveRideFare(ctx context.Context, fare *domain.RideFareModel) error {

	r.rideFares[fare.ID.Hex()] = fare

	return nil
}
