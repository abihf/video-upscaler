package task

import (
	"os"
	"path/filepath"
	"regexp"
)

const (
	PriorityDefault = "default"
	PriorityLow     = "low"
	PriorityHigh    = "high"
)

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

var fileNameCleaner = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func genId(outFile string) string {
	outBase := filepath.Base(outFile)
	id := fileNameCleaner.ReplaceAllString(outBase, "_")
	return id
}
