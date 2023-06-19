// ffprog package handle ffmpeg progress metrics
package ffprog

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type metricsFn func(string)

var mapping = map[string]metricsFn{
	"fps": newFn("fps", nil),
	"bitrate": newFn("bitrate", func(s string) string {
		return strings.Replace(strings.TrimSpace(s), "kbits/s", "", 1)
	}),
}

var fileWriter *os.File

func Handle(cmd *exec.Cmd) error {
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

		fn := mapping[name]
		if fn != nil {
			fn(value)
		}
	}
}

func newFn(name string, transform func(string) string) metricsFn {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: "ffmpeg_" + name})
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

/*
dup_frames=0
drop_frames=0
speed=0.394x
progress=continue
frame=483
fps=9.55
stream_0_0_q=19.0
bitrate=6075.7kbits/s
total_size=15204352
out_time_us=20020000
out_time_ms=20020000
out_time=00:00:20.020000
dup_frames=0
drop_frames=0
speed=0.396x
progress=continue
frame=533
fps=9.58
stream_0_0_q=22.0
bitrate=5597.5kbits/s
total_size=15466496
out_time_us=22105000
out_time_ms=22105000
out_time=00:00:22.105000
dup_frames=0
drop_frames=0
speed=0.397x
progress=continue
*/
