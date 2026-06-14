package nacos

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuick_ParsesEndpoint(t *testing.T) {
	s, err := Quick("127.0.0.1:8848", "public", "DEFAULT_GROUP", "app.yaml")
	require.NoError(t, err)
	require.Equal(t, "nacos:public/DEFAULT_GROUP/app.yaml", s.ID())
	require.Equal(t, "127.0.0.1", s.cfg.Servers[0].IPAddr)
	require.EqualValues(t, 8848, s.cfg.Servers[0].Port)
}

func TestQuick_RejectsBadEndpoint(t *testing.T) {
	_, err := Quick("not-a-host-port", "", "g", "d")
	require.Error(t, err)
}

func TestMustQuick_PanicsOnBadInput(t *testing.T) {
	require.Panics(t, func() {
		_ = MustQuick("oops", "", "g", "d")
	})
}

func TestNew_ValidatesRequired(t *testing.T) {
	_, err := New(Config{})
	require.Error(t, err)
	_, err = New(Config{DataID: "x"})
	require.Error(t, err)
}
