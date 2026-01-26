package transport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/aizacoders/gotrails/gotrails"
	"github.com/aizacoders/gotrails/internal/body"
	"github.com/aizacoders/gotrails/internal/header"
	"github.com/aizacoders/gotrails/masker"
)

// HTTPRoundTripper wraps an http.RoundTripper to capture HTTP calls as integrations
type HTTPRoundTripper struct {
	Base http.RoundTripper
}

func (rt *HTTPRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		reqBody any
	)

	cfg := gotrails.GetConfig(req.Context())
	if cfg == nil {
		cfg = gotrails.DefaultConfig()
	}

	hf := header.NewFilter(
		header.WithExcludeHeaders(cfg.ExcludeHeaders),
		header.WithMaskValue(cfg.MaskValue),
	)
	if cfg.IncludeHeaders != nil {
		hf = header.NewFilter(
			header.WithIncludeHeaders(cfg.IncludeHeaders),
			header.WithExcludeHeaders(cfg.ExcludeHeaders),
			header.WithMaskValue(cfg.MaskValue),
		)
	}

	reqReader := body.NewReader(body.WithMaxSize(cfg.MaxRequestBodySize))
	respReader := body.NewReader(body.WithMaxSize(cfg.MaxResponseBodySize))
	msk := masker.New(
		masker.WithFields(cfg.MaskFields),
		masker.WithMaskValue(cfg.MaskValue),
		masker.WithEnabled(cfg.EnableMasking),
	)

	if req.Body != nil && req.ContentLength != 0 {
		if bodyBytes, newBody, err := reqReader.ReadAndRestore(req.Body); err == nil {
			req.Body = newBody
			reqBody = parseAndMaskJSON(msk, bodyBytes)
		}
	}

	start := time.Now()
	resp, err := rt.Base.RoundTrip(req)
	latencyMs := time.Since(start).Milliseconds()

	if trail := gotrails.GetTrail(req.Context()); trail != nil {
		integration := gotrails.Integration{
			Type:      gotrails.IntegrationTypeHTTP,
			Name:      req.Method + " " + req.URL.Host + req.URL.Path,
			LatencyMs: latencyMs,
			Request: map[string]any{
				"method": req.Method,
				"url":    req.URL.String(),
				"headers": func() map[string][]string {
					return hf.Filter(req.Header)
				}(),
				"body": reqBody,
			},
		}
		if resp != nil {
			var respBody any
			if resp.Body != nil {
				if bodyBytes, newBody, err := respReader.ReadAndRestore(resp.Body); err == nil {
					resp.Body = newBody
					respBody = parseAndMaskJSON(msk, bodyBytes)
				}
			}
			integration.Response = map[string]any{
				"status":  resp.StatusCode,
				"headers": hf.Filter(resp.Header),
				"body":    respBody,
			}
		}
		if err != nil {
			integration.Error = err.Error()
		}
		trail.AddIntegration(integration)
	}

	return resp, err
}

// NewHTTPRoundTripper returns a wrapped http.RoundTripper
func NewHTTPRoundTripper(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &HTTPRoundTripper{Base: base}
}

func parseAndMaskJSON(msk *masker.Masker, data []byte) any {
	if len(data) == 0 {
		return nil
	}
	if msk != nil {
		if v, err := msk.ParseAndMaskJSON(data); err == nil {
			return v
		}
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return string(bytes.TrimSpace(data))
	}
	return v
}
