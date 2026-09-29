// configx/nacos/nacos.go
// Nacos 远程配置实现
package nacos

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"gopkg.in/yaml.v3"
)

// Config Nacos 配置实现
type Config struct {
	client    config_client.IConfigClient
	namespace string
	group     string
	dataID    string
	mu        sync.RWMutex
	data      map[string]interface{}
	watchers  []watcher
}

type watcher struct {
	key      string
	callback func(newVal interface{})
}

// New 创建 Nacos 配置实例
func New(cfg *NacosConfig) (*Config, error) {
	if len(cfg.ServerConfigs) == 0 {
		return nil, fmt.Errorf("nacos server configs is required")
	}

	// 创建 server configs
	serverConfigs := make([]constant.ServerConfig, 0, len(cfg.ServerConfigs))
	for _, sc := range cfg.ServerConfigs {
		serverConfigs = append(serverConfigs, constant.ServerConfig{
			IpAddr:      sc.IP,
			Port:        sc.Port,
			ContextPath: sc.Context,
		})
	}

	// 创建客户端
	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig: &constant.ClientConfig{
				NamespaceId: cfg.NamespaceID,
				Username:    cfg.Username,
				Password:    cfg.Password,
			},
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create nacos client: %w", err)
	}

	// 获取配置
	data, err := getConfig(client, cfg.NamespaceID, cfg.Group, cfg.DataID)
	if err != nil {
		return nil, fmt.Errorf("get nacos config: %w", err)
	}

	return &Config{
		client:    client,
		namespace: cfg.NamespaceID,
		group:     cfg.Group,
		dataID:    cfg.DataID,
		data:      data,
	}, nil
}

// getConfig 从 Nacos 获取配置
func getConfig(client config_client.IConfigClient, namespace, group, dataID string) (map[string]interface{}, error) {
	content, err := client.GetConfig(vo.ConfigParam{
		DataId: dataID,
		Group:  group,
	})
	if err != nil {
		return nil, err
	}

	// 先尝试 JSON 解析
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err == nil {
		return data, nil
	}

	// 再尝试 YAML 解析
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &yamlData); err == nil {
		// 将嵌套 map 展平为点分隔的 key
		flat := make(map[string]interface{})
		flattenMap(yamlData, "", flat)
		return flat, nil
	}

	return nil, fmt.Errorf("unsupported config format for dataId=%s", dataID)
}

// flattenMap 将嵌套 map 展平为点分隔的 key
func flattenMap(data map[string]interface{}, prefix string, result map[string]interface{}) {
	for k, v := range data {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		switch val := v.(type) {
		case map[string]interface{}:
			flattenMap(val, key, result)
		case []interface{}:
			// 数组转为字符串切片
			strs := make([]string, 0, len(val))
			for _, item := range val {
				strs = append(strs, fmt.Sprintf("%v", item))
			}
			result[key] = strs
		default:
			result[key] = val
		}
	}
}

// Get 获取配置值
func (c *Config) Get(key string) interface{} {
	return c.getByPath(key)
}

// getByPath 通过路径获取值（支持 a.b.c 格式）
func (c *Config) getByPath(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// 直接查找点分隔的 key
	if val, ok := c.data[key]; ok {
		return val
	}

	// 尝试按路径逐级查找
	keys := strings.Split(key, ".")
	var result interface{} = c.data

	for _, k := range keys {
		if m, ok := result.(map[string]interface{}); ok {
			result = m[k]
		} else {
			return nil
		}
	}

	return result
}

// GetString 获取字符串值
func (c *Config) GetString(key string) string {
	v := c.Get(key)
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// GetInt 获取整数值
func (c *Config) GetInt(key string) int {
	v := c.Get(key)
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	default:
		return 0
	}
}

// GetInt64 获取 int64 值
func (c *Config) GetInt64(key string) int64 {
	v := c.Get(key)
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case string:
		i, _ := strconv.ParseInt(val, 10, 64)
		return i
	default:
		return 0
	}
}

// GetBool 获取布尔值
func (c *Config) GetBool(key string) bool {
	v := c.Get(key)
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		b, _ := strconv.ParseBool(val)
		return b
	default:
		return false
	}
}

// GetFloat64 获取 float64 值
func (c *Config) GetFloat64(key string) float64 {
	v := c.Get(key)
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

// GetDuration 获取时间间隔
func (c *Config) GetDuration(key string) time.Duration {
	v := c.Get(key)
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case time.Duration:
		return val
	case string:
		d, _ := time.ParseDuration(val)
		return d
	default:
		return 0
	}
}

// GetSlice 获取字符串切片
func (c *Config) GetSlice(key string) []string {
	v := c.Get(key)
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	case []string:
		return val
	default:
		return nil
	}
}

// GetMap 获取 map
func (c *Config) GetMap(key string) map[string]interface{} {
	v := c.Get(key)
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// Unmarshal 反序列化到结构体
func (c *Config) Unmarshal(key string, out interface{}) error {
	v := c.Get(key)
	if v == nil {
		return fmt.Errorf("key %s not found", key)
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, out)
}

// IsSet 判断 key 是否存在
func (c *Config) IsSet(key string) bool {
	return c.Get(key) != nil
}

// Watch 监听配置变更
func (c *Config) Watch(ctx context.Context, key string, callback func(newVal interface{})) error {
	c.watchers = append(c.watchers, watcher{key: key, callback: callback})

	// 监听 Nacos 配置变更
	err := c.client.ListenConfig(vo.ConfigParam{
		Group:  c.group,
		DataId: c.dataID,
		OnChange: func(namespace, group, dataId, data string) {
			// 解析新配置
			var newData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &newData); err == nil {
				c.mu.Lock()
				c.data = newData
				c.mu.Unlock()
			} else {
				var yamlData map[string]interface{}
				if err := yaml.Unmarshal([]byte(data), &yamlData); err == nil {
					flat := make(map[string]interface{})
					flattenMap(yamlData, "", flat)
					c.mu.Lock()
					c.data = flat
					c.mu.Unlock()
				}
			}

			// 通知所有 watcher
			for _, w := range c.watchers {
				if w.key == "" {
					w.callback(c.Get(w.key))
				} else {
					w.callback(c.getByPath(w.key))
				}
			}
		},
	})

	return err
}

// Close 关闭配置
func (c *Config) Close() error {
	c.watchers = nil
	return nil
}

// NacosConfig Nacos 配置选项
type NacosConfig struct {
	ServerConfigs []NacosServerConfig `yaml:"server_configs"`
	NamespaceID   string              `yaml:"namespace_id"`
	Group         string              `yaml:"group"`
	DataID        string              `yaml:"data_id"`
	Username      string              `yaml:"username"`
	Password      string              `yaml:"password"`
}

// NacosServerConfig Nacos 服务器配置
type NacosServerConfig struct {
	IP      string `yaml:"ip"`
	Port    uint64 `yaml:"port"`
	Context string `yaml:"context"`
}
