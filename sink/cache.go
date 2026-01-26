package sink

import (
	"context"
	"time"
)

// CacheClient is an interface for Redis/Cache operations
// Replace this with your actual cache client interface if needed
type CacheClient interface {
	Do(ctx context.Context, cmd string, args ...any) (any, error)
}

// IntegrationCacheClient wraps a CacheClient to capture integration events
type IntegrationCacheClient struct {
	Base CacheClient
}

func (c *IntegrationCacheClient) Do(ctx context.Context, cmd string, args ...any) (any, error) {
	start := time.Now()
	result, err := c.Base.Do(ctx, cmd, args...)
	latency := time.Since(start)

	integration := map[string]any{
		"type":    "redis",
		"command": cmd,
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

// NewIntegrationCacheClient wraps a CacheClient
func NewIntegrationCacheClient(base CacheClient) CacheClient {
	return &IntegrationCacheClient{Base: base}
}
