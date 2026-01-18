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

	for i := range maxRetries {
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

		if i < maxRetries-1 {
			backoff := time.Duration(1<<uint(i)) * time.Second
			slog.Warn("failed to connect to temporal, retrying", "attempt", i+1, "backoff", backoff, "error", err)
			time.Sleep(backoff)
		}
	}

	return nil, err

}
