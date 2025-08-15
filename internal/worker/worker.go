package worker

import (
	"context"
	"fmt"
	"os"

	"github.com/abihf/video-upscaler/internal/activity"
	"github.com/abihf/video-upscaler/internal/conn"
	"github.com/abihf/video-upscaler/internal/workflow"
	"go.temporal.io/sdk/worker"
)

func Run(ctx context.Context) error {

	c, err := conn.DialContext(ctx)
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}
	defer c.Close()

	name, _ := os.Hostname()
	w := worker.New(c, "upscaler", worker.Options{
		MaxConcurrentActivityExecutionSize: 1,

		Identity: name,
	})

	w.RegisterWorkflow(workflow.Upscale)
	w.RegisterActivity(activity.Upscale)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		return fmt.Errorf("unable to start worker: %w", err)
	}

	return nil
}
