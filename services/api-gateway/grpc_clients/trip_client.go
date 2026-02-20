package grpc_clients

import (
	"os"

	pb "github.com/high-la/ride-sharing/shared/proto/trip"
	"google.golang.org/grpc"
)

type tripServiceClient struct {
	Client pb.TripServiceClient
	Conn   *grpc.ClientConn
}

func NewTripServiceClient() (*tripServiceClient, error) {

	tripServiceURL := os.Getenv("TRIP_SERVICE_URL")
	if tripServiceURL == "" {
		tripServiceURL = "trip-service:9093"
	}

	conn, err := grpc.NewClient(tripServiceURL)
	if err != nil {
		return nil, err
	}

	client := pb.NewTripServiceClient(conn)

	return &tripServiceClient{
		Client: client,
		Conn:   conn,
	}, nil
}

func (c *tripServiceClient) Close() {

	if c.Conn != nil {

		err := c.Conn.Close()
		if err != nil {
			return
		}
	}
}
