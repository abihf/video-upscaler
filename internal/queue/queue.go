package queue

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"time"

	"github.com/abihf/video-upscaler/internal/model"
	"github.com/hibiken/asynq"
)

var fileNameNormalizer = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func Add(ctx context.Context, conn asynq.RedisConnOpt, in, out, priority string, force bool) error {

	payload, _ := json.Marshal(model.VideoUpscaleTask{
		In:  in,
		Out: out,
	})

	name := fileNameNormalizer.ReplaceAllString(path.Base(out), "_")
	idBytes := sha1.Sum([]byte(out))
	id := name + "-" + base64.URLEncoding.EncodeToString(idBytes[0:6])
	task := asynq.NewTask(model.TaskVideoUpscaleType, payload,
		asynq.Timeout(3*time.Hour),
		asynq.MaxRetry(2),
		asynq.Retention(30*24*time.Hour),
		asynq.TaskID(id),
		asynq.Queue(priority),
	)

	client := asynq.NewClient(conn)
	_, err := client.EnqueueContext(ctx, task)
	if !force || err == nil || !errors.Is(err, asynq.ErrTaskIDConflict) {
		return err
	}

	ri := asynq.NewInspector(conn)
	err = ri.DeleteTask(priority, id)
	if err != nil {
		return fmt.Errorf("can not delete task %s: %w", id, err)
	}

	_, err = client.EnqueueContext(ctx, task)
	return err
}
