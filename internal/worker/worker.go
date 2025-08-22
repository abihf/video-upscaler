package worker

import (
	"context"
	"fmt"

	"github.com/abihf/video-upscaler/internal/activities"
	"github.com/abihf/video-upscaler/internal/conn"
	"github.com/abihf/video-upscaler/internal/workflows"
	"go.temporal.io/sdk/worker"
)

func Run(ctx context.Context) error {

	c, err := conn.DialContext(ctx)
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}
	defer c.Close()

	w := worker.New(c, "upscaler", worker.Options{
		MaxConcurrentActivityExecutionSize: 1,
		BackgroundActivityContext:          ctx,
	})

	w.RegisterWorkflow(workflows.Upscale4K)
	w.RegisterWorkflow(workflows.Upscale)
	w.RegisterActivity(activities.Prepare)
	w.RegisterActivity(activities.Info)
	w.RegisterActivity(activities.Upscale)
	w.RegisterActivity(activities.MoveFile)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		return fmt.Errorf("unable to start worker: %w", err)
	}

	return nil
}
