package workflow

import (
	"time"

	"github.com/abihf/video-upscaler/internal/activity"
	"go.temporal.io/sdk/workflow"
)

func Upscale(ctx workflow.Context, inFile string, outFile string) (string, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Second * 10,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result string
	err := workflow.ExecuteActivity(ctx, activity.Upscale, inFile, outFile).Get(ctx, &result)
	if err != nil {
		return "", err
	}

	return result, nil

}
