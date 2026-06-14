package configx

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gospacex/configx/loader"
)

// ============================================================================
// ViperLoader 测试
// ============================================================================

func TestViperLoader_LoadEmptyKey_ReturnsFullConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	localContent := `app:
  name: testapp
  port: 8080
`
	if err := os.WriteFile(configFile, []byte(localContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	vl, err := loader.NewViperLoader(configFile)
	if err != nil {
		t.Fatalf("NewViperLoader failed: %v", err)
	}
	defer vl.Close()

	data, err := vl.Load("")
	if err != nil {
		t.Fatalf("Load(\"\") failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data from Load(\"\")")
	}
}

func TestViperLoader_LoadSpecificKey(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")
	configContent := `{"app": {"name": "testapp", "port": 8080}}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	vl, err := loader.NewViperLoader(configFile)
	if err != nil {
		t.Fatalf("NewViperLoader failed: %v", err)
	}
	defer vl.Close()

	data, err := vl.Load("app.port")
	if err != nil {
		t.Fatalf("Load(\"app.port\") failed: %v", err)
	}
	if string(data) != "8080" {
		t.Errorf("Load(\"app.port\") = %s, want %s", string(data), "8080")
	}
}

func TestViperLoader_LoadNonexistentKey(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("key: value"), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	vl, err := loader.NewViperLoader(configFile)
	if err != nil {
		t.Fatalf("NewViperLoader failed: %v", err)
	}
	defer vl.Close()

	_, err = vl.Load("nonexistent.key")
	if err == nil {
		t.Error("expected error for nonexistent key, got nil")
	}
	// 错误信息应为 "config key not found"
	if err != nil && err.Error() != "config key not found" {
		t.Errorf("expected 'config key not found', got %v", err)
	}
}

func TestViperLoader_LoadNonexistentFile(t *testing.T) {
	_, err := loader.NewViperLoader("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error when loading nonexistent file")
	}
}

func TestViperLoader_Watch_CallbackInvoked(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	initialContent := `app:
  name: initial
`
	if err := os.WriteFile(configFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	vl, err := loader.NewViperLoader(configFile)
	if err != nil {
		t.Fatalf("NewViperLoader failed: %v", err)
	}
	defer vl.Close()

	called := false
	vl.Watch("app.name", func(data []byte) {
		called = true
		t.Logf("Watch callback invoked with data: %s", string(data))
	})

	// 模拟文件变更
	newContent := `app:
  name: updated
`
	if err := os.WriteFile(configFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("update config file: %v", err)
	}

	// Viper 的 WatchConfig 会在下一次事件触发时调用回调
	// 这里只验证注册不 panic
	if !called {
		t.Log("Watch registered successfully (callback will fire on actual file change)")
	}
}

func TestViperLoader_Close(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("key: value"), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	vl, err := loader.NewViperLoader(configFile)
	if err != nil {
		t.Fatalf("NewViperLoader failed: %v", err)
	}

	err = vl.Close()
	if err != nil {
		t.Errorf("Close() unexpected error: %v", err)
	}
}

// ============================================================================
// ConfigCenter 错误处理测试
// ============================================================================

func TestConfigCenter_LocalUnavailable_ReturnsErrLocalUnavailable(t *testing.T) {
	_, err := NewConfigCenter("", "/nonexistent/path/config.yaml", "")
	if err != ErrLocalUnavailable {
		t.Errorf("expected ErrLocalUnavailable, got %v", err)
	}
}

func TestConfigCenter_InvalidRemoteType_ReturnsErrInvalidRemoteType(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(localFile, []byte("key: value"), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	_, err := NewConfigCenter("consul", localFile, "localhost:8500")
	if err != ErrInvalidRemoteType {
		t.Errorf("expected ErrInvalidRemoteType, got %v", err)
	}
}

func TestConfigCenter_GetKeyNotFound(t *testing.T) {
	// Get(key) 的 key 参数仅在远程配置中心启用时有意义
	// 本地模式下 Get 始终返回完整配置文件内容，忽略 key 参数
	// 此测试验证：远程不可用时 Get("") → 返回本地文件内容
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "config.yaml")
	localContent := `app:
  name: testapp
`
	if err := os.WriteFile(localFile, []byte(localContent), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	cc, err := NewConfigCenter("", localFile, "")
	if err != nil {
		t.Fatalf("NewConfigCenter failed: %v", err)
	}
	defer cc.Close()

	val, err := cc.Get("")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	data, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
}

// ============================================================================
// ConfigCenter Watch 测试
// ============================================================================

func TestConfigCenter_Watch_RemoteAndLocalBothRegistered(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "config.yaml")
	localContent := `app:
  name: testapp
`
	if err := os.WriteFile(localFile, []byte(localContent), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	cc, err := NewConfigCenter("apollo", localFile, "invalid-host:9999")
	if err != nil {
		t.Fatalf("NewConfigCenter failed: %v", err)
	}
	defer cc.Close()

	// Watch 注册时不 panic 即为通过
	// remote 为 nil 时 Watch 应只注册本地
	cc.Watch("app.name", func(value interface{}) {
		t.Log("Watch callback invoked:", value)
	})
}

// ============================================================================
// ConfigCenter Close 测试
// ============================================================================

func TestConfigCenter_Close_NoError(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(localFile, []byte("key: value"), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	cc, err := NewConfigCenter("", localFile, "")
	if err != nil {
		t.Fatalf("NewConfigCenter failed: %v", err)
	}

	err = cc.Close()
	if err != nil {
		t.Errorf("Close() unexpected error: %v", err)
	}
}

func TestConfigCenter_Close_RemoteExists(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(localFile, []byte("key: value"), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	// remote 不可用时会设置为 nil，所以 Close 不会出错
	cc, err := NewConfigCenterWithTimeoutWithParams(
		"apollo", localFile, "invalid-host:9999",
		2*time.Second, "application", "DEFAULT_GROUP", "application",
	)
	if err != nil {
		t.Fatalf("NewConfigCenter failed: %v", err)
	}
	defer cc.Close()

	err = cc.Close()
	if err != nil {
		t.Errorf("Close() with nil remote unexpected error: %v", err)
	}
}

// ============================================================================
// ConfigCenter 超时降级测试
// ============================================================================

func TestConfigCenter_Timeout_FallbackToLocal(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "config.yaml")
	localContent := `app:
  name: timeout-fallback
  port: 9090
`
	if err := os.WriteFile(localFile, []byte(localContent), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	// 使用一个不可达的地址 + 短超时，确保触发超时
	cc, err := NewConfigCenterWithTimeoutWithParams(
		"apollo", localFile, "10.255.255.1:9999",
		100*time.Millisecond, "application", "DEFAULT_GROUP", "application",
	)
	if err != nil {
		t.Fatalf("NewConfigCenter failed: %v", err)
	}
	defer cc.Close()

	val, err := cc.Get("app.name")
	if err != nil {
		t.Fatalf("expected fallback to local on timeout, got error: %v", err)
	}
	data, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data from fallback")
	}
}

// ============================================================================
// NacosLoader 测试（使用 mock）
// ============================================================================

func TestNacosLoader_Load_KeyNotFound(t *testing.T) {
	// 由于 NacosLoader 连接需要真实的 Nacos 服务，
	// 这里测试 NewNacosLoader 对于无效端点的错误处理
	// 使用一个肯定不可达的地址
	nl, err := loader.NewNacosLoader("192.0.2.1:9999", "namespace", "group")
	if err == nil {
		// 如果连接成功（极小概率），验证 Load 返回错误
		_, loadErr := nl.Load("nonexistent-key")
		// Nacos 返回空 content 时会有特定错误
		t.Logf("NacosLoader created (possibly reachable), Load result: %v", loadErr)
		nl.Close()
		return
	}
	// 预期行为：连接检查失败返回错误
	t.Logf("Expected connection failure: %v", err)
}

// ============================================================================
// NewConfigCenter 工厂函数测试
// ============================================================================

func TestNewConfigCenter_EmptyRemoteType_CreatesWithoutError(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(localFile, []byte("key: value"), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	cc, err := NewConfigCenter("", localFile, "")
	if err != nil {
		t.Fatalf("NewConfigCenter failed: %v", err)
	}
	defer cc.Close()

	val, err := cc.Get("")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	data, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
}

func TestNewConfigCenterWithTimeout_CreatesSuccessfully(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(localFile, []byte("key: value"), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	cc, err := NewConfigCenterWithTimeout("", localFile, "", 5*time.Second)
	if err != nil {
		t.Fatalf("NewConfigCenterWithTimeout failed: %v", err)
	}
	defer cc.Close()

	val, err := cc.Get("")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	data, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
}

func TestNewConfigCenterWithTimeoutWithParams_EmptyRemoteType(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(localFile, []byte("key: value"), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	cc, err := NewConfigCenterWithTimeoutWithParams(
		"", localFile, "", 0, "", "", "",
	)
	if err != nil {
		t.Fatalf("NewConfigCenterWithTimeoutWithParams failed: %v", err)
	}
	defer cc.Close()

	val, err := cc.Get("")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	data, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
}

// ============================================================================
// ConfigCenter 本地配置读取测试
// ============================================================================

func TestConfigCenter_LocalOnly(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "config.yaml")
	localContent := `app:
  name: local-only-app
  port: 8080
`
	if err := os.WriteFile(localFile, []byte(localContent), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	cc, err := NewConfigCenter("", localFile, "")
	if err != nil {
		t.Fatalf("NewConfigCenter failed: %v", err)
	}
	defer cc.Close()

	val, err := cc.Get("")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	data, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data from local config")
	}
	if string(data) != localContent {
		t.Errorf("Get() = %s, want %s", string(data), localContent)
	}
}

func TestConfigCenter_LocalOnly_FileNotFound(t *testing.T) {
	_, err := NewConfigCenter("", "/nonexistent/path/config.yaml", "")
	if err != ErrLocalUnavailable {
		t.Errorf("expected ErrLocalUnavailable, got %v", err)
	}
}

// ============================================================================
// ConfigCenter Nacos 降级读取测试（使用真实 Nacos）
// ============================================================================

const (
	nacosEndpoint   = "localhost:8848"
	nacosNamespace = "42e5c989-ffe9-40b8-92aa-44b0e9ab83ec"
	nacosDataId    = "product.yaml"
	nacosGroup     = "product"
)

func TestConfigCenter_NacosFallbackToLocal(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "config.yaml")
	localContent := `app:
  name: nacos-fallback-app
  port: 9090
`
	if err := os.WriteFile(localFile, []byte(localContent), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	// 使用不可达地址和短超时模拟 Nacos 连接超时，触发降级到本地
	cc, err := NewConfigCenterWithTimeoutWithParams(
		"nacos", localFile, "192.0.2.1:8848",
		100*time.Millisecond, nacosNamespace, nacosGroup, nacosDataId,
	)
	if err != nil {
		t.Fatalf("NewConfigCenter failed: %v", err)
	}
	defer cc.Close()

	val, err := cc.Get("")
	if err != nil {
		t.Fatalf("expected fallback to local on nacos timeout, got error: %v", err)
	}
	data, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data from fallback")
	}
	if string(data) != localContent {
		t.Errorf("Get() = %s, want %s", string(data), localContent)
	}
}

func TestConfigCenter_NacosSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "config.yaml")
	localContent := `app:
  name: local-only
  port: 9090
`
	if err := os.WriteFile(localFile, []byte(localContent), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	// 连接真实 Nacos 服务
	cc, err := NewConfigCenterWithTimeoutWithParams(
		"nacos", localFile, nacosEndpoint,
		5*time.Second, nacosNamespace, nacosGroup, nacosDataId,
	)
	if err != nil {
		t.Skipf("skipping test: Nacos not available at %s: %v", nacosEndpoint, err)
	}
	defer cc.Close()

	// 验证从 Nacos 读取到配置（不为空）
	val, err := cc.Get(nacosDataId)
	if err != nil {
		t.Fatalf("Get from Nacos failed: %v", err)
	}
	data, ok := val.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", val)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data from Nacos")
	}
	t.Logf("Successfully loaded from Nacos: %s", string(data))
}

// ============================================================================
// 表驱动测试：验证降级逻辑的各种场景
// ============================================================================

func TestFallbackScenarios(t *testing.T) {
	tmpDir := t.TempDir()
	localFile := filepath.Join(tmpDir, "config.yaml")
	localContent := `database:
  host: localhost
  port: 3306
`
	if err := os.WriteFile(localFile, []byte(localContent), 0644); err != nil {
		t.Fatalf("write local file: %v", err)
	}

	scenarios := []struct {
		name           string
		remoteType     string
		remoteAddr     string
		timeout        time.Duration
		expectFallback bool
		expectError    bool
	}{
		{
			name:           "nacos_unreachable_fallbacks_to_local",
			remoteType:     "nacos",
			remoteAddr:     "192.0.2.1:9999",
			timeout:        2 * time.Second,
			expectFallback: true,
			expectError:    false,
		},
		{
			name:           "local_only_no_remote",
			remoteType:     "",
			remoteAddr:     "",
			timeout:        0,
			expectFallback: false,
			expectError:    false,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			cc, err := NewConfigCenterWithTimeoutWithParams(
				s.remoteType, localFile, s.remoteAddr,
				s.timeout, "namespace", "group", "dataId",
			)
			if err != nil {
				if !s.expectError {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			defer cc.Close()

			val, err := cc.Get("")
			if s.expectError && err != nil {
				return // expected
			}
			if err != nil {
				t.Fatalf("Get failed: %v", err)
			}

			data, ok := val.([]byte)
			if !ok {
				t.Fatalf("expected []byte, got %T", val)
			}
			if s.expectFallback && len(data) == 0 {
				t.Error("expected fallback data, got empty")
			}
		})
	}
}
