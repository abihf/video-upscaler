package activities

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.temporal.io/sdk/activity"
)

func Prepare(ctx context.Context, outFile string) (string, error) {
	info := activity.GetInfo(ctx)
	id := info.WorkflowExecution.ID
	tmpDir := filepath.Join("/media/data/upscaler", id)
	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	return tmpDir, nil
}
