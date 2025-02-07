package upscaler

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"

	"github.com/abihf/video-upscaler/internal/model"
	"github.com/hibiken/asynq"
)

type Handler struct {
	TempDir string
}

var fileNameNormalizer = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func (h *Handler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p model.VideoUpscaleTask
	err := json.Unmarshal(t.Payload(), &p)
	if err != nil {
		return fmt.Errorf("can not decode payload %w", err)
	}

	tempName := fileNameNormalizer.ReplaceAllString(path.Base(p.Out), "_")
	ut := Task{
		Input:   p.In,
		Output:  p.Out,
		TempDir: path.Join(h.TempDir, tempName),
	}
	return ut.Upscale(ctx)
}
