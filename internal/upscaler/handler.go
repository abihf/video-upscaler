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
		Input:   p.In,
		Output:  p.Out,
		TempDir: tempdir,
	}
	return ut.Upscale(ctx)
}
