// configx/apollo/apollo.go
// Apollo 远程配置实现
package apollo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/apollo.v0"
	"gopkg.in/yaml.v3"
)

// Config Apollo 配置实现
type Config struct {
	client    *apollo.Client
	namespace string
	mu        sync.RWMutex
	data      map[string]interface{}
	watchers  []watcher
}

type watcher struct {
	key      string
	callback func(newVal interface{})
}

// New 创建 Apollo 配置实例
func New(cfg *ApolloConfig) (*Config, error) {
	if cfg.AppID == "" {
		return nil, fmt.Errorf("apollo app_id is required")
	}
	if cfg.IP == "" {
		return nil, fmt.Errorf("apollo ip is required")
	}

	// 构建 Conf
	conf := &apollo.Conf{
		AppID:      cfg.AppID,
		Cluster:    cfg.Cluster,
		Namespaces: []string{cfg.Namespace},
		IP:         cfg.IP,
	}

	client := apollo.NewClient(conf)

	// client.Start() may panic if the server is unreachable (library bug: nil longPoller).
	// Recover and return a clean error.
	var startErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				startErr = fmt.Errorf("start apollo client: %v", r)
			}
		}()
		if err := client.Start(); err != nil {
			startErr = fmt.Errorf("start apollo client: %w", err)
		}
	}()
	if startErr != nil {
		return nil, startErr
	}

	// 获取初始配置
	ns := cfg.Namespace
	if ns == "" {
		ns = "application"
	}
	content := client.GetNameSpaceContent(ns, "")

	// 解析配置
	data, err := parseConfig(content, ns)
	if err != nil {
		// 忽略解析错误，使用空配置
		data = make(map[string]interface{})
	}

	cfg2 := &Config{
		client:    client,
		namespace: ns,
		data:      data,
	}

	// 启动变更监听
	go cfg2.watchUpdates()

	return cfg2, nil
}

// parseConfig 解析 Apollo 配置内容
func parseConfig(content, namespace string) (map[string]interface{}, error) {
	if content == "" {
		return make(map[string]interface{}), nil
	}

	// 先尝试 JSON 解析
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err == nil {
		return data, nil
	}

	// 再尝试 YAML 解析
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &yamlData); err == nil {
		flat := make(map[string]interface{})
		flattenMap(yamlData, "", flat)
		return flat, nil
	}

	// 原始字符串作为 content key
	return map[string]interface{}{"content": content}, nil
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

// watchUpdates 监听 Apollo 配置变更
func (c *Config) watchUpdates() {
	updateChan := c.client.WatchUpdate()
	for change := range updateChan {
		if change.Namespace != c.namespace {
			continue
		}
		c.mu.Lock()
		for key, ch := range change.Changes {
			if ch.NewValue != nil {
				if parsed, err := parseConfig(string(ch.NewValue), change.Namespace); err == nil {
					// 合并到现有 data
					for k, v := range parsed {
						c.data[k] = v
					}
				} else {
					c.data[key] = string(ch.NewValue)
				}
			}
			if ch.ChangeType == apollo.DELETE {
				delete(c.data, key)
			}
		}
		c.mu.Unlock()

		// 通知所有 watcher
		for _, w := range c.watchers {
			w.callback(c.Get(w.key))
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
	return nil
}

// Close 关闭配置
func (c *Config) Close() error {
	c.watchers = nil
	_ = c.client.Stop()
	return nil
}

// ApolloConfig Apollo 配置选项
type ApolloConfig struct {
	AppID      string `yaml:"app_id"`
	Cluster    string `yaml:"cluster"`
	Namespace  string `yaml:"namespace"`
	SecretKey  string `yaml:"secret_key"`
	IP         string `yaml:"ip"`
	ServerAddr string `yaml:"server_addr"`
}
