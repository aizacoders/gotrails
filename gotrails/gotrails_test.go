package gotrails

import (
	"context"
	"errors"
	"math/rand"
	"testing"
)

func TestFinalizeSetsHashAndImmutability(t *testing.T) {
	cfg := NewConfig()
	cfg.Immutable = true
	trail := NewTrail("trace-1", "req-1", cfg)
	if trail == nil {
		t.Fatal("expected trail, got nil")
	}

	trail.SetPrevHash("prev-hash")
	trail.SetRequest(&HTTPRequest{Method: "POST", Path: "/v1/payments"})
	trail.SetResponse(&HTTPResponse{Status: 201})

	trail.Finalize()
	if trail.Hash == "" {
		t.Fatal("expected hash to be set")
	}
	if trail.Hash != trail.ComputeHash() {
		t.Fatalf("expected hash to match ComputeHash, got %s", trail.Hash)
	}

	trail.AddError("source", "error message")
	if len(trail.Errors) != 0 {
		t.Fatalf("expected no errors after immutable finalize, got %d", len(trail.Errors))
	}
}

func TestTraceStepAddsInternalStep(t *testing.T) {
	cfg := NewConfig()
	trail := NewTrail("trace-2", "req-2", cfg)
	if trail == nil {
		t.Fatal("expected trail, got nil")
	}

	ctx := WithTrail(context.Background(), trail)
	resp, err := TraceStep(ctx, "CreatePayment", "req", func(ctx context.Context) (any, error) {
		return "resp", errors.New("boom")
	})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected error boom, got %v", err)
	}
	if resp != "resp" {
		t.Fatalf("expected resp, got %v", resp)
	}

	if len(trail.InternalSteps) != 1 {
		t.Fatalf("expected 1 internal step, got %d", len(trail.InternalSteps))
	}
	step := trail.InternalSteps[0]
	if step.Name != "CreatePayment" {
		t.Fatalf("unexpected step name: %s", step.Name)
	}
	if step.Error != "boom" {
		t.Fatalf("unexpected step error: %s", step.Error)
	}
	if step.LatencyMs < 0 {
		t.Fatalf("unexpected negative latency: %d", step.LatencyMs)
	}
}

func TestSamplingRateDeterministic(t *testing.T) {
	rand.Seed(1)
	val := rand.Float64()
	rand.Seed(1)

	cfg := NewConfig(WithSamplingRate(0.5))
	trail := NewTrail("trace-3", "req-3", cfg)
	if val > cfg.SamplingRate && trail != nil {
		t.Fatal("expected nil trail due to sampling")
	}
	if val <= cfg.SamplingRate && trail == nil {
		t.Fatal("expected trail due to sampling")
	}
}
