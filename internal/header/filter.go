package header

import (
	"strings"
)

// Filter provides header filtering functionality
type Filter struct {
	excludeHeaders map[string]bool
	includeHeaders map[string]bool
	maskValue      string
}

// FilterOption is an option for Filter
type FilterOption func(*Filter)

// WithExcludeHeaders sets headers to exclude
func WithExcludeHeaders(headers []string) FilterOption {
	return func(f *Filter) {
		f.excludeHeaders = make(map[string]bool)
		for _, h := range headers {
			f.excludeHeaders[strings.ToLower(h)] = true
		}
	}
}

// WithIncludeHeaders sets headers to include (whitelist mode)
func WithIncludeHeaders(headers []string) FilterOption {
	return func(f *Filter) {
		f.includeHeaders = make(map[string]bool)
		for _, h := range headers {
			f.includeHeaders[strings.ToLower(h)] = true
		}
	}
}

// WithMaskValue sets the mask value for sensitive headers
func WithMaskValue(value string) FilterOption {
	return func(f *Filter) {
		f.maskValue = value
	}
}

// NewFilter creates a new header filter
func NewFilter(opts ...FilterOption) *Filter {
	f := &Filter{
		excludeHeaders: map[string]bool{
			"authorization": true,
			"cookie":        true,
			"set-cookie":    true,
			"x-api-key":     true,
		},
		includeHeaders: nil,
		maskValue:      "***MASKED***",
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// Filter filters and masks headers based on configuration
func (f *Filter) Filter(headers map[string][]string) map[string][]string {
	if headers == nil {
		return nil
	}

	result := make(map[string][]string)

	for key, values := range headers {
		lowerKey := strings.ToLower(key)

		// If whitelist mode is enabled, only include specified headers
		if f.includeHeaders != nil {
			if !f.includeHeaders[lowerKey] {
				continue
			}
		}

		// Check if header should be excluded
		if f.excludeHeaders[lowerKey] {
			// Mask instead of excluding completely
			result[key] = []string{f.maskValue}
			continue
		}

		// Copy the header values
		result[key] = make([]string, len(values))
		copy(result[key], values)
	}

	return result
}

// ShouldExclude checks if a header should be excluded
func (f *Filter) ShouldExclude(header string) bool {
	return f.excludeHeaders[strings.ToLower(header)]
}

// ShouldInclude checks if a header should be included
func (f *Filter) ShouldInclude(header string) bool {
	if f.includeHeaders == nil {
		return true
	}
	return f.includeHeaders[strings.ToLower(header)]
}

// AddExcludeHeader adds a header to the exclude list
func (f *Filter) AddExcludeHeader(header string) {
	f.excludeHeaders[strings.ToLower(header)] = true
}

// RemoveExcludeHeader removes a header from the exclude list
func (f *Filter) RemoveExcludeHeader(header string) {
	delete(f.excludeHeaders, strings.ToLower(header))
}
