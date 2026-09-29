// configx/configx_test.go
package configx

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_New(t *testing.T) {
	// 测试本地配置
	cfg, err := New(&ConfigOptions{
		ConfigFile: "example/config.yaml",
		ConfigType: "yaml",
		Remote: RemoteConfig{
			Enabled: false,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, cfg)
	defer cfg.Close()

	// 验证配置读取
	assert.Equal(t, "myapp", cfg.GetString("app.name"))
	assert.Equal(t, "1.0.0", cfg.GetString("app.version"))
	assert.Equal(t, "development", cfg.GetString("app.env"))
}

func TestProvider_Get(t *testing.T) {
	cfg, err := New(&ConfigOptions{
		ConfigFile: "example/config.yaml",
		ConfigType: "yaml",
		Remote: RemoteConfig{
			Enabled: false,
		},
	})
	require.NoError(t, err)
	defer cfg.Close()

	// 测试 Get
	assert.NotNil(t, cfg.Get("app"))

	// 测试 GetString
	assert.Equal(t, "myapp", cfg.GetString("app.name"))

	// 测试 GetInt
	assert.Equal(t, 100, cfg.GetInt("xmiddleware.sqlx.databases.default.max_open_conns"))

	// 测试 GetBool
	assert.False(t, cfg.GetBool("logging.otel.enabled"))

	// 测试 GetDuration
	assert.Equal(t, 5*time.Minute, cfg.GetDuration("xmiddleware.sqlx.databases.default.conn_max_lifetime"))

	// 测试 GetSlice
	brokers := cfg.GetSlice("xmiddleware.mqx.instances.default.brokers")
	assert.Contains(t, brokers, "localhost:9092")
}

func TestProvider_IsSet(t *testing.T) {
	cfg, err := New(&ConfigOptions{
		ConfigFile: "example/config.yaml",
		ConfigType: "yaml",
		Remote: RemoteConfig{
			Enabled: false,
		},
	})
	require.NoError(t, err)
	defer cfg.Close()

	// 测试 IsSet
	assert.True(t, cfg.IsSet("app.name"))
	assert.False(t, cfg.IsSet("app.nonexistent"))
}

func TestProvider_Unmarshal(t *testing.T) {
	cfg, err := New(&ConfigOptions{
		ConfigFile: "example/config.yaml",
		ConfigType: "yaml",
		Remote: RemoteConfig{
			Enabled: false,
		},
	})
	require.NoError(t, err)
	defer cfg.Close()

	// 测试 Unmarshal
	type AppConfig struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Env     string `json:"env"`
	}

	var appCfg AppConfig
	err = cfg.Unmarshal("app", &appCfg)
	require.NoError(t, err)
	assert.Equal(t, "myapp", appCfg.Name)
	assert.Equal(t, "1.0.0", appCfg.Version)
	assert.Equal(t, "development", appCfg.Env)
}

func TestProvider_Watch(t *testing.T) {
	cfg, err := New(&ConfigOptions{
		ConfigFile: "example/config.yaml",
		ConfigType: "yaml",
		Remote: RemoteConfig{
			Enabled: false,
		},
	})
	require.NoError(t, err)
	defer cfg.Close()

	// 测试 Watch
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	callbackCalled := false
	err = cfg.Watch(ctx, "app.name", func(newVal interface{}) {
		callbackCalled = true
	})
	assert.NoError(t, err)

	// 等待超时
	<-ctx.Done()
	// 由于是本地配置且无变更，callback 不应被调用
	assert.False(t, callbackCalled)
}
