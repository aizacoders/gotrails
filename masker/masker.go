package masker

import (
	"strings"
)

// Masker provides field masking functionality
type Masker struct {
	fields    map[string]bool
	maskValue string
	enabled   bool
}

// Option is an option for Masker
type Option func(*Masker)

// WithFields sets the fields to mask
func WithFields(fields []string) Option {
	return func(m *Masker) {
		m.fields = make(map[string]bool)
		for _, f := range fields {
			m.fields[strings.ToLower(f)] = true
		}
	}
}

// WithMaskValue sets the mask replacement value
func WithMaskValue(value string) Option {
	return func(m *Masker) {
		m.maskValue = value
	}
}

// WithEnabled enables or disables masking
func WithEnabled(enabled bool) Option {
	return func(m *Masker) {
		m.enabled = enabled
	}
}

// New creates a new Masker
func New(opts ...Option) *Masker {
	m := &Masker{
		fields: map[string]bool{
			"password":      true,
			"token":         true,
			"secret":        true,
			"api_key":       true,
			"apikey":        true,
			"authorization": true,
			"credit_card":   true,
			"creditcard":    true,
			"cvv":           true,
			"pin":           true,
		},
		maskValue: "***MASKED***",
		enabled:   true,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// ShouldMask checks if a field should be masked
func (m *Masker) ShouldMask(field string) bool {
	if !m.enabled {
		return false
	}
	return m.fields[strings.ToLower(field)]
}

// Mask masks a value if the field should be masked
func (m *Masker) Mask(field string, value any) any {
	if m.ShouldMask(field) {
		return m.maskValue
	}
	return value
}

// MaskString masks a string value if the field should be masked
func (m *Masker) MaskString(field, value string) string {
	if m.ShouldMask(field) {
		return m.maskValue
	}
	return value
}

// MaskMap masks values in a map based on field names
func (m *Masker) MaskMap(data map[string]any) map[string]any {
	if !m.enabled || data == nil {
		return data
	}

	result := make(map[string]any, len(data))
	for k, v := range data {
		if m.ShouldMask(k) {
			result[k] = m.maskValue
		} else if nested, ok := v.(map[string]any); ok {
			result[k] = m.MaskMap(nested)
		} else if arr, ok := v.([]any); ok {
			result[k] = m.MaskSlice(arr)
		} else {
			result[k] = v
		}
	}
	return result
}

// MaskSlice masks values in a slice
func (m *Masker) MaskSlice(data []any) []any {
	if !m.enabled || data == nil {
		return data
	}

	result := make([]any, len(data))
	for i, v := range data {
		if nested, ok := v.(map[string]any); ok {
			result[i] = m.MaskMap(nested)
		} else if arr, ok := v.([]any); ok {
			result[i] = m.MaskSlice(arr)
		} else {
			result[i] = v
		}
	}
	return result
}

// MaskHeaders masks sensitive headers
func (m *Masker) MaskHeaders(headers map[string][]string) map[string][]string {
	if !m.enabled || headers == nil {
		return headers
	}

	result := make(map[string][]string, len(headers))
	for k, v := range headers {
		if m.ShouldMask(k) {
			result[k] = []string{m.maskValue}
		} else {
			result[k] = v
		}
	}
	return result
}

// AddField adds a field to be masked
func (m *Masker) AddField(field string) {
	m.fields[strings.ToLower(field)] = true
}

// RemoveField removes a field from masking
func (m *Masker) RemoveField(field string) {
	delete(m.fields, strings.ToLower(field))
}

// SetEnabled enables or disables masking
func (m *Masker) SetEnabled(enabled bool) {
	m.enabled = enabled
}

// GetMaskValue returns the mask value
func (m *Masker) GetMaskValue() string {
	return m.maskValue
}
