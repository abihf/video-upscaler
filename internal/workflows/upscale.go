package workflows

import (
	"github.com/abihf/video-upscaler/internal/activities"
	"go.temporal.io/sdk/workflow"
)

func Upscale(ctx workflow.Context, inFile string, outFile string) error {

	var tmpDir string
	err := workflow.ExecuteActivity(ctx, activities.Prepare, outFile).Get(ctx, &tmpDir)
	if err != nil {
		return err
	}

	var tmpOut string
	err = workflow.ExecuteActivity(ctx, activities.Upscale, inFile, outFile, tmpDir).Get(ctx, &tmpOut)
	if err != nil {
		return err
	}

	err = workflow.ExecuteActivity(ctx, activities.MoveFile, tmpOut, outFile).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil

}
