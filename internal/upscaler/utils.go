package upscaler

import (
	"fmt"
	"os"

	"github.com/google/shlex"
)

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func awaitAll[Arg any](fn func(Arg) error, args ...Arg) error {
	errChan := make(chan error, len(args))

	for _, arg := range args {
		go func(arg Arg) {
			errChan <- fn(arg)
		}(arg)
	}

	for i := 0; i < len(args); i++ {
		err := <-errChan
		if err != nil {
			return err
		}
	}
	return nil
}

func parseArgsFromEnv(name string, def ...string) []string {
	ffArgsEnv := os.Getenv(name)
	if ffArgsEnv != "" {
		ffTranscodeArgs, err := shlex.Split(ffArgsEnv)
		if err != nil {
			panic(fmt.Errorf("can't parse args %s: %w", name, err))
		}
		return ffTranscodeArgs
	}
	return def
}
