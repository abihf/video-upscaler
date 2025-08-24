package task

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/abihf/video-upscaler/internal/conn"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

func Add(ctx context.Context, inRelative string, outRelative string, priorityStr string, force bool) error {
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

	policy := enums.WORKFLOW_ID_CONFLICT_POLICY_FAIL
	if force {
		policy = enums.WORKFLOW_ID_CONFLICT_POLICY_TERMINATE_EXISTING
	}
	priority := 3
	switch priorityStr {
	case PriorityLow:
		priority = 5
	case PriorityHigh:
		priority = 1
	}
	options := client.StartWorkflowOptions{
		ID:        genId(out),
		TaskQueue: "upscaler",
		Priority:  temporal.Priority{PriorityKey: priority},

		WorkflowIDConflictPolicy: policy,
	}
	c.ExecuteWorkflow(ctx, options, "Upscale", in, out)
	return nil
}
