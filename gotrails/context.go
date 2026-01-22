package gotrails

import (
	"context"
)

// contextKey is a private type for context keys
type contextKey string

const (
	trailContextKey  contextKey = "gotrails_trail"
	configContextKey contextKey = "gotrails_config"
)

// WithTrail adds a Trail to the context
func WithTrail(ctx context.Context, trail *Trail) context.Context {
	return context.WithValue(ctx, trailContextKey, trail)
}

// GetTrail retrieves the Trail from the context
func GetTrail(ctx context.Context) *Trail {
	if trail, ok := ctx.Value(trailContextKey).(*Trail); ok {
		return trail
	}
	return nil
}

// MustGetTrail retrieves the Trail from the context, panics if not found
func MustGetTrail(ctx context.Context) *Trail {
	trail := GetTrail(ctx)
	if trail == nil {
		panic("gotrails: trail not found in context")
	}
	return trail
}

// WithConfig adds a Config to the context
func WithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configContextKey, cfg)
}

// GetConfig retrieves the Config from the context
func GetConfig(ctx context.Context) *Config {
	if cfg, ok := ctx.Value(configContextKey).(*Config); ok {
		return cfg
	}
	return nil
}

// HasTrail checks if a Trail exists in the context
func HasTrail(ctx context.Context) bool {
	return GetTrail(ctx) != nil
}

// AddIntegrationToContext adds an integration to the trail in context
func AddIntegrationToContext(ctx context.Context, integration Integration) {
	if trail := GetTrail(ctx); trail != nil {
		trail.AddIntegration(integration)
	}
}

// AddErrorToContext adds an error to the trail in context
func AddErrorToContext(ctx context.Context, source, message string) {
	if trail := GetTrail(ctx); trail != nil {
		trail.AddError(source, message)
	}
}

// SetMetadataToContext sets metadata to the trail in context
func SetMetadataToContext(ctx context.Context, key string, value any) {
	if trail := GetTrail(ctx); trail != nil {
		trail.SetMetadata(key, value)
	}
}

// AddInternalStepToContext adds an internal step to the trail in context
func AddInternalStepToContext(ctx context.Context, step InternalStep) {
	if trail := GetTrail(ctx); trail != nil {
		trail.AddInternalStep(step)
	}
}
