//go:build !noworker

package cmd

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/abihf/video-upscaler/internal/ffmet"
	"github.com/abihf/video-upscaler/internal/model"
	"github.com/abihf/video-upscaler/internal/upscaler"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

var workerFlags struct {
	metricsExporterAddr string
	tempDir             string
}

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,

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

		if workerFlags.metricsExporterAddr != "" {
			ffmet.Active = true
			go runMetricsServer()
		}

		u := upscaler.Handler{TempDir: workerFlags.tempDir}
		mux := asynq.NewServeMux()
		mux.Handle(model.TaskVideoUpscaleType, &u)
		return srv.Run(mux)
	},
}

func runMetricsServer() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "OK")
	})
	http.ListenAndServe(workerFlags.metricsExporterAddr, mux)
}

func init() {
	rootCmd.AddCommand(workerCmd)
	workerCmd.Flags().StringVar(&workerFlags.tempDir, "temp-dir", getEnv("TEMP_DIR", "/var/cache/upscalers"), "Help message for toggle")
	workerCmd.Flags().StringVar(&workerFlags.metricsExporterAddr, "metrics-exporter", getEnv("METRICS_EXPORTER", ""), "Help message for toggle")
}
