package await

import (
	"golang.org/x/sync/errgroup"
)

// All executes fn concurrently for each arg, returning the first non-nil error encountered.
// It waits for all goroutines to complete before returning.
func All[Arg any](fn func(Arg) error, args ...Arg) error {
	var eg errgroup.Group

	for _, arg := range args {
		arg := arg // capture loop variable
		eg.Go(func() error {
			return fn(arg)
		})
	}

	return eg.Wait()
}
