package configx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gospacex/traceflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withTraceflag 临时设置 traceflag 全局总闸并在测试结束时恢复。
func withTraceflag(t *testing.T, v bool) {
	t.Helper()
	prev := traceflag.Enabled()
	traceflag.SetEnabled(v)
	t.Cleanup(func() { traceflag.SetEnabled(prev) })
}

func TestNew_LocalOnly(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(cfgFile, []byte(`
server:
  host: localhost
  port: 8080
`), 0644)
	require.NoError(t, err)

	cfg := &ConfigOptions{
		ConfigFile: cfgFile,
	}
	c, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "localhost", c.GetString("server.host"))
	assert.Equal(t, 8080, c.GetInt("server.port"))
}

func TestNew_RemoteDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(cfgFile, []byte(`
server:
  host: localhost
`), 0644)
	require.NoError(t, err)

	cfg := &ConfigOptions{
		ConfigFile: cfgFile,
		Remote: RemoteConfig{
			Enabled: false,
		},
	}
	c, err := New(cfg)
	require.NoError(t, err)
	assert.Equal(t, "localhost", c.GetString("server.host"))
}

func TestReadApolloConfigFromLocal(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(cfgFile, []byte(`
configx:
  remote:
    enabled: true
    provider: apollo
    apollo:
      app_id: test-app
      cluster: default
      namespace: application
      ip: 127.0.0.1:8070
`), 0644)
	require.NoError(t, err)

	cfg := &ConfigOptions{
		ConfigFile: cfgFile,
	}
	_, err = New(cfg)
	// Should fail because apollo server is not running — but config is parsed correctly
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apollo")
}

func TestNew_ProgrammaticApollo(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(cfgFile, []byte(`
server:
  host: localhost
`), 0644)
	require.NoError(t, err)

	cfg := &ConfigOptions{
		ConfigFile: cfgFile,
		Remote: RemoteConfig{
			Enabled:  true,
			Provider: "apollo",
			Apollo: &ApolloConfig{
				AppID:     "test-app",
				Cluster:   "default",
				Namespace: "application",
				IP:        "127.0.0.1:8070",
			},
		},
	}
	_, err = New(cfg)
	// Should fail because apollo server is not running
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apollo")
}

func TestNew_UnsupportedProvider(t *testing.T) {
	cfg := &ConfigOptions{
		Remote: RemoteConfig{
			Enabled:  true,
			Provider: "etcd",
		},
	}
	_, err := New(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestNew_EmptyConfig(t *testing.T) {
	c, err := New(nil)
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestReadApolloConfigFromLocal_WithServerConfigs(t *testing.T) {
	// Verify readApolloConfigFromLocal parses server_configs correctly
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(cfgFile, []byte(`
configx:
  remote:
    enabled: true
    provider: apollo
    apollo:
      app_id: test-app
      server_configs:
        - host: apollo1.example.com
          port: 8070
        - host: apollo2.example.com
          port: 8070
`), 0644)
	require.NoError(t, err)

	cfg := &ConfigOptions{
		ConfigFile: cfgFile,
	}
	_, err = New(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apollo")
}

// TestTraceflagGlobalKey 验证配置文件里统一的调用链总开关键：
// 显式设置 traceflag.enabled 时自动映射到 traceflag.SetEnabled；
// 未设置时保持缺省放行（true，opt-out）。
func TestTraceflagGlobalKey(t *testing.T) {
	withTraceflag(t, true)
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(cfgFile, []byte(`
app:
  name: myapp
traceflag:
  enabled: false
`), 0644)
	require.NoError(t, err)

	c, err := New(&ConfigOptions{ConfigFile: cfgFile})
	require.NoError(t, err)
	_ = c
	assert.False(t, traceflag.Enabled(), "traceflag.enabled: false 应映射到全局总闸")

	// 配置文件缺省该键时不覆盖，保持缺省放行
	traceflag.SetEnabled(true)
	plainFile := filepath.Join(tmpDir, "plain.yaml")
	require.NoError(t, os.WriteFile(plainFile, []byte("app:\n  name: plain\n"), 0644))
	c2, err := New(&ConfigOptions{ConfigFile: plainFile})
	require.NoError(t, err)
	_ = c2
	assert.True(t, traceflag.Enabled(), "未配置 traceflag.enabled 时应保持缺省放行")
}

// TestTraceflagGlobalKey_EmptyConfig 空配置（无任何配置文件）不触碰 traceflag。
func TestTraceflagGlobalKey_EmptyConfig(t *testing.T) {
	withTraceflag(t, false)
	c, err := New(nil)
	require.NoError(t, err)
	_ = c
	assert.False(t, traceflag.Enabled(), "缺省配置不应覆盖总闸")
}
