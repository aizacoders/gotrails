package masker

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMaskMapNested(t *testing.T) {
	m := New()
	input := map[string]any{
		"password": "secret",
		"nested": map[string]any{
			"token": "abc",
		},
		"list": []any{
			map[string]any{"cvv": "123"},
		},
	}

	out := m.MaskMap(input)
	if out["password"] != m.maskValue {
		t.Fatalf("expected password masked, got %v", out["password"])
	}
	nested := out["nested"].(map[string]any)
	if nested["token"] != m.maskValue {
		t.Fatalf("expected token masked, got %v", nested["token"])
	}
	list := out["list"].([]any)
	first := list[0].(map[string]any)
	if first["cvv"] != m.maskValue {
		t.Fatalf("expected cvv masked, got %v", first["cvv"])
	}
}

func TestParseAndMaskJSON(t *testing.T) {
	m := New()
	data := []byte(`{"password":"secret","nested":{"token":"abc"},"list":[{"cvv":"123"}]}`)
	v, err := m.ParseAndMaskJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b, _ := json.Marshal(v)
	out := string(b)
	if out == "" || out == string(data) {
		t.Fatal("expected masked JSON output")
	}
	if !strings.Contains(out, m.maskValue) {
		t.Fatalf("expected masked value in output, got %s", out)
	}
}
