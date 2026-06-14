package codec

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"
)

type tomlCodec struct{}

func (tomlCodec) Name() string { return "toml" }

func (tomlCodec) Unmarshal(data []byte) (map[string]any, error) {
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	m := map[string]any{}
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("toml: %w", err)
	}
	return m, nil
}

func (tomlCodec) Marshal(v map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(v); err != nil {
		return nil, fmt.Errorf("toml: %w", err)
	}
	return buf.Bytes(), nil
}

func init() { Register(tomlCodec{}) }
