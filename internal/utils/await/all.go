package await

func All[Arg any](fn func(Arg) error, args ...Arg) error {
	errChan := make(chan error, len(args))

	for _, arg := range args {
		go func(arg Arg) {
			errChan <- fn(arg)
		}(arg)
	}

	for range args {
		err := <-errChan
		if err != nil {
			return err
		}
	}
	return nil
}
