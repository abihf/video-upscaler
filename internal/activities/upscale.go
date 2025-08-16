package activities

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/abihf/video-upscaler/internal/utils/await"
	"github.com/abihf/video-upscaler/internal/utils/ffprog"
	"go.temporal.io/sdk/activity"
)

func Upscale(ctx context.Context, inFile string, outFile string, tmpDir string) (string, error) {
	cacheFile := filepath.Join(tmpDir, "cache")
	tmpOut := filepath.Join(tmpDir, "out.mkv")

	// Create vspipe command
	vspipe := exec.Command("vspipe", "-c", "y4m",
		"-a", fmt.Sprintf("in=%s", inFile), "-a", fmt.Sprintf("cache=%s", cacheFile),
		"/upscale/script.py", "-")

	// Create ffmpeg command
	ffmpeg := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "info", "-noautorotate",
		"-progress", "pipe:3", "-stats_period", "10",
		"-colorspace", "bt709", "-color_primaries", "bt709", "-color_trc", "bt709", // force b709
		"-i", "-", "-i", inFile, // take input from stdin and source file
		"-map_metadata", "1", "-map", "0:v:0", "-map", "1", "-map", "-1:v:0", // take video from stdin and other streams from source file
		"-pix_fmt", "p010le", "-c:v", "hevc_nvenc", "-profile:v", "main10", "-preset:v", "slow",
		"-rc:v", "vbr", "-cq:v", "16", "-spatial-aq", "1", "-bf", "3", "-aud", "1", "-b_ref_mode", "middle",
		"-g", "48", "-keyint_min", "48", "-forced-idr", "1", "-sc_threshold", "0", "-fflags", "+genpts", "-rc-lookahead", "20",
		"-y", tmpOut)

	// Create pipe between commands
	pipe, err := vspipe.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create pipe: %w", err)
	}
	ffmpeg.Stdin = pipe
	defer pipe.Close()

	ffLog, err := os.OpenFile(filepath.Join(tmpDir, "ffmpeg.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create ffmpeg log file: %w", err)
	}
	defer ffLog.Close()
	ffmpeg.Stdout = ffLog
	ffmpeg.Stderr = ffLog
	fmt.Fprintf(ffLog, "Running command: %s\n", ffmpeg.String())
	defer fmt.Fprintf(ffLog, "\n-----------------------------\n")

	ticker := time.NewTicker(10 * time.Second)
	progress := ffprog.Start()
	ffmpeg.ExtraFiles = append(ffmpeg.ExtraFiles, progress.Writer)
	defer progress.Close()

	doneCh := make(chan struct{}, 1)
	defer close(doneCh)

	go func() {
		for {
			select {
			case <-doneCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				activity.RecordHeartbeat(ctx, progress.String())

			case <-activity.GetWorkerStopChannel(ctx):
				if vspipe.Process != nil {
					vspipe.Process.Kill()
				}
				if ffmpeg.Process != nil {
					ffmpeg.Process.Kill()
				}
			}
		}
	}()

	err = await.All(func(cmd *exec.Cmd) error {
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("%s error: %w", cmd.Path, err)
		}
		return nil
	}, vspipe, ffmpeg)
	if err != nil {
		return "", err
	}

	return tmpOut, nil
}
