//go:build !noworker

package cmd

import (
	"github.com/abihf/video-upscaler/internal/worker"
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
		return worker.Run(cmd.Context())
	},
}
