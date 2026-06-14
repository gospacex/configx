package configx_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gospacex/configx"
	"github.com/gospacex/configx/source/file"
)

func newCfg(t *testing.T, body string) *configx.Config {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yaml")
	require.NoError(t, os.WriteFile(p, []byte(body), 0o644))
	c, err := configx.New(configx.WithSource(file.New(p, file.WithWatch(false))))
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestString_AllPaths(t *testing.T) {
	c := newCfg(t, "a: hello\nb: 12\nc: true\nd: 1.5\n")
	v, err := c.String("a")
	require.NoError(t, err)
	require.Equal(t, "hello", v)

	v, err = c.String("b")
	require.NoError(t, err)
	require.Equal(t, "12", v)

	_, err = c.String("missing")
	require.ErrorIs(t, err, configx.ErrKeyNotFound)
}

func TestInt_AllPaths(t *testing.T) {
	c := newCfg(t, "a: 7\nb: \"42\"\nc: 1.9\nd: bad\n")

	v, err := c.Int("a")
	require.NoError(t, err)
	require.EqualValues(t, 7, v)

	v, err = c.Int("b")
	require.NoError(t, err)
	require.EqualValues(t, 42, v)

	v, err = c.Int("c")
	require.NoError(t, err)
	require.EqualValues(t, 1, v) // truncated

	_, err = c.Int("d")
	require.ErrorIs(t, err, configx.ErrTypeMismatch)

	_, err = c.Int("missing")
	require.ErrorIs(t, err, configx.ErrKeyNotFound)

	require.EqualValues(t, 99, c.IntOr("missing", 99))
}

func TestFloat_AllPaths(t *testing.T) {
	c := newCfg(t, "a: 1.5\nb: \"3.14\"\nc: 5\nd: nope\n")
	for _, k := range []string{"a", "b", "c"} {
		_, err := c.Float(k)
		require.NoError(t, err, k)
	}
	_, err := c.Float("d")
	require.ErrorIs(t, err, configx.ErrTypeMismatch)
	require.Equal(t, 9.9, c.FloatOr("missing", 9.9))
}

func TestBool_AllPaths(t *testing.T) {
	c := newCfg(t, "a: true\nb: \"false\"\nc: 1\nd: bogus\n")
	v, err := c.Bool("a")
	require.NoError(t, err)
	require.True(t, v)

	v, err = c.Bool("b")
	require.NoError(t, err)
	require.False(t, v)

	v, err = c.Bool("c")
	require.NoError(t, err)
	require.True(t, v)

	_, err = c.Bool("d")
	require.ErrorIs(t, err, configx.ErrTypeMismatch)
	require.True(t, c.BoolOr("missing", true))
}

func TestDuration_AllPaths(t *testing.T) {
	c := newCfg(t, "a: \"1500ms\"\nb: 1000000000\nc: nope\n")

	v, err := c.Duration("a")
	require.NoError(t, err)
	require.Equal(t, 1500*time.Millisecond, v)

	v, err = c.Duration("b")
	require.NoError(t, err)
	require.Equal(t, time.Second, v)

	_, err = c.Duration("c")
	require.ErrorIs(t, err, configx.ErrTypeMismatch)
	require.Equal(t, time.Minute, c.DurationOr("missing", time.Minute))
}

func TestStringSlice_AllPaths(t *testing.T) {
	c := newCfg(t, "a: [x, y, z]\nb: [1, 2]\nc: notalist\n")

	v, err := c.StringSlice("a")
	require.NoError(t, err)
	require.Equal(t, []string{"x", "y", "z"}, v)

	_, err = c.StringSlice("b") // ints inside, type-mismatch
	require.ErrorIs(t, err, configx.ErrTypeMismatch)

	_, err = c.StringSlice("c")
	require.ErrorIs(t, err, configx.ErrTypeMismatch)

	require.Equal(t, []string{"d"}, c.StringSliceOr("missing", []string{"d"}))
}

func TestHas(t *testing.T) {
	c := newCfg(t, "a: 1\n")
	require.True(t, c.Has("a"))
	require.False(t, c.Has("nope"))
}

func TestErrorsAreMatchable(t *testing.T) {
	c := newCfg(t, "a: 1\n")
	_, err := c.String("missing")
	require.True(t, errors.Is(err, configx.ErrKeyNotFound))
}

func TestUnmarshalKey(t *testing.T) {
	c := newCfg(t, "server:\n  host: h\n  port: 99\n")
	var s struct {
		Host string `configx:"host"`
		Port int    `configx:"port"`
	}
	require.NoError(t, c.UnmarshalKey("server", &s))
	require.Equal(t, "h", s.Host)
	require.Equal(t, 99, s.Port)

	require.ErrorIs(t, c.UnmarshalKey("nope", &s), configx.ErrKeyNotFound)
}
