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
	"go.temporal.io/sdk/temporal"
)

func Scan(ctx context.Context, dirName string) error {
	c, err := conn.DialContext(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	return scanDir(ctx, dirName, c)
}

type episodeInfo struct {
	filePath string
	has1080p bool
	has2160p bool
}

var episodeRE = regexp.MustCompile(`S\d{2}E\d{2}`)

func scanDir(ctx context.Context, dirName string, c client.Client) error {
	listFiles, err := os.ReadDir(dirName)
	if err != nil {
		return fmt.Errorf("failed to read directory %q: %w", dirName, err)
	}
	subdirs := make([]string, 0)
	episodes := make(map[string]*episodeInfo)

	for _, entry := range listFiles {
		name := entry.Name()
		fullPath := filepath.Join(dirName, name)
		if entry.IsDir() {
			subdirs = append(subdirs, fullPath)
		} else {
			ext := filepath.Ext(name)
			if ext != ".mkv" {
				continue
			}
			matches := episodeRE.FindStringSubmatch(name)
			if len(matches) == 0 {
				continue
			}
			ep := matches[0]
			info, ok := episodes[ep]
			if !ok {
				info = &episodeInfo{}
				episodes[ep] = info
			}
			if strings.Contains(name, "2160p") {
				info.has2160p = true
			} else if strings.Contains(name, "1080p") {
				info.filePath = fullPath
				info.has1080p = true
			}

		}
	}

	for _, info := range episodes {
		if info.has1080p && !info.has2160p {
			outPath := strings.Replace(info.filePath, "1080p", "2160p", 1)
			fmt.Printf("Adding %s -> %s\n", filepath.Base(info.filePath), filepath.Base(outPath))
			options := client.StartWorkflowOptions{
				ID:        genId(outPath),
				TaskQueue: "upscaler",
				Priority:  temporal.Priority{PriorityKey: 4},

				WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_FAIL,
			}
			_, err := c.ExecuteWorkflow(ctx, options, "Upscale", info.filePath, outPath)
			if err != nil {
				fmt.Printf("Failed to add %s: %v\n", info.filePath, err)
			}
		}
	}

	for _, subdir := range subdirs {
		if err := scanDir(ctx, subdir, c); err != nil {
			return err
		}
	}
	return nil
}
