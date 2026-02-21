package domain

import (
	"time"

	pb "github.com/high-la/ride-sharing/shared/proto/trip"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RideFareModel struct {
	ID                primitive.ObjectID
	UserID            string
	PackageSlug       string // e.g: van, vits, BYD, luxury
	TotalPriceInCents float64
	ExpiresAt         time.Time
}

func (r *RideFareModel) ToProto() *pb.RideFare {

	return &pb.RideFare{
		Id:                r.ID.Hex(),
		UserID:            r.UserID,
		PackageSlug:       r.PackageSlug,
		TotalPriceInCents: r.TotalPriceInCents,
	}
}

func ToRideFaresProto(fares []*RideFareModel) []*pb.RideFare {

	var protoFares []*pb.RideFare

	for _, f := range fares {

		protoFares = append(protoFares, f.ToProto())
	}

	return protoFares
}
