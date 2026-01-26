package middleware

import (
	"bytes"
	"context"
	"net/http"

	"github.com/aizacoders/gotrails/gotrails"
	"github.com/aizacoders/gotrails/internal/body"
	"github.com/aizacoders/gotrails/internal/header"
	"github.com/aizacoders/gotrails/masker"
	"github.com/aizacoders/gotrails/sink"
)

// HTTPMiddleware is the gotrails middleware for net/http (native Go mux)
type HTTPMiddleware struct {
	cfg          *gotrails.Config
	sink         sink.Sink
	masker       *masker.Masker
	headerFilter *header.Filter
	bodyReader   *body.Reader
}

// HTTPOption is an option for HTTPMiddleware
type HTTPOption func(*HTTPMiddleware)

// WithHTTPConfig sets the config
func WithHTTPConfig(cfg *gotrails.Config) HTTPOption {
	return func(m *HTTPMiddleware) {
		m.cfg = cfg
	}
}

// WithHTTPSink sets the sink
func WithHTTPSink(s sink.Sink) HTTPOption {
	return func(m *HTTPMiddleware) {
		m.sink = s
	}
}

// WithHTTPMasker sets the masker
func WithHTTPMasker(msk *masker.Masker) HTTPOption {
	return func(m *HTTPMiddleware) {
		m.masker = msk
	}
}

// NewHTTPMiddleware creates a new net/http middleware
func NewHTTPMiddleware(opts ...HTTPOption) *HTTPMiddleware {
	m := &HTTPMiddleware{
		cfg:    gotrails.DefaultConfig(),
		sink:   sink.NewStdoutSink(),
		masker: masker.New(),
	}

	for _, opt := range opts {
		opt(m)
	}

	// Initialize header filter with config
	m.headerFilter = header.NewFilter(
		header.WithExcludeHeaders(m.cfg.ExcludeHeaders),
		header.WithMaskValue(m.cfg.MaskValue),
	)
	if m.cfg.IncludeHeaders != nil {
		m.headerFilter = header.NewFilter(
			header.WithIncludeHeaders(m.cfg.IncludeHeaders),
			header.WithExcludeHeaders(m.cfg.ExcludeHeaders),
			header.WithMaskValue(m.cfg.MaskValue),
		)
	}

	// Initialize body reader with config
	m.bodyReader = body.NewReader(
		body.WithMaxSize(m.cfg.MaxRequestBodySize),
	)

	return m
}

// Handler wraps an http.Handler with gotrails
func (m *HTTPMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace and request IDs
		traceID := gotrails.ExtractTraceID(r, m.cfg)
		requestID := gotrails.ExtractRequestID(r, m.cfg)

		// Create new trail
		trail := gotrails.NewTrail(traceID, requestID, m.cfg)

		// Read and restore request body
		var reqBody any
		if r.Body != nil && r.ContentLength > 0 {
			bodyBytes, newBody, err := m.bodyReader.ReadAndRestore(r.Body)
			if err == nil {
				r.Body = newBody
				if m.cfg.EnableMasking {
					reqBody, _ = m.masker.ParseAndMaskJSON(bodyBytes)
				} else {
					reqBody, _ = parseJSON(bodyBytes)
				}
			}
		}

		// Set request info
		trail.SetRequest(&gotrails.HTTPRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Query:   r.URL.RawQuery,
			Headers: m.headerFilter.Filter(r.Header),
			Body:    reqBody,
		})

		// Add trail to context
		ctx := gotrails.WithTrail(r.Context(), trail)
		ctx = gotrails.WithConfig(ctx, m.cfg)
		r = r.WithContext(ctx)

		// Set trace headers in response
		w.Header().Set(m.cfg.TraceIDHeader, traceID)
		w.Header().Set(m.cfg.RequestIDHeader, requestID)

		// Create response writer wrapper
		rw := &responseWriter{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
			maxSize:        m.cfg.MaxResponseBodySize,
			status:         http.StatusOK,
		}

		// Process request
		next.ServeHTTP(rw, r)

		// Capture response
		var respBody any
		if rw.body.Len() > 0 {
			if m.cfg.EnableMasking {
				respBody, _ = m.masker.ParseAndMaskJSON(rw.body.Bytes())
			} else {
				respBody, _ = parseJSON(rw.body.Bytes())
			}
		}

		trail.SetResponse(&gotrails.HTTPResponse{
			Status:  rw.status,
			Headers: m.headerFilter.Filter(rw.Header()),
			Body:    respBody,
		})

		// Finalize and flush trail
		trail.Finalize()
		_ = m.sink.Write(context.Background(), trail)
	})
}

// HandlerFunc wraps an http.HandlerFunc with gotrails
func (m *HTTPMiddleware) HandlerFunc(next http.HandlerFunc) http.Handler {
	return m.Handler(next)
}

// Middleware returns a middleware function compatible with common middleware patterns
func (m *HTTPMiddleware) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return m.Handler(next)
	}
}

// HTTPMiddlewareFunc returns a simple middleware function for quick setup
func HTTPMiddlewareFunc(cfg *gotrails.Config, s sink.Sink) func(http.Handler) http.Handler {
	m := NewHTTPMiddleware(
		WithHTTPConfig(cfg),
		WithHTTPSink(s),
	)
	return m.Middleware()
}
