package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aizacoders/gotrails/gotrails"
	"github.com/gin-gonic/gin"
)

func TestGinMiddlewareCapturesRequestHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := gotrails.NewConfig()
	cfg.EnableMasking = true

	sink := &captureSink{}
	r := gin.New()
	r.Use(GinMiddlewareFunc(cfg, sink))
	r.POST("/v1/payments", func(c *gin.Context) {
		c.Header("X-Test", "ok")
		c.JSON(http.StatusCreated, gin.H{"token": "abc"})
	})

	body := bytes.NewBufferString(`{"password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/v1/payments", body)
	req.Header.Set("Authorization", "Bearer abc")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

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
	if trail.Response.Body != nil {
		data, _ := json.Marshal(trail.Response.Body)
		t.Fatalf("expected nil response body, got %s", data)
	}
}
