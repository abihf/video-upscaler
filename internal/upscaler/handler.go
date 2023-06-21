package upscaler

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"

	"github.com/abihf/video-upscaler/internal/model"
	"github.com/hibiken/asynq"
)

type Handler struct {
	TempDir string
}

func (h *Handler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p model.VideoUpscaleTask
	err := json.Unmarshal(t.Payload(), &p)
	if err != nil {
		return fmt.Errorf("can not decode payload %w", err)
	}

	hash := sha1.Sum([]byte(p.In))
	b64 := base64.RawURLEncoding.EncodeToString(hash[:])
	tempdir := path.Join(h.TempDir, b64[:2], b64[2:])
	ut := Task{
		Input:   p.In,
		Output:  p.Out,
		TempDir: tempdir,
	}
	return ut.Upscale(ctx)
}
