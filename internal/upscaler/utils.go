package upscaler

import (
	"fmt"
	"os"
	"os/exec"
)

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func runCmd(c *exec.Cmd, chErr chan error) {
	err := c.Run()
	if err != nil {
		chErr <- fmt.Errorf("%s error: %w", c.Path, err)
		return
	}
	chErr <- nil
}
