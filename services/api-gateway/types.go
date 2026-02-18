package main

import "github.com/high-la/ride-sharing/shared/types"

type previewTripRequest struct {
	UserID      string           `json:"UserID"`
	Pickup      types.Coordinate `json:"pickup"`
	Destination types.Coordinate `json:"destination"`
}
