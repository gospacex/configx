// configx/configx.go
// 统一配置接口，为所有 x 系列 SDK 提供配置句柄
package configx

import (
	"context"
	"fmt"
	"os"
	"time"
)

// Config 统一配置接口
// 所有 x 系列 SDK 通过此接口获取配置
type Config interface {
	// Get 获取配置值（interface{}）
	Get(key string) interface{}
	// GetString 获取字符串值
	GetString(key string) string
	// GetInt 获取整数值
	GetInt(key string) int
	// GetInt64 获取 int64 值
	GetInt64(key string) int64
	// GetBool 获取布尔值
	GetBool(key string) bool
	// GetFloat64 获取 float64 值
	GetFloat64(key string) float64
	// GetDuration 获取时间间隔
	GetDuration(key string) time.Duration
	// GetSlice 获取字符串切片
	GetSlice(key string) []string
	// GetMap 获取 map
	GetMap(key string) map[string]interface{}
	// Unmarshal 反序列化到结构体
	Unmarshal(key string, out interface{}) error
	// IsSet 判断 key 是否存在
	IsSet(key string) bool

	// Watch 监听配置变更
	// key 为空时监听所有变更
	Watch(ctx context.Context, key string, callback func(newVal interface{})) error
	// Close 关闭配置中心连接
	Close() error
}

// ConfigOptions 配置选项
type ConfigOptions struct {
	// ===== 本地配置（默认）=====
	ConfigFile string `yaml:"config_file" json:"config_file"`
	ConfigType string `yaml:"config_type" json:"config_type"` // yaml, json, toml
	ConfigPath string `yaml:"config_path" json:"config_path"` // 配置搜索路径

	// ===== 远程配置开关 =====
	Remote RemoteConfig `yaml:"remote" json:"remote"`
}

// RemoteConfig 远程配置
type RemoteConfig struct {
	// 开关：是否开启远程配置（默认 false，使用 viper 本地配置）
	Enabled bool `yaml:"enabled" json:"enabled" env:"REMOTE_ENABLED"`

	// 远程配置中心类型：nacos
	Provider string `yaml:"provider" json:"provider" env:"REMOTE_PROVIDER"`

	// ===== Nacos 配置（开启远程时使用）=====
	Nacos NacosConfig `yaml:"nacos" json:"nacos"`

	// ===== Apollo 配置（开启远程时使用）=====
	Apollo *ApolloConfig `yaml:"apollo" json:"apollo"`
}

// NacosConfig Nacos 配置中心配置
type NacosConfig struct {
	ServerConfigs []NacosServerConfig `yaml:"server_configs" json:"server_configs"`
	NamespaceID   string              `yaml:"namespace_id" json:"namespace_id"`
	Group         string              `yaml:"group" json:"group"`     // 默认 DEFAULT_GROUP
	DataID        string              `yaml:"data_id" json:"data_id"` // 配置 data_id
	ClusterName   string              `yaml:"cluster_name" json:"cluster_name"`
	Username      string              `yaml:"username" json:"username"`
	Password      string              `yaml:"password" json:"password"`
	// 配置刷新
	RefreshInterval time.Duration `yaml:"refresh_interval" json:"refresh_interval"` // 默认 30s
}

// NacosServerConfig Nacos 服务器配置
type NacosServerConfig struct {
	IP      string `yaml:"ip" json:"ip"`
	Port    uint64 `yaml:"port" json:"port"`
	Context string `yaml:"context" json:"context"`
}

// ApolloServerConfig Apollo 服务器配置（集群模式）
type ApolloServerConfig struct {
	Host string `yaml:"host" json:"host"`
	Port uint16 `yaml:"port" json:"port"`
}

// ApolloConfig Apollo 配置中心配置
type ApolloConfig struct {
	AppID         string               `yaml:"app_id" json:"app_id" env:"APOLLO_APP_ID"`
	Cluster       string               `yaml:"cluster" json:"cluster" env:"APOLLO_CLUSTER"`
	Namespace     string               `yaml:"namespace" json:"namespace" env:"APOLLO_NAMESPACE"`
	SecretKey     string               `yaml:"secret_key" json:"secret_key" env:"APOLLO_SECRET_KEY"`
	IP            string               `yaml:"ip" json:"ip" env:"APOLLO_IP"`
	ServerConfigs []ApolloServerConfig `yaml:"server_configs" json:"server_configs"`
	ServerAddr    string               `yaml:"server_addr" json:"server_addr" env:"APOLLO_SERVER_ADDR"`
}

// ReadConfig 创建一个统一配置句柄，供所有 x 系列 SDK 使用。
//
// 自动检测逻辑：
//  1. ConfigFile 已显式设置 → 直接使用
//  2. HUBX_CONFIG 环境变量 → 使用环境变量值
//  3. 默认路径 ./config.yaml
//
// 远程配置开关（nacos/apollo）由配置文件中的 configx.remote.enabled 控制，
// 或通过在 ConfigOptions.Remote 中显式设置来覆盖。
func ReadConfig(opts ...ConfigOptions) (Config, error) {
	var cfg ConfigOptions
	if len(opts) > 0 {
		cfg = opts[0]
	}

	if cfg.ConfigFile == "" {
		if p := os.Getenv("HUBX_CONFIG"); p != "" {
			cfg.ConfigFile = p
		} else {
			cfg.ConfigFile = "config.yaml"
		}
	}

	c, err := New(&cfg)
	if err != nil {
		return nil, fmt.Errorf("configx: %w", err)
	}
	return c, nil
}
