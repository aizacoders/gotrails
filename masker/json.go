package masker

import (
	"encoding/json"
)

// MaskJSON masks sensitive fields in a JSON byte slice
func (m *Masker) MaskJSON(data []byte) ([]byte, error) {
	if !m.enabled || len(data) == 0 {
		return data, nil
	}

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return data, err
	}

	masked := m.maskAny(v)
	return json.Marshal(masked)
}

// MaskJSONString masks sensitive fields in a JSON string
func (m *Masker) MaskJSONString(data string) (string, error) {
	result, err := m.MaskJSON([]byte(data))
	if err != nil {
		return data, err
	}
	return string(result), nil
}

// maskAny recursively masks any value
func (m *Masker) maskAny(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return m.MaskMap(val)
	case []any:
		return m.MaskSlice(val)
	default:
		return v
	}
}

// ParseAndMaskJSON parses a JSON byte slice, masks it, and returns the result as any
func (m *Masker) ParseAndMaskJSON(data []byte) (any, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}

	if !m.enabled {
		return v, nil
	}

	return m.maskAny(v), nil
}
