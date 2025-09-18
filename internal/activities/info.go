package activities

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"go.temporal.io/sdk/activity"
)

func Info(ctx context.Context, inFile, tmpDir string) (map[string]string, error) {
	cacheFile := getCacheFile(tmpDir)

	// Create vspipe command
	vspipe := exec.Command("vspipe", "-i",
		"-a", fmt.Sprintf("in=%s", inFile), "-a", fmt.Sprintf("cache=%s", cacheFile),
		"/upscale/script.py")

	stdout, err := vspipe.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	defer stdout.Close()

	result := make(map[string]string)
	go func() {
		br := bufio.NewScanner(stdout)
		for br.Scan() {
			line := br.Text()
			splitted := strings.SplitN(line, ":", 2)
			if len(splitted) == 2 {
				key := strings.TrimSpace(splitted[0])
				value := strings.TrimSpace(splitted[1])
				result[key] = value
			}
		}
	}()

	stderr, err := vspipe.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	defer stderr.Close()
	errBuff := &strings.Builder{}
	go func() {
		scanner := bufio.NewScanner(io.TeeReader(stderr, errBuff))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			activity.RecordHeartbeat(ctx, line)
		}
	}()

	err = vspipe.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run vspipe: %w\n%s", err, errBuff.String())
	}
	return result, nil
}
