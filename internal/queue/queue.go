package queue

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/abihf/video-upscaler/internal/model"
	"github.com/hibiken/asynq"
)

func Add(ctx context.Context, client *asynq.Client, in, out, priority string) error {

	payload, _ := json.Marshal(model.VideoUpscaleTask{
		In:  in,
		Out: out,
	})

	id := sha1.Sum([]byte(out))
	task := asynq.NewTask(model.TaskVideoUpscaleType, payload,
		asynq.Timeout(3*time.Hour),
		asynq.MaxRetry(2),
		asynq.Retention(30*24*time.Hour),
		asynq.TaskID(base64.RawURLEncoding.EncodeToString(id[:])),
		asynq.Queue(priority),
	)

	_, err := client.EnqueueContext(ctx, task)
	return err
}
