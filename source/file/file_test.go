package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gospacex/configx"
)

func TestFile_LoadYAML(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yaml")
	require.NoError(t, os.WriteFile(p, []byte("k: v\n"), 0o644))

	s := New(p, WithWatch(false))
	m, err := s.Load(context.Background())
	require.NoError(t, err)
	require.Equal(t, "v", m["k"])
}

func TestFile_WatchEmitsOnWrite(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yaml")
	require.NoError(t, os.WriteFile(p, []byte("k: 1\n"), 0o644))

	s := New(p, WithDebounce(20*time.Millisecond))
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := s.Watch(ctx)
	require.NoError(t, err)
	require.NotNil(t, ch)

	require.NoError(t, os.WriteFile(p, []byte("k: 2\n"), 0o644))

	select {
	case ev := <-ch:
		require.Equal(t, "2", asString(ev.Data["k"]))
	case <-time.After(3 * time.Second):
		t.Fatal("no event")
	}
}

func TestFile_Optional(t *testing.T) {
	s := New("/no/such.yaml", WithOptional(true), WithFormat("yaml"), WithWatch(false))
	m, err := s.Load(context.Background())
	require.NoError(t, err)
	require.Empty(t, m)
}

func TestFile_ID(t *testing.T) {
	s := New("foo.yaml")
	require.NotEmpty(t, s.ID())
	require.NotEqual(t, "file:", s.ID())
}

func TestFile_WatchClosed(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yaml")
	require.NoError(t, os.WriteFile(p, []byte("k: 1\n"), 0o644))
	s := New(p)
	require.NoError(t, s.Close())
	_, err := s.Watch(context.Background())
	require.ErrorIs(t, err, configx.ErrSourceClosed)
}

func asString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case int:
		return string(rune('0' + x))
	}
	return ""
}
