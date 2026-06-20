package ffprog

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestProgressStringReflectsLatestParsedValues(t *testing.T) {
	progress := Start()
	t.Cleanup(func() {
		progress.Close()
	})

	if _, err := io.WriteString(progress.Writer, "fps=59.94\nout_time=00:00:03.000000\n"); err != nil {
		t.Fatalf("failed to write progress: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for {
		got := progress.String()
		if strings.Contains(got, "59.94") && strings.Contains(got, "00:00:03.000000") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("progress string did not update in time: %q", got)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
