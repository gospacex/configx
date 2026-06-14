package flatten

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGet_Nested(t *testing.T) {
	root := map[string]any{
		"a": map[string]any{
			"b": map[string]any{"c": 42},
		},
		"list": []any{"x", "y", map[string]any{"k": "v"}},
	}
	v, ok := Get(root, "a.b.c")
	require.True(t, ok)
	require.Equal(t, 42, v)

	v, ok = Get(root, "list.2.k")
	require.True(t, ok)
	require.Equal(t, "v", v)

	_, ok = Get(root, "a.x.y")
	require.False(t, ok)

	_, ok = Get(root, "list.99")
	require.False(t, ok)
}

func TestSet_CreatesIntermediate(t *testing.T) {
	root := map[string]any{}
	Set(root, "a.b.c", 1)
	v, ok := Get(root, "a.b.c")
	require.True(t, ok)
	require.Equal(t, 1, v)
}
