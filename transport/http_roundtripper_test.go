package transport

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aizacoders/gotrails/gotrails"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHTTPRoundTripperCapturesIntegration(t *testing.T) {
	cfg := gotrails.NewConfig()
	cfg.EnableMasking = true

	trail := gotrails.NewTrail("trace-1", "req-1", cfg)
	if trail == nil {
		t.Fatal("expected trail, got nil")
	}

	base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		respBody := io.NopCloser(bytes.NewBufferString(`{"password":"secret"}`))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Resp": []string{"ok"}},
			Body:       respBody,
		}, nil
	})

	rt := NewHTTPRoundTripper(base)

	req := httptest.NewRequest(http.MethodPost, "http://example.com/external/charge", bytes.NewBufferString(`{"token":"abc"}`))
	req.Header.Set("Authorization", "Bearer abc")

	ctx := gotrails.WithTrail(context.Background(), trail)
	ctx = gotrails.WithConfig(ctx, cfg)
	req = req.WithContext(ctx)

	_, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(trail.Integrations) != 1 {
		t.Fatalf("expected 1 integration, got %d", len(trail.Integrations))
	}
	integration := trail.Integrations[0]
	reqMap, ok := integration.Request.(map[string]any)
	if !ok {
		t.Fatalf("expected request map, got %T", integration.Request)
	}
	reqBody := reqMap["body"].(map[string]any)
	if reqBody["token"] != cfg.MaskValue {
		t.Fatalf("expected masked token, got %v", reqBody["token"])
	}

	respMap := integration.Response.(map[string]any)
	respBody := respMap["body"].(map[string]any)
	if respBody["password"] != cfg.MaskValue {
		t.Fatalf("expected masked password, got %v", respBody["password"])
	}

	respHeaders := respMap["headers"].(map[string][]string)
	if got := respHeaders["X-Resp"][0]; got != "ok" {
		t.Fatalf("expected response header X-Resp, got %s", got)
	}
}
