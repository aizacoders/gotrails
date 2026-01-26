# gotrails

> **Global Audit Trail System for Go Services**

**ONE request = ONE trail log** — containing ALL internal steps and ALL integrations (HTTP, external API, Kafka, database, cache, gRPC, etc).

[![Go Reference](https://pkg.go.dev/badge/github.com/aizacoders/gotrails.svg)](https://pkg.go.dev/github.com/aizacoders/gotrails)
[![Go Report Card](https://goreportcard.com/badge/github.com/aizacoders/gotrails)](https://goreportcard.com/report/github.com/aizacoders/gotrails)

## Features

- ✅ **One Request → One Trail Log** — All processes collected, flushed once at the end
- ✅ **Context-based Trail Storage** — Thread-safe trail propagation
- ✅ **Automatic Request/Response Capture** — Headers, body, status
- ✅ **Field Masking** — Sensitive data protection (passwords, tokens, credit cards)
- ✅ **Body Size Limits** — Prevent memory issues with large payloads
- ✅ **Header Filtering** — Include/exclude specific headers
- ✅ **Async Processing** — Non-blocking trail writes
- ✅ **Multiple Sinks** — Stdout, File, and more
- ✅ **Framework Support** — Gin, net/http (native ServeMux)

## Installation

```bash
go get github.com/aizacoders/gotrails
```

## Quick Start

### With Gin

```go
package main

import (
    "github.com/aizacoders/gotrails/async"
    "github.com/aizacoders/gotrails/gotrails"
    "github.com/aizacoders/gotrails/middleware"
    "github.com/aizacoders/gotrails/sink"
    "github.com/gin-gonic/gin"
)

func main() {
    // Create configuration
    cfg := gotrails.NewConfig(
        gotrails.WithServiceName("my-service"),
        gotrails.WithEnvironment("production"),
    )

    // Create async stdout sink
    stdoutSink := sink.NewStdoutSink(sink.WithPrettyPrint(true))
    asyncSink := async.NewAsyncSink(stdoutSink, 1000)
    defer asyncSink.Close()

    // Setup Gin with gotrails middleware
    r := gin.New()
    r.Use(middleware.GinMiddlewareFunc(cfg, asyncSink))

    r.POST("/api/payments", func(c *gin.Context) {
        // Access trail from context
        ctx := c.Request.Context()
        if trail := gotrails.GetTrail(ctx); trail != nil {
            trail.SetMetadata("user_id", "u-123")
        }
        
        c.JSON(200, gin.H{"status": "ok"})
    })

    r.Run(":8080")
}
```

### With net/http (Native Go ServeMux)

```go
package main

import (
    "net/http"
    
    "github.com/aizacoders/gotrails/gotrails"
    "github.com/aizacoders/gotrails/middleware"
    "github.com/aizacoders/gotrails/sink"
)

func main() {
    cfg := gotrails.NewConfig(
        gotrails.WithServiceName("my-service"),
    )
    
    stdoutSink := sink.NewStdoutSink()
    
    m := middleware.NewHTTPMiddleware(
        middleware.WithHTTPConfig(cfg),
        middleware.WithHTTPSink(stdoutSink),
    )
    
    // Using Go 1.22+ ServeMux with enhanced routing
    mux := http.NewServeMux()
    
    mux.Handle("GET /health", m.HandlerFunc(healthHandler))
    mux.Handle("POST /v1/orders", m.HandlerFunc(createOrderHandler))
    mux.Handle("GET /v1/orders/{id}", m.HandlerFunc(getOrderHandler))
    
    http.ListenAndServe(":8080", mux)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    // Your handler logic
}

func createOrderHandler(w http.ResponseWriter, r *http.Request) {
    // Access trail from context
    if trail := gotrails.GetTrail(r.Context()); trail != nil {
        trail.SetMetadata("user_id", "u-123")
    }
    // Your handler logic
}

func getOrderHandler(w http.ResponseWriter, r *http.Request) {
    // Get path parameter using Go 1.22+ PathValue
    orderID := r.PathValue("id")
    // Your handler logic
}
```

### With http.Handler Middleware Pattern

```go
package main

import (
    "net/http"
    
    "github.com/aizacoders/gotrails/gotrails"
    "github.com/aizacoders/gotrails/middleware"
    "github.com/aizacoders/gotrails/sink"
)

func main() {
    cfg := gotrails.NewConfig(
        gotrails.WithServiceName("my-service"),
    )
    
    stdoutSink := sink.NewStdoutSink()
    
    // Using middleware pattern
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello"))
    })
    
    wrapped := middleware.HTTPMiddlewareFunc(cfg, stdoutSink)(handler)
    
    http.ListenAndServe(":8080", wrapped)
}
```

## Trail Output Example

```json
{
  "timestamp": "2026-01-23T10:30:45.123Z",
  "trace_id": "abc123def456",
  "request_id": "req-789",
  "service": "payment-service",
  "environment": "production",
  "request": {
    "method": "POST",
    "path": "/v1/payments",
    "headers": {
      "content-type": ["application/json"]
    },
    "body": {
      "amount": 150000,
      "payment_method": "bank_transfer",
      "password": "***MASKED***"
    }
  },
  "response": {
    "status": 201,
    "body": {
      "payment_id": "pay-123",
      "status": "PENDING"
    }
  },
  "latency_ms": 45,
  "internal_steps": [],
  "integrations": [],
  "errors": [],
  "metadata": {
    "user_id": "u-123"
  }
}
```

## Configuration Options

```go
cfg := gotrails.NewConfig(
    // Service identification
    gotrails.WithServiceName("my-service"),
    gotrails.WithEnvironment("production"),
    
    // Trace headers
    gotrails.WithTraceIDHeader("X-Trace-ID"),
    gotrails.WithRequestIDHeader("X-Request-ID"),
    
    // Body size limits
    gotrails.WithMaxRequestBodySize(64 * 1024),  // 64KB
    gotrails.WithMaxResponseBodySize(64 * 1024), // 64KB
    
    // Masking
    gotrails.WithMaskFields([]string{"password", "token", "secret"}),
    gotrails.WithMaskValue("***MASKED***"),
    gotrails.WithMaskingEnabled(true),
    
    // Header filtering
    gotrails.WithExcludeHeaders([]string{"authorization", "cookie"}),
    
    // Async processing
    gotrails.WithAsyncEnabled(true),
    gotrails.WithAsyncQueueSize(1000),
)
```

## Adding Trail Data

```go
// Get trail from context
trail := gotrails.GetTrail(ctx)

// Add metadata
trail.SetMetadata("user_id", "u-123")
trail.SetMetadata("order_id", "ord-456")

// Add error
trail.AddError("payment-gateway", "connection timeout")

// Add integration (for external calls)
trail.AddIntegration(gotrails.Integration{
    Type:      gotrails.IntegrationTypeHTTP,
    Name:      "stripe.charge",
    LatencyMs: 234,
    Request:   requestData,
    Response:  responseData,
})
```

## Sinks

### Stdout Sink
```go
sink := sink.NewStdoutSink(
    sink.WithPrettyPrint(true),
)
```

### Async Sink
```go
asyncSink := async.NewAsyncSink(baseSink, 1000,
    async.WithWorkers(4),
    async.WithDropOnFull(true),
    async.WithOnError(func(err error) {
        log.Printf("sink error: %v", err)
    }),
)
defer asyncSink.Close()
```

### Multi Sink
```go
multiSink := sink.NewMultiSink(
    stdoutSink,
    fileSink,
)
```

## Advanced Features

### Sampling
Control the percentage of requests that get a trail log:
```go
cfg := gotrails.NewConfig(
    gotrails.WithSamplingRate(0.1), // 10% of requests will be logged
)
```

### Immutable Trail
Prevent any further changes to a trail after it is finalized (audit-grade):
```go
cfg := gotrails.NewConfig(
    gotrails.WithImmutable(true),
)
// After trail.Finalize(), all mutating methods become no-ops.
```

### Hash Chaining
Each trail log includes a cryptographic hash of its contents and the previous log's hash:
```go
trail.SetPrevHash(prevHash) // Set previous hash before Finalize
trail.Finalize()
fmt.Println(trail.Hash) // SHA-256 hash for audit compliance
```

### OpenTelemetry Bridge
Correlate gotrails logs with OpenTelemetry traces:
```go
import oteltrace "go.opentelemetry.io/otel/trace"
// ...
trail := gotrails.GetTrail(ctx)
gotrails.InjectOtelSpanToTrail(ctx, trail)
// Trail metadata will include otel_trace_id, otel_span_id, otel_span_sampled
```

### Internal Steps API
Capture internal processing steps with latency:
```go
// Manual
step := gotrails.StartStep("ValidateRequest", req, nil)
// ... do work ...
gotrails.EndStep(&step, resp, err)
trail.AddInternalStep(step)

// Automatic
_, _ = gotrails.TraceStep(ctx, "CreatePayment", req, func(ctx context.Context) (any, error) {
    // ... do work ...
    return resp, err
})
```

---

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
