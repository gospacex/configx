<div align="center">

# configx

**HubX 全系 x 系列 SDK 的统一配置基座 —— 一个 `configx.Config` 句柄，管住本地文件与远程配置中心**

The unified configuration foundation for every HubX x-series SDK: one `configx.Config` handle over local files and remote config centers.

[![Go](https://img.shields.io/badge/Go-1.21-00ADD8?logo=go&logoColor=white)](https://go.dev/)
![License](https://img.shields.io/badge/License-MIT-green.svg)
[![Release](https://img.shields.io/badge/release-v0.0.1-blue.svg)](https://github.com/gospacex/configx/tags)

</div>

---

## 目录

- [1. 定位与设计边界](#1-定位与设计边界)
- [2. 核心特性](#2-核心特性)
- [3. 架构](#3-架构)
- [4. 在 hubx 生态中的角色](#4-在-hubx-生态中的角色)
- [5. 快速开始](#5-快速开始)
- [6. 配置参考](#6-配置参考)
- [7. API 参考](#7-api-参考)
- [8. 后端矩阵](#8-后端矩阵)
- [9. 可观测性](#9-可观测性)
- [10. 示例与文档](#10-示例与文档)
- [11. 测试与质量](#11-测试与质量)
- [12. 已知限制](#12-已知限制)
- [13. 版本与兼容性](#13-版本与兼容性)
- [14. 许可证](#14-许可证)

---

## 1. 定位与设计边界

hubx 的每个 x 系列 SDK（dbx、cachex、mqx、logx、searchx…）都要读同一棵配置树，
而生产环境又要求这些 key 能被 Nacos / Apollo 远程覆盖。configx 把这两件事收敛成一件事：

**对外只暴露一个 `configx.Config` 接口和两个入口函数（`New` / `ReadConfig`），
内部按需拼装「本地 Viper + 远程配置中心」，并按 key 做远程优先的合并读取。**

### 它做什么

| 能力 | 说明 |
| :--- | :--- |
| 统一配置句柄 | 13 个方法的 `Config` 接口（读取 / 反序列化 / 监听 / 关闭），全系 SDK 共用同一类型 |
| 本地文件加载 | 默认由 `configx/viper` 加载配置文件（默认 YAML，可传 JSON / TOML），支持搜索路径与 `HUBX_*` 环境变量 |
| 远程配置中心 | `configx/nacos`、`configx/apollo` 两个后端，可由程序或配置文件开启 |
| 远程优先合并 | `Provider` 逐 key 判断：远程已存在则取远程，否则回落本地 |
| 变更监听 | `Watch(ctx, key, cb)`：Nacos 走 `ListenConfig`，Apollo 走 `WatchUpdate`，本地文件走轮询 |

### 它不做什么

- **不做配置校验 / schema / 类型系统**：读哪些 key、什么类型，由各 SDK 自己负责。
- **不做 `${VAR}` 占位符展开**：源码中没有任何展开逻辑，配置里的 `${MYSQL_DSN}`
  会被当作字面量返回（见 [§12 已知限制](#12-已知限制)）。
- **不做配置中心服务端**：`docker-compose.yml` 只是本地开发依赖。
- **不提供加密、灰度、权限、发布审批**：这些属于配置中心自身的职责。
- **不做跨进程同步**：句柄只在当前进程内生效，`Watch` 只在配置中心推送/文件变更时触发。

### 与同系列 SDK 的关系

- `logx`：把 `configx.Config` 存为全局配置源（`logx.ReadConfig` → `configx.ReadConfig`，`logx.GetConfig()` 取回）。
- `dbx` / `cachex` / `mqx` / `searchx` / `coordx`：把 `configx.Config` 作为函数入参，
  从中读取 `xmiddleware.<sdk>.*` 配置树。
- 配置内容本身（`app.*`、`logx.*`、`xmiddleware.*`）由各 SDK 定义；configx 只负责**取值**。

---

## 2. 核心特性

- **统一接口**：`configx.Config` 覆盖 `Get*` / `Unmarshal` / `IsSet` / `Watch` / `Close`，
  消费方不感知背后是文件、Nacos 还是 Apollo。
- **三后端一套句柄**：本地文件（Viper）、Nacos、Apollo 共用同一接口（[§8 后端矩阵](#8-后端矩阵)）。
- **远程优先合并**：`Provider` 对每个 key 先问远程 `IsSet`，未命中再回落本地，
  因此「本地兜底 + 远程覆盖」天然成立。
- **两种开启方式**：程序化传 `ConfigOptions.Remote`，或在配置文件写 `configx.remote.enabled`，
  后者可让同一份二进制在开发/生产之间零改码切换。
- **配置文件自动发现**：显式 `ConfigFile` > `HUBX_CONFIG` 环境变量 > `./config.yaml`。
- **变更监听**：`Watch` 支持监听单个 key 或全部变更（`key` 传空字符串）。
- **单向依赖**：消费方只 import `github.com/gospacex/configx`，不需要接触子包。

---

## 3. 架构

```
业务代码 / x 系列 SDK（dbx、cachex、mqx、logx、searchx、coordx …）
        │
        │   只面向 configx.Config 接口编程
        ▼
┌──────────────────────── configx ─────────────────────────┐
│  入口                                                     │
│   ReadConfig(opts ...ConfigOptions) (Config, error)       │
│   New(cfg *ConfigOptions) (Config, error)                 │
│                                                          │
│  远程未开启 ───────────────► 直接返回 viper 实现           │
│  远程已开启 ──► Provider{ remote, local }   逐 key 择取    │
│                 ├─ remote: nacos.Config │ apollo.Config   │
│                 └─ local : viper.Config                   │
└───────────────────────────────────────────────────────────┘
        ▲                     ▲                     ▲
  configx/viper         configx/nacos          configx/apollo
  spf13/viper           nacos-sdk-go/v2        gopkg.in/apollo.v0
  本地配置文件           Nacos 配置中心          Apollo 配置中心
```

读取顺序（`Provider` 逐 key 执行，见 `provider.go`）：

```text
远程 Provider.IsSet(key) 命中 → 返回远程值
否则                          → 返回本地 Viper 值
本地 Viper 层内（Viper 自身优先级）：
显式 Set > 环境变量(HUBX_*) > 配置文件 > 默认值
```

---

## 4. 在 hubx 生态中的角色

configx 是 hubx 全系的配置入口。下列关系均可在对应模块源码中核验
（本目录之外的模块通过 `replace github.com/gospacex/configx => ../configx` 指向本模块）。

### 4.1 统一入口三件套

除 configx 自身外，每个 x 系列 SDK 都暴露同一套配置入口，**名字与语义逐字一致**，
只有返回类型 `T` 随组件而异：

| 函数 | 签名 | 语义 |
| :--- | :--- | :--- |
| `ReadConfig` | `func ReadConfig(opts ...configx.ConfigOptions) error` | 只解析配置、不建实例。用于参数预检，或"先验证配置可用再决定是否初始化" |
| `ReadAndCreate` | `func ReadAndCreate(opts ...configx.ConfigOptions) (T, error)` | 读配置 → `Unmarshal` 本组件的段 → 建实例 |
| `MustReadAndCreate` | `func MustReadAndCreate(opts ...configx.ConfigOptions) T` | 同上，失败写 stderr 并 `os.Exit(1)` |

配置来源一律交由 `configx.ReadConfig` 解析，因此显式路径 / `HUBX_CONFIG` /
默认路径三条通路对所有 SDK 一致，切换远程配置中心只需改配置文件的
`configx.remote.*`，业务代码零改动。

需要打开**非默认实例**（如 `xmiddleware.sqlx.databases.replica`）时，用组件的
`ReadConfigFromConfigx(cfg, name)` + `New/Open` 组合，见下表"备注"列。

### 4.2 各组件实现（2026-09-28 统一，源码可核）

| 模块 | `T` | 配置段 | 备注 |
| :--- | :--- | :--- | :--- |
| `cachex` | `Instances` | `cachex` | 段内还可带 `tracing` 并 `Watch` 热更新开关 |
| `coordx` | `Register, Discover, Lease` | `coordx` | 唯一三返回值的组件：三件套齐备但 `T` 是三元组 |
| `kvx` | `Stores` | `kvx` | |
| `lockx` | `Locker` | `lockx` | `cluster` 优先于 `standalone` |
| `logx` | `Logger` | `logx` | 段在**顶层** `logx:`（不在 `xmiddleware` 下）；`ReadAndCreate` 内部复用本包 `ReadConfig`，故 `GetConfig()` 同样可用 |
| `mongodbx` | `Instances` | `mongodbx` | 段名与企业配置的 `docx:` **不一致**（见 §4.3） |
| `mqx` | `*SDK` | `xmiddleware.mqx.instances.<name>` | 单例语义：重复调用返回同一句柄；`Init` 亦已改走 `configx.ReadConfig` |
| `objectx` | `BucketClient` | `objectx` | |
| `otelx` | `Provider` | `otelx` | 装配 TracerProvider/Sampler/Exporter；只 `ReadConfig` 等于什么也没做 |
| `searchx` | `SearchClient` | `searchx` | |
| `vectorx` | `CollectionClient` | `vectorx.stores.<name>` | `name` 是**前置参数**：`ReadAndCreate(name string, opts ...)` |
| `webx` | `*App` | `webx` | `Config` 含函数/接口成员无法直接反序列化，故用可序列化的 `WebSection` 中转；注入中间件用 `ReadConfigFromConfigx` + `New(WithConfig...)` |
| `dbx`（`dbsql` 包） | `*gorm.DB` | `xmiddleware.sqlx.databases.<name>` | `ReadAndCreate` 固定打开 `default`；其它实例用 `ReadConfigFromConfigx` + `OpenFromConfigx` |

### 4.3 取段路径目前有两套（已知不一致）

| 取段方式 | 组件 | 与官方配置的关系 |
| :--- | :--- | :--- |
| **顶层段** `Unmarshal("<sdk>")` | cachex、coordx、kvx、lockx、logx、mongodbx、objectx、otelx、searchx、vectorx、webx | 与各组件自带的示例配置一致；**读不到** `configx/configs/config.enterprise.yaml`（那里所有段都在 `xmiddleware` 下） |
| **嵌套段** `xmiddleware.<sdk>.*` | dbx、mqx | 与官方企业配置一致；读不到只写了顶层段的配置 |

`monolith/README.md` 也记录了这一现象（它因此绕开了 `cachex.ReadAndCreate`）。
两者的收敛需要先定"哪套是合同"，属于破坏性变更，本轮未动。

> **远程配置下的额外限制**：Nacos 通道在配置内容是 YAML 时会把嵌套结构**展平**
> 成点分隔的叶子键（`configx/nacos/nacos.go` 的 `flattenMap`），此时段级键
> `xmiddleware.cachex` 查不到，只有叶子键可读；JSON 格式的配置不受影响。
> 因此"段级 `Unmarshal`"目前只在本地 viper 与 Nacos-JSON 下可靠。

**消费约定**：下游拿到的是 `configx.Config` 接口，不感知后端；
因此「切换本地/配置中心」只改配置文件的 `configx.remote.*`，业务代码零改动。

> **依赖声明注意**：`replace` 不会传递给下游 Module。
> 任何直接或间接依赖本模块的项目，都要在自己的 `go.mod` 中重复声明 replace（当前为未发布状态）。

---

## 5. 快速开始

### 5.1 安装

```bash
go get github.com/gospacex/configx
```

hubx 单仓内使用时，兄弟模块以本地路径引用：

```go
// go.mod
replace github.com/gospacex/configx => ../configx
```

### 5.2 路径一：本地文件（最小可用）

```go
package main

import (
	"fmt"
	"log"

	"github.com/gospacex/configx"
)

func main() {
	cfg, err := configx.New(&configx.ConfigOptions{
		ConfigFile: "configs/config.yaml",
		ConfigType: "yaml",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cfg.Close()

	fmt.Println("app:", cfg.GetString("app.name"))
	fmt.Println("dsn:", cfg.GetString("xmiddleware.sqlx.databases.default.dsn"))
}
```

未设置 `Remote`、且配置文件中也没有 `configx.remote.enabled: true` 时，
`New` 只加载本地文件并返回 Viper 实现，不产生任何网络连接。

### 5.3 路径二：配置中心 —— Nacos

```go
cfg, err := configx.New(&configx.ConfigOptions{
	ConfigFile: "configs/config.yaml",
	Remote: configx.RemoteConfig{
		Enabled:  true,
		Provider: "nacos",
		Nacos: configx.NacosConfig{
			ServerConfigs: []configx.NacosServerConfig{{IP: "127.0.0.1", Port: 8848}},
			Group:         "DEFAULT_GROUP",
			DataID:        "myapp",
		},
	},
})
if err != nil {
	log.Fatal(err)
}
defer cfg.Close()

// Nacos 中存在的 key 优先返回；Nacos 没有的 key 自动回落本地文件
fmt.Println(cfg.GetString("app.name"))
```

不写 `Remote`，改为在配置文件里开启也可以（同一份代码切换环境）：

```yaml
configx:
  remote:
    enabled: true
    provider: nacos
    nacos:
      server_configs:
        - ip: 127.0.0.1
          port: 8848
      namespace_id: ""
      group: "DEFAULT_GROUP"
      data_id: "myapp"
```

### 5.4 路径二：配置中心 —— Apollo

```go
cfg, err := configx.New(&configx.ConfigOptions{
	ConfigFile: "configs/config.yaml",
	Remote: configx.RemoteConfig{
		Enabled:  true,
		Provider: "apollo",
		Apollo: &configx.ApolloConfig{
			AppID:     "myapp",
			Cluster:   "default",
			Namespace: "application",
			IP:        "127.0.0.1:8085", // config service 的 host:port
		},
	},
})
if err != nil {
	log.Fatal(err)
}
defer cfg.Close()
```

`Apollo` 必须是非 nil 指针，否则 `New` 返回 `apollo config is required`；
`app_id` 与 `ip` 缺省会分别报 `apollo app_id is required` / `apollo ip is required`。

### 5.5 用 ReadConfig 免记查找顺序

```go
cfg, err := configx.ReadConfig(configx.ConfigOptions{ConfigFile: "./config.yaml"})
if err != nil {
	log.Fatal(err)
}
defer cfg.Close()

fmt.Println(cfg.GetString("app.name"))
```

`ReadConfig` 与 `New` 的差别只有一点：`ConfigFile` 为空时依次尝试
`HUBX_CONFIG` 环境变量、`./config.yaml`；远程开关的解析两种入口完全一致（见 [§6.2](#62-配置文件键configxremote)）。

### 5.6 监听变更与反序列化

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// key 传空字符串表示监听所有变更
if err := cfg.Watch(ctx, "logx.level", func(newVal interface{}) {
	log.Printf("logx.level -> %v", newVal)
}); err != nil {
	log.Fatal(err)
}

// 把某个子树反序列化到结构体
var section struct {
	Driver string `yaml:"driver"`
}
if err := cfg.Unmarshal("xmiddleware.sqlx", &section); err != nil {
	log.Fatal(err)
}
```

---

## 6. 配置参考

### 6.1 配置文件发现

| 优先级 | 来源 | 生效条件 |
| :--- | :--- | :--- |
| 1 | `ConfigOptions.ConfigFile` 显式指定 | 总是 |
| 2 | 环境变量 `HUBX_CONFIG` | `ConfigFile` 为空时（`ReadConfig` 路径） |
| 3 | `./config.yaml` | `ConfigFile` 为空时（`ReadConfig` 路径） |
| 4 | 搜索路径 `.` / `./configs` / `/etc/myapp` 再加 `ConfigPath` | 调用 `New` 且 `ConfigFile` 为空（此时按文件名 `config` 搜索） |

文件不存在不算错误：Viper 返回 `ConfigFileNotFoundError` 时按「只有默认值」处理。
`ConfigType` 缺省为 `yaml`（可传 `json`、`toml` 等 Viper 支持的格式）。

### 6.2 配置文件键（`configx.remote.*`）

以下键名逐一取自 `configx.go` 的 struct tag 与 `provider.go` 的读取逻辑。

```yaml
configx:
  remote:
    enabled: false              # 总开关；false → 只用本地文件
    provider: nacos             # nacos | apollo；开启但为空时按 nacos 处理
    nacos:
      server_configs:           # 列表格式（集群），按索引 0..9 顺序读取
        - ip: 127.0.0.1
          port: 8848
          context: ""           # Nacos ContextPath
      # server_ip: 127.0.0.1    # 单机简写：server_configs 为空时才读取
      # server_port: 8848
      namespace_id: ""
      group: "DEFAULT_GROUP"
      data_id: "myapp"
      username: ""
      password: ""
      cluster_name: ""          # 已声明，当前未传给 Nacos SDK（无效）
      refresh_interval: 30s     # 已声明，当前未传给 Nacos SDK（无效）
    apollo:
      app_id: myapp             # 必填
      cluster: default
      namespace: application
      ip: "127.0.0.1:8085"      # host:port，直接请求 http://IP/configs/...
      # server_addr: ""         # 与 ip 二选一：文件路径下会被写入 ip
      server_configs:           # 只取第一个节点，转成 host:port 写入 ip
        - host: 127.0.0.1
          port: 8085
      secret_key: ""            # 已声明，当前未接入 apollo 客户端（无效）
```

| 键 | 类型 | 必填 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- |
| `configx.remote.enabled` | bool | 否 | `false` | 程序化 `Remote.Enabled` 为 true 时不读取该键 |
| `configx.remote.provider` | string | 否 | `nacos` | 其它取值报 `unsupported remote config provider` |
| `…nacos.server_configs[].ip` | string | 是 | — | 与 `port` 搭配，最多读取 10 个节点 |
| `…nacos.server_configs[].port` | int | 是 | — | — |
| `…nacos.server_configs[].context` | string | 否 | 空 | 透传给 Nacos SDK 的 `ContextPath` |
| `…nacos.server_ip` / `…nacos.server_port` | string / int | 否 | — | 单机简写，仅在 `server_configs` 为空时读取 |
| `…nacos.namespace_id` | string | 否 | 空 | 命名空间 ID |
| `…nacos.group` | string | 否 | 无 | 结构体注释写 `DEFAULT_GROUP`，但代码**不填充**默认值，建议显式配置 |
| `…nacos.data_id` | string | 是 | — | 远程配置的 dataId |
| `…nacos.username` / `…nacos.password` | string | 否 | 空 | Nacos 鉴权 |
| `…nacos.cluster_name` / `…nacos.refresh_interval` | string / duration | 否 | — | 仅有字段声明，`toNacosConfig` 未透传，当前不生效 |
| `…apollo.app_id` | string | 是 | — | 缺失报 `apollo app_id is required` |
| `…apollo.cluster` | string | 否 | 空 | 透传给 Apollo 客户端 |
| `…apollo.namespace` | string | 否 | 空 | 取初始内容时把空值按 `application` 处理 |
| `…apollo.ip` | string | 是 | — | 缺失报 `apollo ip is required`；`server_addr` / `server_configs[0]` 会写入该值 |
| `…apollo.server_addr` | string | 否 | — | 仅文件路径生效（写入 `ip`）；程序化传入时会被丢弃 |
| `…apollo.server_configs[].host` / `.port` | string / int | 否 | — | 仅取第一个节点 |
| `…apollo.secret_key` | string | 否 | — | 仅有字段声明，未接入客户端，不生效 |

### 6.3 环境变量

| 变量 | 读取位置 | 作用 |
| :--- | :--- | :--- |
| `HUBX_CONFIG` | `configx.ReadConfig` | 配置文件路径（`ConfigFile` 为空时） |
| `HUBX_*` | `configx/viper` | Viper `SetEnvPrefix("HUBX")` + `AutomaticEnv()` + `.`→`_` 替换，例如 `app.name` → `HUBX_APP_NAME` |

> 结构体上的 `yaml` / `json` / `env` tag（如 `env:"REMOTE_ENABLED"`、`env:"APOLLO_APP_ID"`）
> 在本模块内**没有**对应的解析代码，属声明性标记；远程开关只认 `configx.remote.*` 或
> `ConfigOptions.Remote`。同一 key 同时存在时，Viper 内 `HUBX_*` 环境变量优先于配置文件
> （Viper 官方优先级：env > config）。

### 6.4 完整示例

```yaml
app:
  name: myapp
  version: 1.0.0
  env: development

configx:
  remote:
    enabled: true
    provider: nacos
    nacos:
      server_configs:
        - ip: 127.0.0.1
          port: 8848
      namespace_id: ""
      group: "DEFAULT_GROUP"
      data_id: "myapp"

logx:
  level: debug
  format: console
  output: stdout

xmiddleware:
  sqlx:
    driver: gorm
    databases:
      default:
        dsn: "root:rootpass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
        max_open_conns: 100
        max_idle_conns: 25
  cachex:
    driver: redis
    instances:
      default:
        addr: "localhost:6379"
        password: "redis123456"
        db: 0
```

更完整的样例（含 `mqx` / `vectorx` / `protocolx` 等全配置树）见：

- `configs/config.simple.yaml` —— 单机开发，`configx.remote.enabled: false`
- `configs/config.enterprise.yaml` —— 生产集群，`configx.remote.enabled: true`

> `configx.remote.*` 之外的所有键（`app.*`、`logx.*`、`xmiddleware.*`）只是**被读取的数据**，
> 由各 x 系列 SDK 自行解释，configx 不校验其结构。

---

## 7. API 参考

### 7.1 核心接口

```go
type Config interface {
	Get(key string) interface{}
	GetString(key string) string
	GetInt(key string) int
	GetInt64(key string) int64
	GetBool(key string) bool
	GetFloat64(key string) float64
	GetDuration(key string) time.Duration
	GetSlice(key string) []string
	GetMap(key string) map[string]interface{}
	Unmarshal(key string, out interface{}) error
	IsSet(key string) bool

	Watch(ctx context.Context, key string, callback func(newVal interface{})) error
	Close() error
}
```

| 方法 | 返回 | 说明 |
| :--- | :--- | :--- |
| `Get` | `interface{}` | 原始值，未命中返回 `nil` |
| `GetString` / `GetInt` / `GetInt64` / `GetBool` / `GetFloat64` / `GetDuration` | 对应类型 | 类型转换失败或未命中返回零值 |
| `GetSlice` | `[]string` | 远程实现会把 YAML 数组转成字符串切片 |
| `GetMap` | `map[string]interface{}` | 未命中返回 `nil` |
| `Unmarshal` | `error` | 把 key 下的子树反序列化到结构体；key 不存在返回 `key %s not found` |
| `IsSet` | `bool` | 远程实现以「取值非 nil」判定 |
| `Watch` | `error` | `key` 为空监听全部变更；回调收到 `interface{}` 新值 |
| `Close` | `error` | 清空 watcher 列表；Apollo 实现额外调用客户端 `Stop()`（Nacos 实现没有显式关闭客户端） |

### 7.2 包级函数

```go
func New(cfg *ConfigOptions) (Config, error)
func ReadConfig(opts ...ConfigOptions) (Config, error)
```

| 函数 | 说明 |
| :--- | :--- |
| `New` | 主入口。`cfg` 为 nil 时按零值处理；本地加载失败降级为空配置并 `Warn`，远程连接失败直接返回 error |
| `ReadConfig` | `New` 的便捷封装：补全 `ConfigFile`（`ConfigFile` → `HUBX_CONFIG` → `config.yaml`），错误统一带 `configx: ` 前缀 |

### 7.3 配置结构体（根包 `github.com/gospacex/configx`）

```go
type ConfigOptions struct {
	ConfigFile string       // yaml:"config_file" —— 配置文件路径
	ConfigType string       // yaml:"config_type" —— yaml, json, toml
	ConfigPath string       // yaml:"config_path" —— 配置搜索路径
	Remote     RemoteConfig // yaml:"remote"      —— 远程配置开关与连接信息
}

type RemoteConfig struct {
	Enabled bool          `yaml:"enabled"`  // 默认 false → 本地 viper
	Provider string       `yaml:"provider"` // nacos | apollo
	Nacos   NacosConfig   `yaml:"nacos"`
	Apollo  *ApolloConfig `yaml:"apollo"`   // apollo 时必须非 nil
}

type NacosConfig struct {
	ServerConfigs   []NacosServerConfig `yaml:"server_configs"`
	NamespaceID     string              `yaml:"namespace_id"`
	Group           string              `yaml:"group"`
	DataID          string              `yaml:"data_id"`
	ClusterName     string              `yaml:"cluster_name"`     // 未生效
	Username        string              `yaml:"username"`
	Password        string              `yaml:"password"`
	RefreshInterval time.Duration       `yaml:"refresh_interval"` // 未生效
}

type NacosServerConfig struct {
	IP      string `yaml:"ip"`
	Port    uint64 `yaml:"port"`
	Context string `yaml:"context"`
}

type ApolloConfig struct {
	AppID         string              `yaml:"app_id"`
	Cluster       string              `yaml:"cluster"`
	Namespace     string              `yaml:"namespace"`
	SecretKey     string              `yaml:"secret_key"`   // 未生效
	IP            string              `yaml:"ip"`
	ServerConfigs []ApolloServerConfig `yaml:"server_configs"`
	ServerAddr    string              `yaml:"server_addr"`  // 仅文件路径生效
}

type ApolloServerConfig struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

type Provider struct { /* 字段未导出 */ }
```

- `ConfigOptions` 是纯 Go 入参结构，**不会**被 YAML 反序列化；配置文件里对应的是 `configx.remote.*`。
- `Provider` 是 `New` 在开启远程时返回的具体实现（以 `Config` 接口呈现），
  字段未导出，外部无法构造或替换内部后端。
- 为便于阅读，上文省略了 `json:` tag；字段名与 tag 一律以 `configx.go` 源码为准。

### 7.4 子包

| 包 | 导出符号 | 用途 |
| :--- | :--- | :--- |
| `github.com/gospacex/configx/viper` | `Config`、`ConfigOptions`、`New(*ConfigOptions) (*Config, error)` | 本地文件实现 |
| `github.com/gospacex/configx/nacos` | `Config`、`NacosConfig`、`NacosServerConfig`、`New(*NacosConfig) (*Config, error)` | Nacos 实现 |
| `github.com/gospacex/configx/apollo` | `Config`、`ApolloConfig`、`New(*ApolloConfig) (*Config, error)` | Apollo 实现 |

> 子包可单独使用，但 hubx 生态内的常规做法是只依赖根包的 `New` / `ReadConfig`。
> 注意子包类型与根包同名类型**不通用**（如 `configx.NacosConfig` ≠ `nacos.NacosConfig`），
> `provider.go` 中的 `toNacosConfig` 负责两者转换。

---

## 8. 后端矩阵

| 后端 | 实现状态 | 包 | 初始加载 | 变更监听 | 内容格式 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 本地文件 | ✅ 完整 | `configx/viper` | Viper 读文件（不存在则用默认值） | `Watch` 后启动 10s 轮询 mtime，变更时重读并回调 | YAML / JSON / TOML 等 Viper 支持的格式 |
| Nacos | ✅ 完整 | `configx/nacos` | `IConfigClient.GetConfig` 拉取全量内容 | `ListenConfig` 推送回调，重解析后逐 watcher 通知 | JSON 优先，其次 YAML（YAML 展平为点分隔 key） |
| Apollo | ✅ 可用（字段接入不全） | `configx/apollo` | `GetNameSpaceContent` 拉取初始内容 | `WatchUpdate()` 增量合并（DELETE 会移除 key） | JSON 优先，其次 YAML，最后回落为 `content` 单键 |
| 其它（etcd / consul / …） | ❌ 未实现 | — | — | — | `Provider` 为其它取值时返回 `unsupported remote config provider` |

补充事实：

- `nacos.New` 要求 `ServerConfigs` 非空，否则返回 `nacos server configs is required`。
- 远程连接成功后返回的是 `*Provider`（远程 + 本地），远程 key 缺失时自动回落本地。
- Apollo 客户端 `Start()` 失败（含库内部 panic）会被 recover 并转成 error 返回；
  内容解析不出 JSON / YAML 时回落为 `content` 单键，**不会**导致 `New` 失败。

---

## 9. 可观测性

configx 自身的可观测能力很薄，只有一层结构化日志：

| 事件 | 级别 | 字段 |
| :--- | :--- | :--- |
| `local config loaded` | INFO | `file` |
| `failed to load local config` | WARN | `error` |
| `remote config enabled from config file` | INFO | `provider` |
| `nacos config loaded` | INFO | `namespace` `group` `data_id` |
| `apollo config loaded` | INFO | `app_id` `namespace` |

- Logger 为 `slog.Default().With("component", "configx")`，
  因此可通过 `slog.SetDefault(...)` 全局接管（例如交给 `logx`）。
- **不产生** metrics 与 trace span；需要指标/链路时由消费方（如 `otelx`）自行埋点。
- `Watch` 回调里的日志由调用方负责，configx 不打印回调内容。

---

## 10. 示例与文档

| 路径 | 内容 |
| :--- | :--- |
| `configs/config.simple.yaml` | 单机开发配置（280 行），远程开关关闭 |
| `configs/config.enterprise.yaml` | 生产集群配置（350 行），远程开关开启、三节点 Nacos |
| `example/main.go` | 端到端演示：发布 Nacos 配置 → 对比本地/混合读取 → 打印优先级表并自校验 |
| `example/config.yaml` | example 使用的本地配置（含 `configx.remote.*` 注释） |
| `example/docker-compose.yml` | 仅 MySQL + Nacos（带 `example/data/mysql/init/nacos-schema.sql`） |
| `docker-compose.yml` | MySQL + Nacos + Apollo 三件套（本地配置中心全家桶） |
| `tools/upload.go` | 向 Nacos 发布并回读校验配置的命令行小工具 |

```bash
# 启动本地配置中心（mysql + nacos + apollo）
docker compose up -d

# 只启动 Nacos（跑集成测试所需的最小依赖）
docker compose up -d nacos

open http://localhost:8848/nacos    # Nacos 控制台
open http://localhost:8070          # Apollo Portal（config service 映射在 8085）

# 把配置发布到 Nacos 并回读校验
go run ./tools myapp DEFAULT_GROUP configs/config.simple.yaml

# 运行优先级演示（需先启动 Nacos）
go run ./example
```

`example/main.go` 的验证点：Nacos 中存在的 `app.name` / `logx.level` 覆盖本地，
Nacos 中不存在的 `xmiddleware.cachex.*` 回落本地。

---

## 11. 测试与质量

```bash
go build ./...          # 编译全部包
go vet ./...            # 静态检查
go test -short ./...    # 单元测试（跳过需要本地 Nacos 的集成测试）
go test ./...           # 全量：需先 docker compose up -d nacos
```

| 测试文件 | 类型 | 说明 |
| :--- | :--- | :--- |
| `configx_test.go` | 单元 | `Provider` 的 New / Get / IsSet / Unmarshal / Watch |
| `provider_test.go` | 单元 | 程序化与配置文件两种远程开关、Apollo 参数解析、不支持的 provider |
| `nacos_test.go` | 集成 | 依赖 `http://localhost:8848`；`testing.Short()` 下跳过，**未启动 Nacos 时会失败而非跳过** |
| `apollo/apollo_test.go` | 单元 | JSON/YAML 解析、展平、必填参数校验 |

`viper/`、`nacos/`、`example/`、`tools/` 目前没有独立测试文件。

---

## 12. 已知限制

1. **本地文件热更新是轮询**：`configx/viper` 用 10s ticker 检查 mtime，
   且只有调用 `Watch` 后才启动该 goroutine；未调用 `Watch` 时，句柄读到的仍是首次加载的值。
2. **开启远程后本地监听被旁路**：`Provider.Watch` 只把订阅转给远程实现，
   本地文件的 `Watch` 在远程模式下不再生效。
3. **部分字段只声明不生效**：`NacosConfig.cluster_name` / `refresh_interval`、
   `ApolloConfig.secret_key` 未透传给 SDK；`ApolloConfig.server_addr` 只在
   「从配置文件读取」路径被折算成 `ip`，程序化传入时会在 `New` 内的拷贝中丢失。
4. **集群规模上限**：Nacos 文件路径最多读 10 个节点（循环 `i < 10`）；
   Apollo 多节点只取第一个节点（源码注释：多节点集群模式暂不支持）。
5. **无占位符展开**：`configs/*.yaml` 中的 `${MYSQL_DSN}` 等是字面量，
   本模块不会替换；需要注入密钥请用 `HUBX_*` 环境变量或直接写真实值。
6. **本地加载错误是降级而非失败**：只有「文件存在但读取/解析出错」才会 `Warn` 并降级为
   空配置（所有读取返回零值），文件不存在则按「只有默认值」正常返回；
   而远程连接失败会直接返回 error——两者行为不同，排障时注意区分。
7. **`Provider` 不可外部构造**：字段未导出，无法注入自定义后端或 mock；
   测试中如需替换实现，只能面向 `Config` 接口自己实现一份。
8. **watcher 注册无锁**：三个实现的 `watchers` 切片在注册时未加锁
   （数据本身有 `sync.RWMutex`），建议在初始化阶段调用 `Watch`，避免运行期并发注册。

---

## 13. 版本与兼容性

| 项 | 值 |
| :--- | :--- |
| Module | `github.com/gospacex/configx` |
| Go 版本 | `go 1.21`（取自 `go.mod`） |
| 当前版本标记 | v0.0.1 |
| 直接依赖 | `github.com/spf13/viper v1.18.2`、`github.com/nacos-group/nacos-sdk-go/v2 v2.2.5`、`gopkg.in/apollo.v0 v0.3.0`、`gopkg.in/yaml.v3 v3.0.1`、`github.com/stretchr/testify v1.10.0`（测试） |

- hubx 生态内各模块通过 `replace ... => ../configx` 指向本地路径，版本号不单独演进。
- 对外发布后，消费方需自行声明依赖与 replace；`replace` 不向下传递。

---

## 14. 许可证

MIT License
