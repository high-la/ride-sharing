package messaging

import (
	pb "github.com/high-la/ride-sharing/shared/proto/trip"
)

const (
	FindAvailableDriversQueue = "find_available_drivers"
	DriverCmdTripRequestQueue = "driver_cmd_trip_request"
)

type TripEventData struct {
	Trip *pb.Trip `json:"trip"`
}
