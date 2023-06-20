package logstream

import (
	"fmt"
	"io"
	"strings"
)

type Transformer struct {
	w      io.Writer
	prefix string

	sb strings.Builder
}

func New(w io.Writer, prefix string) io.WriteCloser {
	t := &Transformer{
		w:      w,
		prefix: prefix,
	}
	return t
}

// Write implements io.WriteCloser.
func (t *Transformer) Write(data []byte) (n int, err error) {
	return t.WriteString(string(data))
}

func (t *Transformer) WriteString(data string) (int, error) {
	start := 0
	for {
		nlIndex := strings.IndexAny(data[start:], "\r\n")
		if nlIndex < 0 {
			break
		}
		line := data[start : start+nlIndex]
		if t.sb.Len() > 0 {
			t.sb.WriteString(line)
			line = t.sb.String()
			t.sb.Reset()
		}
		err := t.writeLine(line)
		if err != nil {
			return 0, err
		}
		start += nlIndex + 1
	}
	if start >= len(data) {
		return len(data), nil
	}
	n, err := t.sb.WriteString(data[start:])
	if err != nil {
		return 0, err
	}
	return start + n, nil
}

func (t *Transformer) writeLine(line string) error {
	_, err := fmt.Fprintf(t.w, "%s %s\n", t.prefix, line)
	return err
}

// Close implements io.WriteCloser.
func (t *Transformer) Close() error {
	return t.Flush()
}

// Flush
func (t *Transformer) Flush() error {
	if t.sb.Len() > 0 {
		line := t.sb.String()
		t.sb.Reset()
		return t.writeLine(line)
	}
	return nil
}
