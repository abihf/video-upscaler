package task

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/abihf/video-upscaler/internal/conn"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

var fileNameCleaner = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func Add(ctx context.Context, inRelative string, outRelative string, priority string, force bool) error {
	in, err := filepath.Abs(inRelative)
	if err != nil {
		return err
	}
	if !fileExists(in) {
		return fmt.Errorf("input file %s not exist", in)
	}

	if outRelative == "" {
		outRelative = strings.Replace(inRelative, "1080p", "2160p", 1)
		if outRelative == inRelative {
			return fmt.Errorf("input file %s does not contain 1080p in its name", inRelative)
		}
	}

	out, err := filepath.Abs(outRelative)
	if err != nil {
		return err
	}
	if fileExists(out) {
		return fmt.Errorf("output file %s already exist", out)
	}

	c, err := conn.DialContext(ctx)
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}
	defer c.Close()

	outBase := filepath.Base(out)
	id := fileNameCleaner.ReplaceAllString(outBase, "_")
	policy := enums.WORKFLOW_ID_CONFLICT_POLICY_FAIL
	if force {
		policy = enums.WORKFLOW_ID_CONFLICT_POLICY_TERMINATE_EXISTING
	}
	options := client.StartWorkflowOptions{
		ID:        id,
		TaskQueue: "upscaler",

		WorkflowIDConflictPolicy: policy,
	}
	c.ExecuteWorkflow(ctx, options, "Upscale", in, out)
	return nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
