// ffmet package handle ffmpeg progress metrics
package ffmet

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metricsFn func(string)

var mapping = map[string]metricsFn{
	"fps": newGauge("fps", nil),
	"bitrate": newGauge("bitrate", func(s string) string {
		return strings.Replace(strings.TrimSpace(s), "kbits/s", "", 1)
	}),
	"speed": newGauge("speed", func(s string) string {
		return s[0 : len(s)-1]
	}),
	"quality": newGauge("quality", nil),
}

var Active = false
var fileWriter *os.File

func Handle(cmd *exec.Cmd) error {
	if !Active {
		return nil
	}

	if fileWriter == nil {
		r, w, err := os.Pipe()
		if err != nil {
			return err
		}
		fileWriter = w
		go readMetrics(r)
	}

	cmd.ExtraFiles = append(cmd.ExtraFiles, fileWriter)
	cmd.Args = append(cmd.Args, "-progress", fmt.Sprintf("pipe:%d", len(cmd.ExtraFiles)+2), "-stats_period", "10")
	return nil
}

func readMetrics(r *os.File) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		splitted := strings.SplitN(line, "=", 2)
		name := splitted[0]
		value := splitted[1]

		if strings.HasPrefix(name, "stream_") && strings.HasSuffix(name, "_q") {
			name = "quality"
		}

		fn := mapping[name]
		if fn != nil {
			fn(value)
		}
	}
}

func newGauge(name string, transform func(string) string) metricsFn {
	gauge := promauto.NewGauge(prometheus.GaugeOpts{Name: "ffmpeg_" + name})
	return func(s string) {
		if transform != nil {
			s = transform(s)
		}
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			gauge.Set(f)
		}
	}
}
