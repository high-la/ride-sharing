package grpc_clients

import (
	"os"

	pb "github.com/high-la/ride-sharing/shared/proto/trip"
	"github.com/high-la/ride-sharing/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	dialOptions := append(
		tracing.DialOptionsWithTracing(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	conn, err := grpc.NewClient(tripServiceURL, dialOptions...)
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
