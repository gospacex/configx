package codec

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type yamlCodec struct{}

func (yamlCodec) Name() string { return "yaml" }

func (yamlCodec) Unmarshal(data []byte) (map[string]any, error) {
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("yaml: %w", err)
	}
	m, ok := normalize(raw).(map[string]any)
	if !ok {
		// Allow empty documents / non-map roots (treat as empty config).
		if raw == nil {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("yaml: root must be a mapping, got %T", raw)
	}
	return m, nil
}

func (yamlCodec) Marshal(v map[string]any) ([]byte, error) {
	return yaml.Marshal(v)
}

// normalize converts map[interface{}]interface{} (yaml.v2 style) and nested
// structures into map[string]any so the rest of the SDK can rely on that
// invariant. yaml.v3 already returns map[string]any, but we run this
// defensively in case external callers ever pre-decode.
func normalize(in any) any {
	switch v := in.(type) {
	case map[any]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			out[fmt.Sprint(k)] = normalize(val)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			out[k] = normalize(val)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = normalize(val)
		}
		return out
	default:
		return v
	}
}

func init() { Register(yamlCodec{}) }
