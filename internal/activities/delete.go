package activities

import (
	"context"
	"os"
)

func Delete(ctx context.Context, file string) error {
	return os.Remove(file)
}
