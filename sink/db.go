package sink

import (
	"context"
	"time"
)

// DBExecutor is an interface for executing SQL queries
// Replace this with your actual DB executor interface if needed
type DBExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (any, error)
}

// IntegrationDBExecutor wraps a DBExecutor to capture integration events
type IntegrationDBExecutor struct {
	Base DBExecutor
}

func (e *IntegrationDBExecutor) ExecContext(ctx context.Context, query string, args ...any) (any, error) {
	start := time.Now()
	result, err := e.Base.ExecContext(ctx, query, args...)
	latency := time.Since(start)

	integration := map[string]any{
		"type":    "sql",
		"query":   query,
		"latency": latency,
		"error":   err,
	}

	// Attach integration to trail in context if present
	trail := ctx.Value("gotrails_trail")
	if trail != nil {
		if t, ok := trail.(interface{ AddIntegration(any) }); ok {
			t.AddIntegration(integration)
		}
	}

	return result, err
}

// NewIntegrationDBExecutor wraps a DBExecutor
func NewIntegrationDBExecutor(base DBExecutor) DBExecutor {
	return &IntegrationDBExecutor{Base: base}
}
