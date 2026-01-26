package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/aizacoders/gotrails/gotrails"
)

type captureSink struct {
	mu     sync.Mutex
	trails []*gotrails.Trail
}

func (s *captureSink) Write(ctx context.Context, trail *gotrails.Trail) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if trail != nil {
		s.trails = append(s.trails, trail.Clone())
	}
	return nil
}

func (s *captureSink) Close() error { return nil }
func (s *captureSink) Name() string { return "capture" }

func (s *captureSink) last() *gotrails.Trail {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.trails) == 0 {
		return nil
	}
	return s.trails[len(s.trails)-1]
}

func TestHTTPMiddlewareCapturesRequestResponse(t *testing.T) {
	cfg := gotrails.NewConfig()
	cfg.EnableMasking = true

	sink := &captureSink{}
	mw := NewHTTPMiddleware(
		WithHTTPConfig(cfg),
		WithHTTPSink(sink),
	)

	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Test", "ok")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"token": "abc"})
	}))

	req := httptest.NewRequest(http.MethodPost, "http://example.com/v1/payments?x=1", bytes.NewBufferString(`{"password":"secret"}`))
	req.Header.Set("Authorization", "Bearer abc")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	trail := sink.last()
	if trail == nil {
		t.Fatal("expected trail in sink")
	}
	if trail.Request == nil || trail.Response == nil {
		t.Fatal("expected request and response to be set")
	}
	if got := trail.Request.Headers["Authorization"][0]; got != cfg.MaskValue {
		t.Fatalf("expected masked authorization header, got %s", got)
	}
	if got := trail.Response.Headers["X-Test"][0]; got != "ok" {
		t.Fatalf("expected response header X-Test, got %s", got)
	}

	respBody, ok := trail.Response.Body.(map[string]any)
	if !ok {
		t.Fatalf("expected response body map, got %T", trail.Response.Body)
	}
	if respBody["token"] != cfg.MaskValue {
		t.Fatalf("expected masked token, got %v", respBody["token"])
	}
}
