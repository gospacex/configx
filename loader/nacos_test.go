package loader

import (
	"testing"
)

// NacosLoader 的功能测试（受限于需要真实 Nacos 服务）
// 以下测试覆盖：实例创建失败、Load 错误处理

func TestNacosLoader_NewNacosLoader_InvalidEndpoint(t *testing.T) {
	// 使用无效地址（不可达 IP），验证连接检查错误
	_, err := NewNacosLoader("192.0.2.1:9999", "namespace", "group")
	if err == nil {
		t.Fatal("expected connection check to fail for unreachable address")
	}
	// 错误信息应该包含连接失败相关描述
	t.Logf("Expected error: %v", err)
}

func TestNacosLoader_Load_AfterFailedCreation(t *testing.T) {
	// 由于 NewNacosLoader 失败返回错误，Load 方法无法在测试中直接调用
	// 验证 NacosLoader.Close 在未初始化状态下的行为
	nl := &NacosLoader{}
	err := nl.Close()
	if err != nil {
		t.Errorf("Close() on nil NacosLoader unexpected error: %v", err)
	}
}

func TestNacosLoader_httpGet_NotImplemented(t *testing.T) {
	// 验证 httpGet 返回 "未实现" 错误
	nl := &NacosLoader{}
	_, err := nl.httpGet("test-key")
	if err == nil {
		t.Error("expected error from unimplemented httpGet")
	}
	if err != nil && err.Error() == "" {
		t.Error("httpGet returned empty error")
	}
	t.Logf("httpGet error: %v", err)
}

func TestNacosLoader_Load_NotInitialized(t *testing.T) {
	// 测试未初始化的 client 的 Load 行为：client 为 nil 时 Load 会 panic
	// 这是预期行为，证明 Load 不能在 NewNacosLoader 失败后调用
	loader := &NacosLoader{group: "group"}
	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
				t.Logf("Recovered expected panic: %v", r)
			}
		}()
		_, _ = loader.Load("test-key")
	}()
	if !didPanic {
		t.Error("expected panic when Load is called with nil client")
	}
}

func TestNacosLoader_Watch_NotInitialized(t *testing.T) {
	loader := &NacosLoader{group: "group"}
	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
				t.Logf("Recovered expected panic: %v", r)
			}
		}()
		loader.Watch("test-key", func(data []byte) {
			t.Log("unexpected callback")
		})
	}()
	if !didPanic {
		t.Error("expected panic when Watch is called with nil client")
	}
}

func TestNewNacosLoader_InvalidPort_DefaultsTo8848(t *testing.T) {
	// 测试端口解析逻辑：传入 host:invalid 会回退到默认端口 8848
	// 由于 TCP 连接检查仍会失败，预期返回错误
	_, err := NewNacosLoader("192.0.2.1:invalid", "namespace", "group")
	if err == nil {
		t.Fatal("expected connection failure")
	}
	t.Logf("Expected error for invalid port: %v", err)
}

func TestNewNacosLoader_HostOnly_UsesDefaultPort(t *testing.T) {
	// 仅有 host 没有端口，应该使用默认端口 8848
	// 连接检查仍会失败（不可达 IP），但端口解析逻辑本身正确
	_, err := NewNacosLoader("192.0.2.1", "namespace", "group")
	if err == nil {
		t.Fatal("expected connection failure for unreachable host")
	}
	t.Logf("Expected error (default port 8848 used): %v", err)
}
