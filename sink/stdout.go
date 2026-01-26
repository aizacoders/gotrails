package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/aizacoders/gotrails/gotrails"
)

// StdoutSink writes trails to stdout as JSON
type StdoutSink struct {
	mu       sync.Mutex
	writer   io.Writer
	pretty   bool
	disabled bool
	identify bool
}

// StdoutOption is an option for StdoutSink
type StdoutOption func(*StdoutSink)

// WithPrettyPrint enables pretty printing of JSON
func WithPrettyPrint(pretty bool) StdoutOption {
	return func(s *StdoutSink) {
		s.pretty = pretty
	}
}

// WithWriter sets a custom writer
func WithWriter(w io.Writer) StdoutOption {
	return func(s *StdoutSink) {
		s.writer = w
	}
}

// WithDisabled disables the sink
func WithDisabled(disabled bool) StdoutOption {
	return func(s *StdoutSink) {
		s.disabled = disabled
	}
}

// WithIdentifier enables a single-line identifier prefix for each trail
func WithIdentifier(enabled bool) StdoutOption {
	return func(s *StdoutSink) {
		s.identify = enabled
	}
}

// NewStdoutSink creates a new StdoutSink
func NewStdoutSink(opts ...StdoutOption) *StdoutSink {
	s := &StdoutSink{
		writer:   os.Stdout,
		pretty:   false,
		identify: true,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Write writes a trail to stdout as JSON
func (s *StdoutSink) Write(ctx context.Context, trail *gotrails.Trail) error {
	if s.disabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var data []byte
	var err error

	if s.pretty {
		data, err = json.MarshalIndent(trail, "", "  ")
	} else {
		data, err = json.Marshal(trail)
	}

	if err != nil {
		return err
	}

	if s.identify {
		method := ""
		path := ""
		if trail != nil && trail.Request != nil {
			method = trail.Request.Method
			path = trail.Request.Path
		}
		_, err = fmt.Fprintf(s.writer, "[GOTRAILS-debug] [trace_id=%s,request_id=%s,method=%s,path=%s,loggers=%s]\n", trail.TraceID, trail.RequestID, method, path, data)
		if err != nil {
			return err
		}
	}

	if s.identify {
		return err
	}

	// Add newline
	data = append(data, '\n')
	_, err = s.writer.Write(data)
	return err
}

// Close closes the stdout sink
func (s *StdoutSink) Close() error {
	return nil
}

// Name returns the name of the stdout sink
func (s *StdoutSink) Name() string {
	return "stdout"
}

// SetPretty sets the pretty print option
func (s *StdoutSink) SetPretty(pretty bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pretty = pretty
}

// SetDisabled sets the disabled option
func (s *StdoutSink) SetDisabled(disabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.disabled = disabled
}
