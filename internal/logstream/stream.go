package logstream

import (
	"io"
	"strings"
)

type LineTransformer struct {
	Callback func(line string) error

	sb strings.Builder
}

func New(callback func(line string) error) io.WriteCloser {
	t := &LineTransformer{
		Callback: callback,
	}
	return t
}

// Write implements io.WriteCloser.
func (t *LineTransformer) Write(data []byte) (n int, err error) {
	return t.WriteString(string(data))
}

func (t *LineTransformer) WriteString(data string) (int, error) {
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
		err := t.Callback(line)
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

// Close implements io.WriteCloser.
func (t *LineTransformer) Close() error {
	return t.Flush()
}

// Flush
func (t *LineTransformer) Flush() error {
	if t.sb.Len() > 0 {
		line := t.sb.String()
		t.sb.Reset()
		return t.Callback(line)
	}
	return nil
}
