package env

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnv_LoadPrefix(t *testing.T) {
	t.Setenv("APP_SERVER_HTTP_PORT", "8080")
	t.Setenv("APP_FEATURE_BETA", "on")
	t.Setenv("UNRELATED_VAR", "x")

	s := New(WithPrefix("APP"))
	m, err := s.Load(context.Background())
	require.NoError(t, err)

	require.Equal(t, "8080", m["server"].(map[string]any)["http"].(map[string]any)["port"])
	require.Equal(t, "on", m["feature"].(map[string]any)["beta"])
	_, hasUnrelated := m["unrelated"]
	require.False(t, hasUnrelated)
}

func TestEnv_NoPrefix(t *testing.T) {
	t.Setenv("FOO_BAR", "v")
	s := New()
	m, err := s.Load(context.Background())
	require.NoError(t, err)
	require.Equal(t, "v", m["foo"].(map[string]any)["bar"])
}
