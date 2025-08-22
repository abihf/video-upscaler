package activities

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Merge(ctx context.Context, orig, video, tmpDir string) (string, error) {
	outFile := filepath.Join(tmpDir, "merged.mkv")
	args := []string{
		"-o", outFile,
		"--clusters-in-meta-seek",
		"--engage", "no_cue_duration",
		"--engage", "no_cue_relative_position",
		"--fix-bitstream-timing-information", "0",
		"--default-duration", "0:24000/1001fps",
		video,
		"--no-video",
		orig,
	}
	mkvmerge := exec.CommandContext(ctx, "mkvmerge", args...)

	logFilePath := filepath.Join(tmpDir, "mkvmerge.log")
	logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return "", err
	}
	defer logFile.Close()
	defer fmt.Fprint(logFile, "\n------------------------------\n\n")
	mkvmerge.Stdout = logFile
	mkvmerge.Stderr = logFile
	if err := mkvmerge.Run(); err != nil {
		return "", err
	}
	return outFile, nil
}
