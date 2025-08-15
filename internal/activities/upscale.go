package activities

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.temporal.io/sdk/activity"
)

func Prepare(ctx context.Context, outFile string) (string, error) {
	info := activity.GetInfo(ctx)
	id := info.WorkflowExecution.ID
	tmpDir := filepath.Join("/media/data/upscaler", id)
	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	return tmpDir, nil
}

func MoveFile(ctx context.Context, inFile string, outFile string) error {
	err := os.Rename(inFile, outFile)
	if err != nil {
		return fmt.Errorf("failed to move file from %s to %s: %w", inFile, outFile, err)
	}

	return nil
}

func Upscale(ctx context.Context, inFile string, outFile string, tmpDir string) (string, error) {
	cacheFile := filepath.Join(tmpDir, "cache")
	tmpOut := filepath.Join(tmpDir, "out.mkv")

	// Create vspipe command
	vspipeCmd := exec.Command("vspipe", "-c", "y4m",
		"-a", fmt.Sprintf("in=%s", inFile), "-a", fmt.Sprintf("cache=%s", cacheFile),
		"/upscale/script.py", "-")

	// Create ffmpeg command
	ffmpegCmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "info", "-noautorotate",
		"-progress", "pipe:4",
		"-colorspace", "bt709", "-color_primaries", "bt709", "-color_trc", "bt709",
		"-i", "-", "-i", inFile,
		"-map_metadata", "1", "-map", "0:v:0", "-map", "1", "-map", "-1:v:0",
		"-pix_fmt", "p010le", "-c:v", "hevc_nvenc", "-profile:v", "main10", "-preset:v", "slow",
		"-rc:v", "vbr", "-cq:v", "16", "-spatial-aq", "1", "-bf", "3", "-aud", "1", "-b_ref_mode", "middle",
		"-g", "48", "-keyint_min", "48", "-forced-idr", "1", "-sc_threshold", "0", "-fflags", "+genpts", "-rc-lookahead", "20",
		"-y", tmpOut)

	ffmpegLogFile := filepath.Join(tmpDir, "ffmpeg.log")
	ffmpegLog, err := os.OpenFile(ffmpegLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open ffmpeg log file: %w", err)
	}
	defer func() {
		fmt.Fprintf(ffmpegLog, "\n\n------------\n\n")
		ffmpegLog.Close()
	}()
	ffmpegCmd.Stdout = ffmpegLog

	// Create pipe between commands
	pipe, err := vspipeCmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create pipe: %w", err)
	}
	ffmpegCmd.Stdin = pipe

	doneCh := make(chan struct{}, 1)
	cancelCh := activity.GetWorkerStopChannel(ctx)

	timer := time.NewTicker(10 * time.Second)
	prog := captureFfmpegProgress(ffmpegCmd)
	go func() {
		for {
			select {
			case <-doneCh:
				return
			case <-ctx.Done():
				return
			case <-timer.C:
				prog.mu.RLock()
				detail := fmt.Sprintf("Current FPS: %s, Time: %s", prog.fps, prog.time)
				prog.mu.RUnlock()
				activity.RecordHeartbeat(ctx, detail)
			case <-cancelCh:
				if vspipeCmd.Process != nil {
					vspipeCmd.Process.Kill()
				}
				if ffmpegCmd.Process != nil {
					ffmpegCmd.Process.Kill()
				}
			}
		}
	}()

	err = func() error {
		defer pipe.Close()
		// Start both commands
		if err := ffmpegCmd.Start(); err != nil {
			return fmt.Errorf("failed to start ffmpeg: %w", err)
		}

		if err := vspipeCmd.Start(); err != nil {
			return fmt.Errorf("failed to start vspipe: %w", err)
		}

		// Wait for vspipe to complete
		if err := vspipeCmd.Wait(); err != nil {
			return fmt.Errorf("vspipe failed: %w", err)
		}
		return nil
	}()

	if err != nil {
		return "", err
	}

	// Close the pipe and wait for ffmpeg

	if err := ffmpegCmd.Wait(); err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w", err)
	}
	doneCh <- struct{}{}

	return tmpOut, nil
}

type ffmpegProgress struct {
	mu   sync.RWMutex
	fps  string
	time string
}

func captureFfmpegProgress(cmd *exec.Cmd) *ffmpegProgress {
	r, w, _ := os.Pipe()
	cmd.ExtraFiles = append(cmd.ExtraFiles, w)
	progress := &ffmpegProgress{}
	go func() {
		br := bufio.NewReader(r)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				break
			}

			if fpsStr, ok := strings.CutPrefix(line, "fps="); ok {
				progress.mu.Lock()
				progress.fps = fpsStr
				progress.mu.Unlock()
			} else if timeStr, ok := strings.CutPrefix(line, "out_time="); ok {
				progress.mu.Lock()
				progress.time = timeStr
				progress.mu.Unlock()
			}
		}
	}()
	return progress
}
