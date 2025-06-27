//go:build !noworker

package cmd

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/abihf/video-upscaler/internal/model"
	"github.com/abihf/video-upscaler/internal/upscaler"
	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
)

var workerFlags struct {
	metricsExporterAddr string
	tempDir             string
}

func init() {
	rootCmd.AddCommand(workerCmd)
	workerCmd.Flags().StringVar(&workerFlags.tempDir, "temp-dir", getEnv("TEMP_DIR", "/var/cache/upscalers"), "Help message for toggle")
	workerCmd.Flags().StringVar(&workerFlags.metricsExporterAddr, "metrics-exporter", getEnv("METRICS_EXPORTER", ""), "Help message for toggle")
}

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Run upscale worker",

	RunE: func(cmd *cobra.Command, _ []string) error {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})))
		srv := asynq.NewServer(
			redisConn(),
			asynq.Config{
				BaseContext: cmd.Context,
				Concurrency: 1,
				Queues: map[string]int{
					"critical": 6,
					"default":  3,
					"low":      1,
				},
				// See the godoc for other configuration options
				ErrorHandler: asynq.ErrorHandlerFunc(func(_ context.Context, task *asynq.Task, err error) {
					var p model.VideoUpscaleTask
					json.Unmarshal(task.Payload(), &p)
					slog.With("err", err, "task", p).Error("Error processing task")
				}),
			},
		)

		u := upscaler.Handler{TempDir: workerFlags.tempDir}
		mux := asynq.NewServeMux()
		mux.Handle(model.TaskVideoUpscaleType, &u)
		return srv.Run(mux)
	},
}
