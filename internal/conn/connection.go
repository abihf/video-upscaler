package conn

import (
	"context"
	"log/slog"
	"os"

	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func DialContext(ctx context.Context) (client.Client, error) {
	hostPort := os.Getenv("TEMPORAL_ADDRESS")
	if hostPort == "" {
		hostPort = "localhost:7233"
	}
	return client.DialContext(ctx, client.Options{
		HostPort:  hostPort,
		Namespace: "upscaler",
		Logger:    slog.Default(),
		ConnectionOptions: client.ConnectionOptions{
			DialOptions: []grpc.DialOption{
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			},
		},
	})

}
