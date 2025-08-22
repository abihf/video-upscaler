package activities

import "path/filepath"

func getCacheFile(tmpDir string) string {
	return filepath.Join(tmpDir, "cache")
}
