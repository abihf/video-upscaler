package cmd

import "os"

func getEnv(name string, def string) string {
	if val, ok := os.LookupEnv(name); ok {
		return val
	}
	return def
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
