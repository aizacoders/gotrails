package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/aizacoders/gotrails/gotrails"
	"github.com/aizacoders/gotrails/internal/body"
	"github.com/aizacoders/gotrails/internal/header"
	"github.com/aizacoders/gotrails/masker"
	"github.com/aizacoders/gotrails/sink"
	"github.com/gin-gonic/gin"
)

// GinMiddleware is the gotrails middleware for Gin
type GinMiddleware struct {
	cfg          *gotrails.Config
	sink         sink.Sink
	masker       *masker.Masker
	headerFilter *header.Filter
	bodyReader   *body.Reader
}

// GinOption is an option for GinMiddleware
type GinOption func(*GinMiddleware)

// WithGinConfig sets the config
func WithGinConfig(cfg *gotrails.Config) GinOption {
	return func(m *GinMiddleware) {
		m.cfg = cfg
	}
}

// WithGinSink sets the sink
func WithGinSink(s sink.Sink) GinOption {
	return func(m *GinMiddleware) {
		m.sink = s
	}
}

// WithGinMasker sets the masker
func WithGinMasker(msk *masker.Masker) GinOption {
	return func(m *GinMiddleware) {
		m.masker = msk
	}
}

// NewGinMiddleware creates a new Gin middleware
func NewGinMiddleware(opts ...GinOption) *GinMiddleware {
	m := &GinMiddleware{
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

// Handler returns the Gin handler function
func (m *GinMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract trace and request IDs
		traceID := gotrails.ExtractTraceID(c.Request, m.cfg)
		requestID := gotrails.ExtractRequestID(c.Request, m.cfg)

		// Create new trail
		trail := gotrails.NewTrail(traceID, requestID, m.cfg)

		// Read and restore request body
		var reqBody any
		if c.Request.Body != nil && c.Request.ContentLength > 0 {
			bodyBytes, newBody, err := m.bodyReader.ReadAndRestore(c.Request.Body)
			if err == nil {
				c.Request.Body = newBody
				// Parse and mask the body
				if m.cfg.EnableMasking {
					reqBody, _ = m.masker.ParseAndMaskJSON(bodyBytes)
				} else {
					reqBody, _ = parseJSON(bodyBytes)
				}
			}
		}

		// Set request info
		trail.SetRequest(&gotrails.HTTPRequest{
			Method:  c.Request.Method,
			Path:    c.Request.URL.Path,
			Query:   c.Request.URL.RawQuery,
			Headers: m.headerFilter.Filter(c.Request.Header),
			Body:    reqBody,
		})

		// Add trail to context
		ctx := gotrails.WithTrail(c.Request.Context(), trail)
		ctx = gotrails.WithConfig(ctx, m.cfg)
		c.Request = c.Request.WithContext(ctx)

		// Set trace headers in response
		c.Header(m.cfg.TraceIDHeader, traceID)
		c.Header(m.cfg.RequestIDHeader, requestID)

		// Create response writer wrapper to capture response
		rw := &ginResponseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
			maxSize:        m.cfg.MaxResponseBodySize,
		}
		c.Writer = rw

		// Process request
		c.Next()

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
			Status: rw.status,
			Body:   respBody,
		})

		// Finalize and flush trail
		trail.Finalize()
		_ = m.sink.Write(context.Background(), trail)
	}
}

// ginResponseWriter wraps gin.ResponseWriter to capture response body
type ginResponseWriter struct {
	gin.ResponseWriter
	body    *bytes.Buffer
	status  int
	maxSize int
}

func (w *ginResponseWriter) Write(data []byte) (int, error) {
	// Capture body up to maxSize
	if w.body.Len() < w.maxSize {
		remaining := w.maxSize - w.body.Len()
		if len(data) <= remaining {
			w.body.Write(data)
		} else {
			w.body.Write(data[:remaining])
		}
	}
	return w.ResponseWriter.Write(data)
}

func (w *ginResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *ginResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// parseJSON parses JSON bytes into any
func parseJSON(data []byte) (any, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		// If not valid JSON, return as string
		return string(data), nil
	}
	return v, nil
}

// GinMiddlewareFunc returns a simple middleware function for quick setup
func GinMiddlewareFunc(cfg *gotrails.Config, s sink.Sink) gin.HandlerFunc {
	m := NewGinMiddleware(
		WithGinConfig(cfg),
		WithGinSink(s),
	)
	return m.Handler()
}

// StandardHTTPMiddleware wraps net/http handler with gotrails
func StandardHTTPMiddleware(cfg *gotrails.Config, s sink.Sink) func(http.Handler) http.Handler {
	msk := masker.New(
		masker.WithFields(cfg.MaskFields),
		masker.WithMaskValue(cfg.MaskValue),
		masker.WithEnabled(cfg.EnableMasking),
	)

	hf := header.NewFilter(
		header.WithExcludeHeaders(cfg.ExcludeHeaders),
		header.WithMaskValue(cfg.MaskValue),
	)

	br := body.NewReader(
		body.WithMaxSize(cfg.MaxRequestBodySize),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace and request IDs
			traceID := gotrails.ExtractTraceID(r, cfg)
			requestID := gotrails.ExtractRequestID(r, cfg)

			// Create new trail
			trail := gotrails.NewTrail(traceID, requestID, cfg)

			// Read and restore request body
			var reqBody any
			if r.Body != nil && r.ContentLength > 0 {
				bodyBytes, newBody, err := br.ReadAndRestore(r.Body)
				if err == nil {
					r.Body = newBody
					if cfg.EnableMasking {
						reqBody, _ = msk.ParseAndMaskJSON(bodyBytes)
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
				Headers: hf.Filter(r.Header),
				Body:    reqBody,
			})

			// Add trail to context
			ctx := gotrails.WithTrail(r.Context(), trail)
			ctx = gotrails.WithConfig(ctx, cfg)
			r = r.WithContext(ctx)

			// Set trace headers in response
			w.Header().Set(cfg.TraceIDHeader, traceID)
			w.Header().Set(cfg.RequestIDHeader, requestID)

			// Create response writer wrapper
			rw := &responseWriter{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
				maxSize:        cfg.MaxResponseBodySize,
				status:         http.StatusOK,
			}

			// Process request
			next.ServeHTTP(rw, r)

			// Capture response
			var respBody any
			if rw.body.Len() > 0 {
				if cfg.EnableMasking {
					respBody, _ = msk.ParseAndMaskJSON(rw.body.Bytes())
				} else {
					respBody, _ = parseJSON(rw.body.Bytes())
				}
			}

			trail.SetResponse(&gotrails.HTTPResponse{
				Status: rw.status,
				Body:   respBody,
			})

			// Finalize and flush trail
			trail.Finalize()
			_ = s.Write(context.Background(), trail)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture response
type responseWriter struct {
	http.ResponseWriter
	body    *bytes.Buffer
	status  int
	maxSize int
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if w.body.Len() < w.maxSize {
		remaining := w.maxSize - w.body.Len()
		if len(data) <= remaining {
			w.body.Write(data)
		} else {
			w.body.Write(data[:remaining])
		}
	}
	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
