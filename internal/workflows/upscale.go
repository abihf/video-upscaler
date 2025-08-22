package workflows

import (
	"fmt"
	"strings"
	"time"

	"github.com/abihf/video-upscaler/internal/activities"
	"go.temporal.io/sdk/workflow"
)

func Upscale4K(ctx workflow.Context, inFile string) error {
	outFile := strings.Replace(inFile, "1080p", "2160p", 1)
	if outFile == inFile {
		return fmt.Errorf("input file %s does not contain 1080p in its name", inFile)
	}
	return Upscale(ctx, inFile, outFile)
}

func Upscale(ctx workflow.Context, inFile string, outFile string) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Hour,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var tmpDir string
	err := workflow.ExecuteActivity(ctx, activities.Prepare, outFile).Get(ctx, &tmpDir)
	if err != nil {
		return err
	}

	err = workflow.ExecuteActivity(ctx, activities.Info, inFile, tmpDir).Get(ctx, nil)
	if err != nil {
		return err
	}

	var tmpOut string
	err = workflow.ExecuteActivity(ctx, activities.Upscale, inFile, tmpDir).Get(ctx, &tmpOut)
	if err != nil {
		return err
	}

	err = workflow.ExecuteActivity(ctx, activities.MoveFile, tmpOut, outFile).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil

}
