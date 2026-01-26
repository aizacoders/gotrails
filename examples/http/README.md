# net/http Example

This example shows:

- net/http middleware for gotrails
- External HTTP integration captured via RoundTripper
- Post-response hook to store trails using the request context

## How It Works

1) Incoming request → gotrails middleware creates trail.
2) Handler runs (business logic).
3) External call uses `transport.NewHTTPRoundTripper`, which auto-appends integration.
4) gotrails finalizes trail and writes to sink.
5) `WithHTTPAfterFlush` callback runs to persist the trail.

## Run

```bash
go run examples/http/main.go
```

## Endpoints

```bash
# health
curl -s http://localhost:8081/health

# create order
curl -s -X POST http://localhost:8081/v1/orders \
  -H 'Content-Type: application/json' \
  -d '{"items":[{"product_id":"p1","quantity":2,"price":10000}]}'

# get order
curl -s http://localhost:8081/v1/orders/ord-123

# charge order (external integration)
curl -s -X POST http://localhost:8081/v1/orders/ord-123/charge
```

## Middleware Order

- gotrails middleware wraps the main mux.
- `WithHTTPAfterFlush` runs after finalize + sink write.

## Integration Capture

- External call is made via `http.Client` with `transport.NewHTTPRoundTripper`.
- Request/response headers and body are captured (masked + size-limited).
- Integration is stored under `trail.Integrations`.

## Storing Trails

Edit the callback in `buildHandler()` in `examples/http/main.go`:

```go
middleware.WithHTTPAfterFlush(func(ctx context.Context, trail *gotrails.Trail) {
    // TODO: save trail to your DB or queue here
})
```

## Notes

- External provider mock runs on `http://localhost:8083/external/charge`.
- Integration capture is automatic via `transport.NewHTTPRoundTripper`.

## Troubleshooting

- If `trail` is nil, ensure the handler is wrapped by gotrails middleware.
