package ffprog

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

type Progress struct {
	mu     sync.RWMutex
	fps    string
	time   string
	r      io.Closer
	Writer *os.File
}

func (p *Progress) String() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return fmt.Sprintf("Current FPS: %s, Time: %s", p.fps, p.time)
}

func (p *Progress) Close() {
	p.r.Close()
	p.Writer.Close()
}

func Start() *Progress {
	r, w, _ := os.Pipe()

	progress := &Progress{r: r, Writer: w}
	go func() {
		br := bufio.NewReader(r)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				slog.Error("failed to read ffmpeg progress", "error", err)
				break
			}

			if fpsStr, ok := strings.CutPrefix(line, "fps="); ok {
				progress.mu.Lock()
				progress.fps = strings.TrimSpace(fpsStr)
				progress.mu.Unlock()
			} else if timeStr, ok := strings.CutPrefix(line, "out_time="); ok {
				progress.mu.Lock()
				progress.time = strings.TrimSpace(timeStr)
				progress.mu.Unlock()
			}
		}
	}()
	return progress
}
