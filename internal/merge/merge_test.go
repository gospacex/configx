package merge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeepMerge_NestedOverride(t *testing.T) {
	a := map[string]any{
		"server": map[string]any{"host": "localhost", "port": 8080},
		"log":    "info",
	}
	b := map[string]any{
		"server": map[string]any{"port": 9090, "tls": true},
	}
	out := DeepMerge(a, b)
	require.Equal(t, "localhost", out["server"].(map[string]any)["host"])
	require.Equal(t, 9090, out["server"].(map[string]any)["port"])
	require.Equal(t, true, out["server"].(map[string]any)["tls"])
	require.Equal(t, "info", out["log"])
}

func TestDeepMerge_SlicesReplaced(t *testing.T) {
	a := map[string]any{"hosts": []any{"a", "b"}}
	b := map[string]any{"hosts": []any{"c"}}
	out := DeepMerge(a, b)
	require.Equal(t, []any{"c"}, out["hosts"])
}

func TestDeepMerge_DoesNotMutateInputs(t *testing.T) {
	a := map[string]any{"x": map[string]any{"y": 1}}
	b := map[string]any{"x": map[string]any{"z": 2}}
	_ = DeepMerge(a, b)
	require.Equal(t, map[string]any{"y": 1}, a["x"])
	require.Equal(t, map[string]any{"z": 2}, b["x"])
}

func TestDeepMergeAll_Order(t *testing.T) {
	out := DeepMergeAll(
		map[string]any{"k": 1},
		map[string]any{"k": 2},
		map[string]any{"k": 3},
	)
	require.Equal(t, 3, out["k"])
}
