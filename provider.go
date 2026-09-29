// configx/provider.go
// 配置提供者：本地 Viper + 远程 Nacos/Apollo 合并
// 优先级：远程 > 本地 Viper > 环境变量 > 默认值
package configx

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	apolloimpl "github.com/gospacex/configx/apollo"
	nacosimpl "github.com/gospacex/configx/nacos"
	viperimpl "github.com/gospacex/configx/viper"
	"github.com/gospacex/traceflag"
)

// traceflagKey 全局调用链总开关键。
// 配置文件顶层的 traceflag.enabled 是各 x 系列 SDK 调用链开关的统一入口：
// 缺省不设置时保持 traceflag 缺省放行（true，opt-out）；显式设置时在
// configx 构造阶段自动映射到 traceflag.SetEnabled，一处熔断所有组件。
const traceflagKey = "traceflag.enabled"

// applyTraceflagGlobal 把配置里的 traceflag.enabled 应用到 traceflag 全局总闸。
// 未显式配置时不做任何覆盖，保持缺省行为。
func applyTraceflagGlobal(c Config) {
	if c.IsSet(traceflagKey) {
		traceflag.SetEnabled(c.GetBool(traceflagKey))
	}
}

// Provider 配置提供者（合并本地+远程）
type Provider struct {
	local  Config
	remote Config
}

// New 创建配置提供者
//
// 支持两种模式：
//  1. 程序化：通过 ConfigOptions.Remote.Enabled 显式开关
//  2. 配置驱动：ConfigOptions.Remote 未设置时，从本地配置文件的
//     configx.remote.enabled / configx.remote.<provider>.* 读取开关和连接信息
//
// 优先级：远程配置 > 本地配置文件 > 环境变量 > 默认值
func New(cfg *ConfigOptions) (Config, error) {
	if cfg == nil {
		cfg = &ConfigOptions{}
	}

	logger := slog.Default().With("component", "configx")

	// 1. 加载本地配置（viper）
	var local Config
	localCfg := &viperimpl.ConfigOptions{
		ConfigFile: cfg.ConfigFile,
		ConfigType: cfg.ConfigType,
		ConfigPath: cfg.ConfigPath,
	}
	lc, err := viperimpl.New(localCfg)
	if err != nil {
		logger.Warn("failed to load local config", "error", err)
		local = &emptyConfig{}
	} else {
		local = lc
		logger.Info("local config loaded", "file", cfg.ConfigFile)
	}

	// 2. 判断是否开启远程配置
	//    优先使用程序化开关，未设置时从本地配置文件读取
	remoteEnabled := cfg.Remote.Enabled
	remoteProvider := cfg.Remote.Provider
	remoteNacos := cfg.Remote.Nacos
	var remoteApollo *ApolloConfig
	if cfg.Remote.Apollo != nil {
		remoteApollo = &ApolloConfig{
			AppID:     cfg.Remote.Apollo.AppID,
			Cluster:   cfg.Remote.Apollo.Cluster,
			Namespace: cfg.Remote.Apollo.Namespace,
			IP:        cfg.Remote.Apollo.IP,
		}
	}

	if !remoteEnabled {
		// 从本地配置文件读取 configx.remote 开关
		if local.IsSet("configx.remote.enabled") {
			remoteEnabled = local.GetBool("configx.remote.enabled")
			if remoteEnabled {
				remoteProvider = local.GetString("configx.remote.provider")
				if remoteProvider == "" {
					remoteProvider = "nacos"
				}
				// 从本地配置文件读取对应远程配置
				switch remoteProvider {
				case "apollo":
					ac := readApolloConfigFromLocal(local)
					remoteApollo = &ac
				default:
					remoteNacos = readNacosConfigFromLocal(local)
				}
				logger.Info("remote config enabled from config file",
					"provider", remoteProvider,
				)
			}
		}
	}

	// 3. 如果开启远程配置，加载对应 provider
	if remoteEnabled {
		if remoteProvider == "" {
			remoteProvider = "nacos"
		}

		switch remoteProvider {
		case "nacos":
			remote, err := nacosimpl.New(toNacosConfig(&remoteNacos))
			if err != nil {
				return nil, fmt.Errorf("init nacos config: %w", err)
			}

			logger.Info("nacos config loaded",
				"namespace", remoteNacos.NamespaceID,
				"group", remoteNacos.Group,
				"data_id", remoteNacos.DataID,
			)

			// 应用全局调用链总开关（本地 → 远程，远程优先）
			applyTraceflagGlobal(local)
			applyTraceflagGlobal(remote)

			return &Provider{
				local:  local,
				remote: remote,
			}, nil

		case "apollo":
			if remoteApollo == nil {
				return nil, fmt.Errorf("apollo config is required")
			}
			remote, err := apolloimpl.New(&apolloimpl.ApolloConfig{
				AppID:     remoteApollo.AppID,
				Cluster:   remoteApollo.Cluster,
				Namespace: remoteApollo.Namespace,
				IP:        remoteApollo.IP,
			})
			if err != nil {
				return nil, fmt.Errorf("init apollo config: %w", err)
			}

			logger.Info("apollo config loaded",
				"app_id", remoteApollo.AppID,
				"namespace", remoteApollo.Namespace,
			)

			// 应用全局调用链总开关（本地 → 远程，远程优先）
			applyTraceflagGlobal(local)
			applyTraceflagGlobal(remote)

			return &Provider{
				local:  local,
				remote: remote,
			}, nil

		default:
			return nil, fmt.Errorf("unsupported remote config provider: %s", remoteProvider)
		}
	}

	// 4. 仅使用本地配置
	applyTraceflagGlobal(local)
	return local, nil
}

// readNacosConfigFromLocal 从本地配置读取 nacos 连接信息
func readNacosConfigFromLocal(local Config) NacosConfig {
	nc := NacosConfig{
		NamespaceID: local.GetString("configx.remote.nacos.namespace_id"),
		Group:       local.GetString("configx.remote.nacos.group"),
		DataID:      local.GetString("configx.remote.nacos.data_id"),
		Username:    local.GetString("configx.remote.nacos.username"),
		Password:    local.GetString("configx.remote.nacos.password"),
	}

	// 读取 server_configs 列表
	// 支持两种格式：
	//   - 简单：configx.remote.nacos.server_ip + server_port
	//   - 列表：configx.remote.nacos.server_configs[0].ip + port
	if local.IsSet("configx.remote.nacos.server_configs") {
		// 列表格式（企业集群）
		for i := 0; i < 10; i++ {
			prefix := fmt.Sprintf("configx.remote.nacos.server_configs.%d", i)
			if !local.IsSet(prefix + ".ip") {
				break
			}
			nc.ServerConfigs = append(nc.ServerConfigs, NacosServerConfig{
				IP:      local.GetString(prefix + ".ip"),
				Port:    uint64(local.GetInt(prefix + ".port")),
				Context: local.GetString(prefix + ".context"),
			})
		}
	}

	if len(nc.ServerConfigs) == 0 {
		// 简单格式（单机）
		ip := local.GetString("configx.remote.nacos.server_ip")
		port := local.GetInt("configx.remote.nacos.server_port")
		if ip != "" {
			nc.ServerConfigs = append(nc.ServerConfigs, NacosServerConfig{
				IP:   ip,
				Port: uint64(port),
			})
		}
	}

	return nc
}

// readApolloConfigFromLocal 从本地配置读取 Apollo 连接信息
func readApolloConfigFromLocal(local Config) ApolloConfig {
	ac := ApolloConfig{
		AppID:     local.GetString("configx.remote.apollo.app_id"),
		Cluster:   local.GetString("configx.remote.apollo.cluster"),
		Namespace: local.GetString("configx.remote.apollo.namespace"),
		IP:        local.GetString("configx.remote.apollo.ip"),
	}

	// 读取 server_configs 列表
	if local.IsSet("configx.remote.apollo.server_configs") {
		for i := 0; i < 10; i++ {
			prefix := fmt.Sprintf("configx.remote.apollo.server_configs.%d", i)
			if !local.IsSet(prefix + ".host") {
				break
			}
			// 多节点集群模式暂不支持，取第一个
			if i == 0 {
				host := local.GetString(prefix + ".host")
				port := local.GetInt(prefix + ".port")
				ac.IP = host + ":" + strconv.Itoa(port)
			}
		}
	}

	if ac.IP == "" {
		ac.IP = local.GetString("configx.remote.apollo.server_addr")
	}

	return ac
}

// toNacosConfig 将 configx.NacosConfig 转换为 nacos 包的 NacosConfig
func toNacosConfig(c *NacosConfig) *nacosimpl.NacosConfig {
	if c == nil {
		return &nacosimpl.NacosConfig{}
	}
	out := &nacosimpl.NacosConfig{
		NamespaceID: c.NamespaceID,
		Group:       c.Group,
		DataID:      c.DataID,
		Username:    c.Username,
		Password:    c.Password,
	}
	for _, sc := range c.ServerConfigs {
		out.ServerConfigs = append(out.ServerConfigs, nacosimpl.NacosServerConfig{
			IP:      sc.IP,
			Port:    sc.Port,
			Context: sc.Context,
		})
	}
	return out
}

// ========== Config 接口实现（远程优先）==========

func (p *Provider) Get(key string) interface{} {
	if p.remote != nil && p.remote.IsSet(key) {
		return p.remote.Get(key)
	}
	if p.local != nil {
		return p.local.Get(key)
	}
	return nil
}

func (p *Provider) GetString(key string) string {
	if p.remote != nil && p.remote.IsSet(key) {
		return p.remote.GetString(key)
	}
	if p.local != nil {
		return p.local.GetString(key)
	}
	return ""
}

func (p *Provider) GetInt(key string) int {
	if p.remote != nil && p.remote.IsSet(key) {
		return p.remote.GetInt(key)
	}
	if p.local != nil {
		return p.local.GetInt(key)
	}
	return 0
}

func (p *Provider) GetInt64(key string) int64 {
	if p.remote != nil && p.remote.IsSet(key) {
		return p.remote.GetInt64(key)
	}
	if p.local != nil {
		return p.local.GetInt64(key)
	}
	return 0
}

func (p *Provider) GetBool(key string) bool {
	if p.remote != nil && p.remote.IsSet(key) {
		return p.remote.GetBool(key)
	}
	if p.local != nil {
		return p.local.GetBool(key)
	}
	return false
}

func (p *Provider) GetFloat64(key string) float64 {
	if p.remote != nil && p.remote.IsSet(key) {
		return p.remote.GetFloat64(key)
	}
	if p.local != nil {
		return p.local.GetFloat64(key)
	}
	return 0
}

func (p *Provider) GetDuration(key string) time.Duration {
	if p.remote != nil && p.remote.IsSet(key) {
		return p.remote.GetDuration(key)
	}
	if p.local != nil {
		return p.local.GetDuration(key)
	}
	return 0
}

func (p *Provider) GetSlice(key string) []string {
	if p.remote != nil && p.remote.IsSet(key) {
		return p.remote.GetSlice(key)
	}
	if p.local != nil {
		return p.local.GetSlice(key)
	}
	return nil
}

func (p *Provider) GetMap(key string) map[string]interface{} {
	if p.remote != nil && p.remote.IsSet(key) {
		return p.remote.GetMap(key)
	}
	if p.local != nil {
		return p.local.GetMap(key)
	}
	return nil
}

func (p *Provider) Unmarshal(key string, out interface{}) error {
	if p.remote != nil && p.remote.IsSet(key) {
		return p.remote.Unmarshal(key, out)
	}
	if p.local != nil {
		return p.local.Unmarshal(key, out)
	}
	return fmt.Errorf("key %s not found", key)
}

func (p *Provider) IsSet(key string) bool {
	if p.remote != nil && p.remote.IsSet(key) {
		return true
	}
	if p.local != nil {
		return p.local.IsSet(key)
	}
	return false
}

func (p *Provider) Watch(ctx context.Context, key string, callback func(newVal interface{})) error {
	if p.remote != nil {
		return p.remote.Watch(ctx, key, callback)
	}
	if p.local != nil {
		return p.local.Watch(ctx, key, callback)
	}
	return nil
}

func (p *Provider) Close() error {
	if p.remote != nil {
		_ = p.remote.Close()
	}
	if p.local != nil {
		_ = p.local.Close()
	}
	return nil
}

// emptyConfig 空配置（当本地配置加载失败时使用）
type emptyConfig struct{}

func (c *emptyConfig) Get(key string) interface{}               { return nil }
func (c *emptyConfig) GetString(key string) string              { return "" }
func (c *emptyConfig) GetInt(key string) int                    { return 0 }
func (c *emptyConfig) GetInt64(key string) int64                { return 0 }
func (c *emptyConfig) GetBool(key string) bool                  { return false }
func (c *emptyConfig) GetFloat64(key string) float64            { return 0 }
func (c *emptyConfig) GetDuration(key string) time.Duration     { return 0 }
func (c *emptyConfig) GetSlice(key string) []string             { return nil }
func (c *emptyConfig) GetMap(key string) map[string]interface{} { return nil }
func (c *emptyConfig) Unmarshal(key string, out interface{}) error {
	return fmt.Errorf("key %s not found", key)
}
func (c *emptyConfig) IsSet(key string) bool { return false }
func (c *emptyConfig) Watch(ctx context.Context, key string, callback func(newVal interface{})) error {
	return nil
}
func (c *emptyConfig) Close() error { return nil }
