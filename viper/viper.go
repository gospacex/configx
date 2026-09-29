// configx/viper/viper.go
// Viper 本地配置实现
package viper

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config Viper 配置实现
type Config struct {
	v        *viper.Viper
	filePath string
	watchers []watcher
}

type watcher struct {
	key      string
	callback func(newVal interface{})
}

// New 创建 Viper 配置实例
func New(cfg *ConfigOptions) (*Config, error) {
	v := viper.New()

	// 设置默认值
	v.SetConfigType("yaml")
	if cfg.ConfigType != "" {
		v.SetConfigType(cfg.ConfigType)
	}

	// 设置配置文件
	if cfg.ConfigFile != "" {
		v.SetConfigFile(cfg.ConfigFile)
	} else {
		// 搜索配置文件
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/myapp")

		if cfg.ConfigPath != "" {
			v.AddConfigPath(cfg.ConfigPath)
		}
	}

	// 环境变量支持
	v.SetEnvPrefix("HUBX")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
		// 配置文件不存在，使用默认值
	}

	return &Config{
		v:        v,
		filePath: v.ConfigFileUsed(),
	}, nil
}

// Get 获取配置值
func (c *Config) Get(key string) interface{} {
	return c.v.Get(key)
}

// GetString 获取字符串值
func (c *Config) GetString(key string) string {
	return c.v.GetString(key)
}

// GetInt 获取整数值
func (c *Config) GetInt(key string) int {
	return c.v.GetInt(key)
}

// GetInt64 获取 int64 值
func (c *Config) GetInt64(key string) int64 {
	return c.v.GetInt64(key)
}

// GetBool 获取布尔值
func (c *Config) GetBool(key string) bool {
	return c.v.GetBool(key)
}

// GetFloat64 获取 float64 值
func (c *Config) GetFloat64(key string) float64 {
	return c.v.GetFloat64(key)
}

// GetDuration 获取时间间隔
func (c *Config) GetDuration(key string) time.Duration {
	return c.v.GetDuration(key)
}

// GetSlice 获取字符串切片
func (c *Config) GetSlice(key string) []string {
	return c.v.GetStringSlice(key)
}

// GetMap 获取 map
func (c *Config) GetMap(key string) map[string]interface{} {
	return c.v.GetStringMap(key)
}

// Unmarshal 反序列化到结构体
func (c *Config) Unmarshal(key string, out interface{}) error {
	sub := c.v.Sub(key)
	if sub == nil {
		return fmt.Errorf("key %s not found", key)
	}
	return sub.Unmarshal(out)
}

// IsSet 判断 key 是否存在
func (c *Config) IsSet(key string) bool {
	return c.v.IsSet(key)
}

// Watch 监听配置变更
func (c *Config) Watch(ctx context.Context, key string, callback func(newVal interface{})) error {
	c.watchers = append(c.watchers, watcher{key: key, callback: callback})

	// 如果是文件，监听文件变更
	if c.filePath != "" {
		go c.watchFile(ctx)
	}

	return nil
}

// watchFile 监听文件变更
func (c *Config) watchFile(ctx context.Context) {
	// 简化的文件监听（生产环境应使用 fsnotify）
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var lastModTime time.Time
	if info, err := os.Stat(c.filePath); err == nil {
		lastModTime = info.ModTime()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := os.Stat(c.filePath)
			if err != nil {
				continue
			}

			if info.ModTime().After(lastModTime) {
				lastModTime = info.ModTime()

				// 重新加载配置
				if err := c.v.ReadInConfig(); err != nil {
					continue
				}

				// 通知所有 watcher
				for _, w := range c.watchers {
					if w.key == "" {
						w.callback(c.v.AllSettings())
					} else {
						w.callback(c.v.Get(w.key))
					}
				}
			}
		}
	}
}

// Close 关闭配置
func (c *Config) Close() error {
	c.watchers = nil
	return nil
}

// ConfigOptions 配置选项（与 configx.ConfigOptions 相同）
type ConfigOptions struct {
	ConfigFile string `yaml:"config_file"`
	ConfigType string `yaml:"config_type"`
	ConfigPath string `yaml:"config_path"`
}
