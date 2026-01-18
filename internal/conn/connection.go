package conn

import (
	"context"
	"log/slog"
	"os"
	"time"

	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func DialContext(ctx context.Context) (client.Client, error) {
	hostPort := os.Getenv("TEMPORAL_ADDRESS")
	if hostPort == "" {
		hostPort = "localhost:7233"
	}
	var c client.Client
	var err error
	maxRetries := 5
	backoff := time.Second

	for i := range maxRetries - 1 {
		c, err = client.DialContext(ctx, client.Options{
			HostPort:  hostPort,
			Namespace: "upscaler",
			Logger:    slog.Default(),
			ConnectionOptions: client.ConnectionOptions{
				DialOptions: []grpc.DialOption{
					grpc.WithTransportCredentials(insecure.NewCredentials()),
				},
			},
		})
		if err == nil {
			return c, nil
		}

		slog.Warn("failed to connect to temporal, retrying", "attempt", i+1, "backoff", backoff, "error", err)
		time.Sleep(backoff)
		backoff <<= 1
	}

	return nil, err

}
