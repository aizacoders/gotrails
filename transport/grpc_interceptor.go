package transport

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// IntegrationUnaryClientInterceptor returns a gRPC UnaryClientInterceptor that captures integration events
func IntegrationUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		latency := time.Since(start)

		integration := map[string]any{
			"type":    "grpc",
			"method":  method,
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

		return err
	}
}
