package configx_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gospacex/configx"
	"github.com/gospacex/configx/encrypt"
	"github.com/gospacex/configx/source/env"
	"github.com/gospacex/configx/source/file"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(body), 0o644))
	return p
}

func TestNew_FileSourceBasic(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "app.yaml", `
server:
  host: "0.0.0.0"
  port: 8080
log: info
tags: [a, b, c]
`)

	c, err := configx.New(
		configx.WithSource(file.New(p, file.WithWatch(false))),
	)
	require.NoError(t, err)
	defer c.Close()

	require.Equal(t, "0.0.0.0", c.StringOr("server.host", ""))
	port, err := c.Int("server.port")
	require.NoError(t, err)
	require.EqualValues(t, 8080, port)

	tags, err := c.StringSlice("tags")
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "c"}, tags)
}

func TestNew_MergeFileEnv(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "app.yaml", `
server:
  host: "0.0.0.0"
  port: 8080
`)

	t.Setenv("APP_SERVER_PORT", "9090")
	t.Setenv("APP_FEATURE_BETA", "true")

	c, err := configx.New(
		configx.WithSource(file.New(p, file.WithWatch(false))),
		configx.WithSource(env.New(env.WithPrefix("APP"))),
	)
	require.NoError(t, err)
	defer c.Close()

	// env overrides file
	require.EqualValues(t, 9090, c.IntOr("server.port", 0))
	require.Equal(t, "0.0.0.0", c.StringOr("server.host", ""))
	require.True(t, c.BoolOr("feature.beta", false))
}

type appCfg struct {
	Server struct {
		Host string `configx:"host" validate:"required"`
		Port int    `configx:"port" validate:"required,min=1,max=65535"`
	} `configx:"server"`
	Timeout time.Duration `configx:"timeout"`
}

func TestUnmarshal_WithValidation(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "app.yaml", `
server:
  host: "0.0.0.0"
  port: 8080
timeout: "1500ms"
`)

	c, err := configx.New(configx.WithSource(file.New(p, file.WithWatch(false))))
	require.NoError(t, err)
	defer c.Close()

	var cfg appCfg
	require.NoError(t, c.Unmarshal(&cfg))
	require.Equal(t, "0.0.0.0", cfg.Server.Host)
	require.Equal(t, 8080, cfg.Server.Port)
	require.Equal(t, 1500*time.Millisecond, cfg.Timeout)
}

func TestUnmarshal_ValidationFailure(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "app.yaml", `
server:
  host: ""
  port: 0
`)
	c, err := configx.New(configx.WithSource(file.New(p, file.WithWatch(false))))
	require.NoError(t, err)
	defer c.Close()

	var cfg appCfg
	err = c.Unmarshal(&cfg)
	require.Error(t, err)
}

func TestEncryptedField(t *testing.T) {
	cipher, err := encrypt.NewAESGCM([]byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)

	enc, err := cipher.Encrypt("s3cr3t")
	require.NoError(t, err)

	dir := t.TempDir()
	p := writeFile(t, dir, "app.yaml", `
db:
  password: "enc(`+enc+`)"
`)
	c, err := configx.New(
		configx.WithSource(file.New(p, file.WithWatch(false))),
		configx.WithCipher(cipher),
	)
	require.NoError(t, err)
	defer c.Close()

	require.Equal(t, "s3cr3t", c.StringOr("db.password", ""))
}

func TestHotReload(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "app.yaml", "v: 1\n")

	c, err := configx.New(configx.WithSource(
		file.New(p, file.WithDebounce(20*time.Millisecond)),
	))
	require.NoError(t, err)
	defer c.Close()

	changed := make(chan map[string]any, 1)
	c.OnChange(func(snap map[string]any) {
		select {
		case changed <- snap:
		default:
		}
	})

	require.EqualValues(t, 1, c.IntOr("v", 0))
	require.NoError(t, os.WriteFile(p, []byte("v: 2\n"), 0o644))

	select {
	case <-changed:
	case <-time.After(3 * time.Second):
		t.Fatal("expected OnChange to fire after file write")
	}
	require.EqualValues(t, 2, c.IntOr("v", 0))
}

func TestReload_Manual(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "app.yaml", "v: 1\n")

	c, err := configx.New(configx.WithSource(file.New(p, file.WithWatch(false))))
	require.NoError(t, err)
	defer c.Close()

	require.NoError(t, os.WriteFile(p, []byte("v: 42\n"), 0o644))
	require.NoError(t, c.Reload(context.Background()))
	require.EqualValues(t, 42, c.IntOr("v", 0))
}

func TestSnapshot_IsDeepCopy(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "app.yaml", "server:\n  host: localhost\n")
	c, err := configx.New(configx.WithSource(file.New(p, file.WithWatch(false))))
	require.NoError(t, err)
	defer c.Close()

	snap := c.Snapshot()
	snap["server"].(map[string]any)["host"] = "evil"
	require.Equal(t, "localhost", c.StringOr("server.host", ""))
}

func TestJSONFormat(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "app.json", `{"k": "v", "n": 7}`)
	c, err := configx.New(configx.WithSource(file.New(p, file.WithWatch(false))))
	require.NoError(t, err)
	defer c.Close()
	require.Equal(t, "v", c.StringOr("k", ""))
	require.EqualValues(t, 7, c.IntOr("n", 0))
}

func TestTOMLFormat(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "app.toml", "k = \"v\"\n[srv]\nport = 9090\n")
	c, err := configx.New(configx.WithSource(file.New(p, file.WithWatch(false))))
	require.NoError(t, err)
	defer c.Close()
	require.Equal(t, "v", c.StringOr("k", ""))
	require.EqualValues(t, 9090, c.IntOr("srv.port", 0))
}

func TestOptionalFile(t *testing.T) {
	c, err := configx.New(configx.WithSource(
		file.New("/no/such/file.yaml", file.WithOptional(true), file.WithFormat("yaml"), file.WithWatch(false)),
	))
	require.NoError(t, err)
	defer c.Close()
	require.False(t, c.Has("anything"))
}
