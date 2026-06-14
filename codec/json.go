package codec

import (
	"encoding/json"
	"fmt"
)

type jsonCodec struct{}

func (jsonCodec) Name() string { return "json" }

func (jsonCodec) Unmarshal(data []byte) (map[string]any, error) {
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

func (jsonCodec) Marshal(v map[string]any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func init() { Register(jsonCodec{}) }
