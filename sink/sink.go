package sink

import (
	"context"

	"github.com/aizacoders/gotrails/gotrails"
)

// Sink is the interface for trail output destinations
type Sink interface {
	// Write writes a trail to the sink
	Write(ctx context.Context, trail *gotrails.Trail) error

	// Close closes the sink and releases resources
	Close() error

	// Name returns the name of the sink
	Name() string
}

// MultiSink writes to multiple sinks
type MultiSink struct {
	sinks []Sink
}

// NewMultiSink creates a new MultiSink
func NewMultiSink(sinks ...Sink) *MultiSink {
	return &MultiSink{
		sinks: sinks,
	}
}

// Write writes to all sinks
func (m *MultiSink) Write(ctx context.Context, trail *gotrails.Trail) error {
	var lastErr error
	for _, s := range m.sinks {
		if err := s.Write(ctx, trail); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Close closes all sinks
func (m *MultiSink) Close() error {
	var lastErr error
	for _, s := range m.sinks {
		if err := s.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Name returns the name of the multi sink
func (m *MultiSink) Name() string {
	return "multi"
}

// AddSink adds a sink to the multi sink
func (m *MultiSink) AddSink(s Sink) {
	m.sinks = append(m.sinks, s)
}

// NoopSink is a sink that does nothing (useful for testing)
type NoopSink struct{}

// NewNoopSink creates a new NoopSink
func NewNoopSink() *NoopSink {
	return &NoopSink{}
}

// Write does nothing
func (n *NoopSink) Write(ctx context.Context, trail *gotrails.Trail) error {
	return nil
}

// Close does nothing
func (n *NoopSink) Close() error {
	return nil
}

// Name returns the name of the noop sink
func (n *NoopSink) Name() string {
	return "noop"
}
