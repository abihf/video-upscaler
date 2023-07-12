package logstream

import (
	"bytes"
	"io"
	"strings"
)

type LogStream struct {
	Callback func(line string) error

	sb strings.Builder
}

var _ io.WriteCloser = &LogStream{}

func New(callback func(line string) error) *LogStream {
	s := &LogStream{
		Callback: callback,
	}
	return s
}

// Write implements io.WriteCloser.
func (s *LogStream) Write(data []byte) (int, error) {
	start := 0
	for start < len(data) {
		nlIndex := bytes.IndexAny(data[start:], "\r\n")
		if nlIndex < 0 {
			break
		}
		line := data[start : start+nlIndex]
		var lineStr string
		if s.sb.Len() > 0 {
			s.sb.Write(line)
			lineStr = s.sb.String()
			s.sb.Reset()
		} else {
			lineStr = string(line)
		}
		err := s.Callback(lineStr)
		if err != nil {
			return 0, err
		}
		start += nlIndex + 1
	}
	if start >= len(data) {
		return len(data), nil
	}
	n, err := s.sb.Write(data[start:])
	if err != nil {
		return 0, err
	}
	return start + n, nil
}

func (s *LogStream) WriteString(data string) (int, error) {
	start := 0
	for start < len(data) {
		nlIndex := strings.IndexAny(data[start:], "\r\n")
		if nlIndex < 0 {
			break
		}
		line := data[start : start+nlIndex]
		if s.sb.Len() > 0 {
			s.sb.WriteString(line)
			line = s.sb.String()
			s.sb.Reset()
		}
		err := s.Callback(line)
		if err != nil {
			return 0, err
		}
		start += nlIndex + 1
	}
	if start >= len(data) {
		return len(data), nil
	}
	n, err := s.sb.WriteString(data[start:])
	if err != nil {
		return 0, err
	}
	return start + n, nil
}

// Close implements io.WriteCloser.
func (s *LogStream) Close() error {
	return s.Flush()
}

// Flush
func (s *LogStream) Flush() error {
	if s.sb.Len() > 0 {
		line := s.sb.String()
		s.sb.Reset()
		return s.Callback(line)
	}
	return nil
}
