package cmd

import "os"

func getEnv(name string, def string) string {
	if val, ok := os.LookupEnv(name); ok {
		return val
	}
	return def
}
