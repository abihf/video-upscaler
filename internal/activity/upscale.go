package activity

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tact "go.temporal.io/sdk/activity"
)

//	vspipe -c y4m -a "in=${src}" -a "cache=${tmpdir}/cache" /upscale/script.py - | ffmpeg -hide_banner -loglevel info \
//	  -noautorotate -colorspace bt709 -color_primaries bt709 -color_trc bt709 \
//	  -i - -i "${src}" -map_metadata 1 -map 0:v:0 -map 1 -map -1:v:0 \
//	  -pix_fmt p010le -c:v hevc_nvenc -profile:v main10 -preset:v slow -rc:v vbr -cq:v 16 -spatial-aq 1 -bf 3 -aud 1 -b_ref_mode middle -g 48 -keyint_min 48 -forced-idr 1 -sc_threshold 0 -fflags +genpts -rc-lookahead 20 \
//	  -y "${tmpfile}"
func Upscale(ctx context.Context, inFile string, outFile string) (string, error) {
	info := tact.GetInfo(ctx)
	id := info.WorkflowExecution.ID
	tmpDir := filepath.Join("/media/data/upscaler", id)
	os.MkdirAll(tmpDir, 0755)

	cacheDir := filepath.Join(tmpDir, "cache")
	tmpOut := filepath.Join(tmpDir, "out.mkv")
	os.MkdirAll(cacheDir, 0755)

	// Create vspipe command
	vspipeCmd := exec.Command("vspipe", "-c", "y4m",
		"-a", fmt.Sprintf("in=%s", inFile), "-a", fmt.Sprintf("cache=%s", cacheDir),
		"/upscale/script.py", "-")

	// Create ffmpeg command
	ffmpegCmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "info", "-noautorotate",
		"-colorspace", "bt709", "-color_primaries", "bt709", "-color_trc", "bt709",
		"-i", "-", "-i", inFile,
		"-map_metadata", "1", "-map", "0:v:0", "-map", "1", "-map", "-1:v:0",
		"-pix_fmt", "p010le", "-c:v", "hevc_nvenc", "-profile:v", "main10", "-preset:v", "slow",
		"-rc:v", "vbr", "-cq:v", "16", "-spatial-aq", "1", "-bf", "3", "-aud", "1", "-b_ref_mode", "middle",
		"-g", "48", "-keyint_min", "48", "-forced-idr", "1", "-sc_threshold", "0", "-fflags", "+genpts", "-rc-lookahead", "20",
		"-y", tmpOut)

	// Create pipe between commands
	pipe, err := vspipeCmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create pipe: %w", err)
	}
	ffmpegCmd.Stdin = pipe

	doneCh := make(chan struct{}, 1)
	cancelCh := tact.GetWorkerStopChannel(ctx)

	go func() {
		select {
		case <-doneCh:
			return
		case <-cancelCh:
			if vspipeCmd.Process != nil {
				vspipeCmd.Process.Kill()
			}
			if ffmpegCmd.Process != nil {
				ffmpegCmd.Process.Kill()
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

	err = os.Rename(tmpOut, outFile)
	if err != nil {
		return "", fmt.Errorf("failed to move output file: %w", err)
	}

	return fmt.Sprintf("Upscaled %s to %s", inFile, outFile), nil
}
