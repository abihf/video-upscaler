package activities

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/abihf/video-upscaler/internal/utils/await"
	"github.com/abihf/video-upscaler/internal/utils/ffprog"
	"go.temporal.io/sdk/activity"
)

func Upscale(ctx context.Context, inFile string, tmpDir string, fileInfo FileInfo) (string, error) {
	cacheFile := getCacheFile(tmpDir)
	tmpOut := filepath.Join(tmpDir, "upscaled.mkv")

	// Create vspipe command
	vspipe := exec.CommandContext(ctx, "vspipe", "-c", "y4m",
		"-a", fmt.Sprintf("in=%s", inFile), "-a", fmt.Sprintf("cache=%s", cacheFile),
		"/upscale/script.py", "-")

	fps := "24000/1001"
	if val, ok := fileInfo["FPS"]; ok {
		val = strings.Split(val, " ")[0]
		if val != "" {
			fps = val
		}
	}

	// Create ffmpeg command
	ffmpeg := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "info", "-noautorotate",
		"-progress", "pipe:3", "-nostats",
		"-colorspace", "bt709", "-color_primaries", "bt709", "-color_trc", "bt709", // force b709
		"-i", "-",
		// video encoding settings
		"-c:v", "av1_nvenc",
		"-pix_fmt", "p010le",
		"-r", fps,
		"-preset", "p7",
		"-tune", "hq",
		"-rc", "vbr", "-cq", "18", "-b:v", "30M", "-maxrate:v", "40M", "-bufsize:v", "80M",
		"-multipass", "fullres",
		"-spatial-aq", "1", "-temporal-aq", "1", "-aq-strength", "6",
		"-rc-lookahead", "32",
		"-bf", "4", "-b_ref_mode", "middle",
		"-g", "240", "-keyint_min", "24",
		"-fflags", "+genpts",
		"-muxpreload", "0", "-muxdelay", "0", "-avoid_negative_ts", "make_zero", "-start_at_zero",
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
				if vspipe.Process != nil && vspipe.ProcessState == nil {
					vspipe.Process.Kill()
				}
				if ffmpeg.Process != nil && ffmpeg.ProcessState == nil {
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
