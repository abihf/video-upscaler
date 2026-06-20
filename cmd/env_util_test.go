package cmd

import "testing"

func TestGetEnvReturnsValueWhenSet(t *testing.T) {
	t.Setenv("VIDEO_UPSCALER_TEST_ENV", "configured")

	got := getEnv("VIDEO_UPSCALER_TEST_ENV", "default")
	if got != "configured" {
		t.Fatalf("getEnv returned %q, want %q", got, "configured")
	}
}

func TestGetEnvReturnsDefaultWhenUnset(t *testing.T) {
	const envName = "VIDEO_UPSCALER_TEST_ENV_UNSET"

	got := getEnv(envName, "default")
	if got != "default" {
		t.Fatalf("getEnv returned %q, want %q", got, "default")
	}
}
