# Gin Example

This example shows:

- Gin middleware for gotrails
- External HTTP integration captured via RoundTripper
- Post-response hook to store trails using the request context

## How It Works

1) Incoming request → gotrails middleware creates trail.
2) Handler runs (business logic).
3) External call uses `transport.NewHTTPRoundTripper`, which auto-appends integration.
4) gotrails finalizes trail and writes to sink.
5) `TrailStoreMiddleware()` runs after response to persist the trail.

## Run

```bash
go run examples/gin/main.go
```

## Endpoints

```bash
# health
curl -s http://localhost:8082/health

# create payment
curl -s -X POST http://localhost:8082/v1/payments \
  -H 'Content-Type: application/json' \
  -d '{"amount":150000,"payment_method":"bank_transfer","credit_card":"4111111111111111","cvv":"123"}'

# get payment
curl -s http://localhost:8082/v1/payments/123

# charge (external integration)
curl -s -X POST http://localhost:8082/v1/payments/123/charge
```

## Middleware Order

- gotrails middleware is registered first.
- `TrailStoreMiddleware()` runs after handler and after gotrails has finalized the trail.

## Integration Capture

- External call is made via `http.Client` with `transport.NewHTTPRoundTripper`.
- Request/response headers and body are captured (masked + size-limited).
- Integration is stored under `trail.Integrations`.

## Storing Trails

Edit `TrailStoreMiddleware()` in `examples/gin/main.go`:

```go
trail := gotrails.GetTrail(c.Request.Context())
// TODO: save trail to your DB or queue here
```

## Notes

- External provider mock runs on `http://localhost:8083/external/charge`.
- Integration capture is automatic via `transport.NewHTTPRoundTripper`.

## Troubleshooting

- If you see two logs per request, you are hitting the mock provider on the same server/port.
- If `trail` is nil, ensure gotrails middleware is registered before `TrailStoreMiddleware()`.
