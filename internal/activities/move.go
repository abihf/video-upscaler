package activities

import (
	"context"
	"fmt"
	"os"
)

func MoveFile(ctx context.Context, inFile string, outFile string) error {
	err := os.Rename(inFile, outFile)
	if err != nil {
		return fmt.Errorf("failed to move file from %s to %s: %w", inFile, outFile, err)
	}

	return nil
}
