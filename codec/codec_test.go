package codec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestYAML_Roundtrip(t *testing.T) {
	c := MustGet("yaml")
	in := map[string]any{"a": 1, "b": map[string]any{"c": "d"}}
	data, err := c.Marshal(in)
	require.NoError(t, err)
	out, err := c.Unmarshal(data)
	require.NoError(t, err)
	require.EqualValues(t, 1, out["a"])
	require.Equal(t, "d", out["b"].(map[string]any)["c"])
}

func TestJSON_Roundtrip(t *testing.T) {
	c := MustGet("json")
	in := map[string]any{"a": 1.0}
	data, err := c.Marshal(in)
	require.NoError(t, err)
	out, err := c.Unmarshal(data)
	require.NoError(t, err)
	require.Equal(t, 1.0, out["a"])
}

func TestTOML_Roundtrip(t *testing.T) {
	c := MustGet("toml")
	in := map[string]any{"a": int64(1), "section": map[string]any{"k": "v"}}
	data, err := c.Marshal(in)
	require.NoError(t, err)
	out, err := c.Unmarshal(data)
	require.NoError(t, err)
	require.Equal(t, "v", out["section"].(map[string]any)["k"])
}

func TestGet_Aliases(t *testing.T) {
	c, err := Get(".yml")
	require.NoError(t, err)
	require.Equal(t, "yaml", c.Name())
}

func TestGet_Unknown(t *testing.T) {
	_, err := Get("xml")
	require.Error(t, err)
}
