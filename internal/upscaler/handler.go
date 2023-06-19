package upscaler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"

	"github.com/abihf/video-upscaler/internal/model"
	"github.com/hibiken/asynq"
)

// ffprobe -v error -select_streams v:0 -count_packets -show_entries stream=nb_read_packets -of csv=p=0 /media/data/input.mkv

type Handler struct {
	TempDir string
}

func (h *Handler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p model.VideoUpscaleTask
	err := json.Unmarshal(t.Payload(), &p)
	if err != nil {
		return fmt.Errorf("can not payload json %w", err)
	}

	hash := md5.Sum([]byte(p.In))
	tempdir := path.Join(h.TempDir, hex.EncodeToString(hash[:8]), path.Base(p.In))
	ut := Task{
		Input:              p.In,
		Output:             p.Out,
		TempDir:            tempdir,
		BackgroundFinalize: true,
	}
	return ut.Upscale(ctx)
}

/*
func Run(ctx context.Context, p *model.VideoUpscaleTask) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tmpFile := "lalaaaa"

	vspipe := exec.CommandContext(ctx, "vspipe",
		"-c", "y4m", "/upscale/script.vpy",
		"-a", "in="+p.In, "-")

	ffmpeg := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error",
		"-i", "-", "-i", p.In,
		"-map_metadata", "1", "-map", "0:v:0", "-map", "1", "-map", "-1:v:0", "-c:a", "copy",
		"-c:v", "hevc_nvenc", "-profile:v", "main10", "-preset:v", "slow",
		"-rc:v", "vbr", "-qmin:v", "24", "-qmax:v", "20",
		"-progress", tmpFile+".ffprog", "-stats_period", "5",
		"-y", tmpFile)

	ffmpeg.Stderr = os.Stderr
	ffmpeg.Stdin, _ = vspipe.StdoutPipe()
	ffmpeg.Stdout = os.Stdout
	vspipe.Stdin = os.Stdin
	vspipe.Stderr = os.Stderr

	errChan := make(chan error, 2)
	go run(vspipe, errChan)
	go run(ffmpeg, errChan)

	var err error
	for i := 0; i < 2; i++ {
		err = <-errChan
		if err != nil {
			return err
		}
	}

	return os.Rename(tmpFile, p.Out)
}
*/
