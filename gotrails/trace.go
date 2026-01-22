package gotrails

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

// GenerateTraceID generates a new unique trace ID
func GenerateTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateRequestID generates a new unique request ID
func GenerateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ExtractTraceID extracts trace ID from HTTP headers or generates a new one
func ExtractTraceID(r *http.Request, cfg *Config) string {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Try to get from configured header
	traceID := r.Header.Get(cfg.TraceIDHeader)
	if traceID != "" {
		return traceID
	}

	// Try common trace ID headers
	commonHeaders := []string{
		"X-Trace-ID",
		"X-Request-ID",
		"X-Correlation-ID",
		"Traceparent",
	}

	for _, header := range commonHeaders {
		if strings.EqualFold(header, cfg.TraceIDHeader) {
			continue // Already checked
		}
		if val := r.Header.Get(header); val != "" {
			// For traceparent header (W3C format), extract the trace-id portion
			if strings.EqualFold(header, "Traceparent") {
				parts := strings.Split(val, "-")
				if len(parts) >= 2 {
					return parts[1]
				}
			}
			return val
		}
	}

	// Generate new trace ID
	return GenerateTraceID()
}

// ExtractRequestID extracts request ID from HTTP headers or generates a new one
func ExtractRequestID(r *http.Request, cfg *Config) string {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Try to get from configured header
	requestID := r.Header.Get(cfg.RequestIDHeader)
	if requestID != "" {
		return requestID
	}

	// Generate new request ID
	return GenerateRequestID()
}

// PropagateTraceHeaders adds trace headers to outgoing requests
func PropagateTraceHeaders(req *http.Request, trail *Trail, cfg *Config) {
	if trail == nil || cfg == nil {
		return
	}

	req.Header.Set(cfg.TraceIDHeader, trail.TraceID)
	req.Header.Set(cfg.RequestIDHeader, trail.RequestID)
}
