package gotrails

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	oteltrace "go.opentelemetry.io/otel/trace"
)

// IntegrationType represents the type of integration
type IntegrationType string

const (
	IntegrationTypeHTTP     IntegrationType = "http"
	IntegrationTypeKafka    IntegrationType = "kafka"
	IntegrationTypeDatabase IntegrationType = "database"
	IntegrationTypeCache    IntegrationType = "cache"
	IntegrationTypeGRPC     IntegrationType = "grpc"
	IntegrationTypeCustom   IntegrationType = "custom"
)

// Trail represents a complete audit trail for a single request lifecycle
type Trail struct {
	mu sync.RWMutex `json:"-"`

	// Core identifiers
	Timestamp   time.Time `json:"timestamp"`
	TraceID     string    `json:"trace_id"`
	RequestID   string    `json:"request_id"`
	Service     string    `json:"service"`
	Environment string    `json:"environment"`

	// HTTP Request/Response
	Request  *HTTPRequest  `json:"request,omitempty"`
	Response *HTTPResponse `json:"response,omitempty"`

	// Performance
	LatencyMs int64     `json:"latency_ms"`
	startTime time.Time `json:"-"`

	// Trail components
	InternalSteps []InternalStep `json:"internal_steps,omitempty"`
	Integrations  []Integration  `json:"integrations,omitempty"`
	Errors        []TrailError   `json:"errors,omitempty"`

	// Free-form metadata
	Metadata map[string]any `json:"metadata,omitempty"`

	immutable bool    // set true after Finalize if config.Immutable
	cfg       *Config // keep config reference for immutability check

	// Hash chaining
	Hash     string `json:"hash,omitempty"`
	prevHash string // not exported, for chaining
}

// HTTPRequest represents the incoming HTTP request
type HTTPRequest struct {
	Method  string              `json:"method"`
	Path    string              `json:"path"`
	Query   string              `json:"query,omitempty"`
	Headers map[string][]string `json:"headers,omitempty"`
	Body    any                 `json:"body,omitempty"`
}

// HTTPResponse represents the outgoing HTTP response
type HTTPResponse struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers,omitempty"`
	Body    any                 `json:"body,omitempty"`
}

// InternalStep represents an internal processing step
type InternalStep struct {
	Name      string    `json:"name"`
	LatencyMs int64     `json:"latency_ms"`
	Request   any       `json:"request,omitempty"`
	Response  any       `json:"response,omitempty"`
	Error     string    `json:"error,omitempty"`
	StartTime time.Time `json:"-"`
}

// Integration represents an external integration call
type Integration struct {
	Type      IntegrationType `json:"type"`
	Name      string          `json:"name"`
	LatencyMs int64           `json:"latency_ms"`
	Request   any             `json:"request,omitempty"`
	Response  any             `json:"response,omitempty"`
	Error     string          `json:"error,omitempty"`
	Metadata  map[string]any  `json:"metadata,omitempty"`
}

// TrailError represents an error that occurred during the request
type TrailError struct {
	Source  string `json:"source"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// NewTrail creates a new Trail with the given trace ID
func NewTrail(traceID, requestID string, cfg *Config) *Trail {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Sampling logic: skip trail if random > sampling rate
	if cfg.SamplingRate < 1.0 {
		if rand.Float64() > cfg.SamplingRate {
			return nil
		}
	}

	now := time.Now().UTC()
	return &Trail{
		Timestamp:     now,
		TraceID:       traceID,
		RequestID:     requestID,
		Service:       cfg.ServiceName,
		Environment:   cfg.Environment,
		startTime:     now,
		InternalSteps: make([]InternalStep, 0),
		Integrations:  make([]Integration, 0),
		Errors:        make([]TrailError, 0),
		Metadata:      make(map[string]any),
		cfg:           cfg,
	}
}

// SetRequest sets the incoming HTTP request
func (t *Trail) SetRequest(req *HTTPRequest) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Request = req
}

// SetResponse sets the outgoing HTTP response
func (t *Trail) SetResponse(resp *HTTPResponse) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Response = resp
}

// AddInternalStep adds an internal processing step
func (t *Trail) AddInternalStep(step InternalStep) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.immutable {
		return
	}
	t.InternalSteps = append(t.InternalSteps, step)
}

// AddIntegration adds an external integration call
func (t *Trail) AddIntegration(integration Integration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.immutable {
		return
	}
	t.Integrations = append(t.Integrations, integration)
}

// AddError adds an error to the trail
func (t *Trail) AddError(source, message string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.immutable {
		return
	}
	t.Errors = append(t.Errors, TrailError{
		Source:  source,
		Message: message,
	})
}

// AddErrorWithCode adds an error with error code to the trail
func (t *Trail) AddErrorWithCode(source, message, code string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.immutable {
		return
	}
	t.Errors = append(t.Errors, TrailError{
		Source:  source,
		Message: message,
		Code:    code,
	})
}

// SetMetadata sets a metadata key-value pair
func (t *Trail) SetMetadata(key string, value any) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.immutable {
		return
	}
	if t.Metadata == nil {
		t.Metadata = make(map[string]any)
	}
	t.Metadata[key] = value
}

// SetPrevHash sets the previous hash for hash chaining
func (t *Trail) SetPrevHash(prev string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.prevHash = prev
}

// ComputeHash calculates the hash of the trail (excluding Hash field itself)
func (t *Trail) ComputeHash() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.computeHashLocked()
}

// Finalize calculates the total latency, prepares the trail for flushing, and sets the hash
func (t *Trail) Finalize() {
	t.mu.Lock()
	t.LatencyMs = time.Since(t.startTime).Milliseconds()
	if t.cfg != nil && t.cfg.Immutable {
		t.immutable = true
	}
	t.Hash = t.computeHashLocked()
	t.mu.Unlock()
}

// computeHashLocked calculates the hash of the trail assuming the lock is already held.
func (t *Trail) computeHashLocked() string {
	// Prepare a minimal struct for hashing (exclude Hash, prevHash, mu, cfg, immutable)
	tmp := struct {
		Timestamp     time.Time
		TraceID       string
		RequestID     string
		Service       string
		Environment   string
		Request       *HTTPRequest
		Response      *HTTPResponse
		LatencyMs     int64
		InternalSteps []InternalStep
		Integrations  []Integration
		Errors        []TrailError
		Metadata      map[string]any
		PrevHash      string
	}{
		Timestamp:     t.Timestamp,
		TraceID:       t.TraceID,
		RequestID:     t.RequestID,
		Service:       t.Service,
		Environment:   t.Environment,
		Request:       t.Request,
		Response:      t.Response,
		LatencyMs:     t.LatencyMs,
		InternalSteps: t.InternalSteps,
		Integrations:  t.Integrations,
		Errors:        t.Errors,
		Metadata:      t.Metadata,
		PrevHash:      t.prevHash,
	}
	b, _ := json.Marshal(tmp)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// Clone creates a deep copy of the trail for safe reading
func (t *Trail) Clone() *Trail {
	t.mu.RLock()
	defer t.mu.RUnlock()

	clone := &Trail{
		Timestamp:     t.Timestamp,
		TraceID:       t.TraceID,
		RequestID:     t.RequestID,
		Service:       t.Service,
		Environment:   t.Environment,
		Request:       t.Request,
		Response:      t.Response,
		LatencyMs:     t.LatencyMs,
		startTime:     t.startTime,
		InternalSteps: make([]InternalStep, len(t.InternalSteps)),
		Integrations:  make([]Integration, len(t.Integrations)),
		Errors:        make([]TrailError, len(t.Errors)),
		Metadata:      make(map[string]any),
	}

	copy(clone.InternalSteps, t.InternalSteps)
	copy(clone.Integrations, t.Integrations)
	copy(clone.Errors, t.Errors)

	for k, v := range t.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// StartStep creates a new InternalStep with the given name and start time
func StartStep(name string, req, resp any) InternalStep {
	return InternalStep{
		Name:      name,
		Request:   req,
		Response:  resp,
		StartTime: time.Now(),
	}
}

// EndStep finalizes an InternalStep, setting latency and optional error/response
func EndStep(step *InternalStep, resp any, err error) {
	step.LatencyMs = time.Since(step.StartTime).Milliseconds()
	if resp != nil {
		step.Response = resp
	}
	if err != nil {
		step.Error = err.Error()
	}
}

// TraceStep runs a function, captures latency, and adds the step to the trail in context
func TraceStep(ctx context.Context, name string, req any, fn func(context.Context) (resp any, err error)) (any, error) {
	step := StartStep(name, req, nil)
	resp, err := fn(ctx)
	EndStep(&step, resp, err)
	AddInternalStepToContext(ctx, step)
	return resp, err
}

// InjectOtelSpanToTrail links the current OpenTelemetry span to the trail (if present in context)
func InjectOtelSpanToTrail(ctx context.Context, trail *Trail) {
	if trail == nil {
		return
	}
	span := oteltrace.SpanFromContext(ctx)
	if span == nil || !span.SpanContext().IsValid() {
		return
	}
	trail.SetMetadata("otel_trace_id", span.SpanContext().TraceID().String())
	trail.SetMetadata("otel_span_id", span.SpanContext().SpanID().String())
	trail.SetMetadata("otel_span_sampled", span.SpanContext().IsSampled())
}
