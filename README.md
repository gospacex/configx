# ConfigX

<div align="center">

<!-- GitHub Stats -->
![Go Version](https://img.shields.io/github/go-mod/go-version/gospacex/configx)
![License](https://img.shields.io/github/license/gospacex/configx)
[![Go Report](https://goreportcard.com/badge/github.com/gospacex/configx)](https://goreportcard.com/report/github.com/gospacex/configx)

<!-- GitHub Activity & Stats -->
[![Stars](https://img.shields.io/github/stars/gospacex/configx?style=flat-square)](https://github.com/gospacex/configx/stargazers)
[![Forks](https://img.shields.io/github/forks/gospacex/configx?style=flat-square)](https://github.com/gospacex/configx/network/members)
[![Issues](https://img.shields.io/github/issues/gospacex/configx?style=flat-square)](https://github.com/gospacex/configx/issues)
[![Last Commit](https://img.shields.io/github/last-commit/gospacex/configx/main?style=flat-square)](https://github.com/gospacex/configx/commits)


</div>

> 一行代码搞定应用配置！支持本地配置、Apollo、Nacos 三种配置源，自动降级，无缝切换。

## 特性

| 特性 | 说明 |
|------|------|
| 多源支持 | 同时支持本地配置、Apollo 配置中心、Nacos 配置中心 |
| 自动降级 | 远程配置不可用时自动降级到本地配置，保证服务可用性 |
| 热更新 | 支持配置监听，配置变更实时生效 |
| 统一 API | 一致的接口调用方式，无需关心底层实现 |
| 超时控制 | 支持配置加载超时，避免请求阻塞 |

## 支持的配置源

| Provider | 说明 | 创建方式 |
|----------|------|----------|
| Viper | 本地配置文件（JSON/YAML/TOML 等） | `NewConfigCenter("", localFile, "")` |
| Apollo | Apollo 配置中心 | `NewConfigCenter("apollo", localFile, "host:port")` |
| Nacos | Nacos 配置中心 | `NewConfigCenter("nacos", localFile, "host:port")` |

## 架构

```
configx/
├── config.go              # 核心入口，ConfigCenter 定义
├── errors.go              # 错误定义
├── loader/                # 配置加载服务层
│   ├── interface.go       # ConfigLoader 接口
│   ├── apollo.go          # Apollo 加载器
│   ├── nacos.go           # Nacos 加载器
│   └── viper.go           # Viper 本地加载器
├── standard/              # 配置模型层
│   ├── redis.go           # Redis 配置
│   ├── mysql.go           # MySQL 配置
│   ├── mongodb.go         # MongoDB 配置
│   ├── kafka.go           # Kafka 配置
│   ├── elasticsearch.go   # Elasticsearch 配置
│   ├── rabbitmq.go        # RabbitMQ 配置
│   ├── rocketmq.go        # RocketMQ 配置
│   ├── dtm.go             # DTM 配置
│   └── jaeger.go          # Jaeger 配置
├── thirdparty/            # 第三方配置
│   └── passthrough.go     # 通用配置映射
└── docs/                  # 文档
    └── ONBOARDING.md      # 新人上手指南
```

### 层级说明

| 层级 | 说明 |
|------|------|
| **API 层** | ConfigCenter 核心接口，统一的配置获取和监听 API |
| **配置加载服务层** | Apollo、Nacos、Viper 三种配置源加载器实现 |
| **配置模型层** | Redis、MySQL、Kafka 等中间件配置结构体 |

## 安装

```bash
go get github.com/gospacex/configx
```

## 快速开始

### 创建配置中心

```go
import "github.com/gospacex/configx"

// 方式一：简单用法
// 支持 remoteType: "apollo", "nacos", ""
cc, err := configx.NewConfigCenter("apollo", "./config.yaml", "http://localhost:8080")

// 方式二：带超时
cc, err := configx.NewConfigCenterWithTimeout("nacos", "./config.yaml", "http://localhost:8848", 5*time.Second)

// 方式三：完整参数（适用于 Nacos）
cc, err := configx.NewConfigCenterWithTimeoutWithParams(
    "nacos",
    "./config.yaml",
    "http://localhost:8848",
    5*time.Second,
    "namespace-id",      // Nacos 命名空间 ID
    "group-name",        // Nacos 分组
    "data-id",           // Nacos Data ID
)
```

### 获取配置

```go
// 获取整个配置文件内容
data, err := cc.Get("")

// 获取特定配置项
value, err := cc.Get("app.name")
```

### 监听配置变化

```go
// 监听配置变化，实时更新
cc.Watch("app.name", func(value interface{}) {
    fmt.Println("配置已更新:", value)
})
```

### 关闭连接

```go
cc.Close()
```

## 降级策略

```
Apollo/Nacos 远程配置中心
    ↓ 可用
  远程配置 ──────────────────────→ Get() 返回配置
    ↓ 不可用（超时/连接失败）
  本地 Viper 配置文件
    ↓
  Get() 返回本地配置
```

| 场景 | 处理方式 |
|------|----------|
| Apollo/Nacos 不可用 | 自动降级到本地配置 |
| 配置 Key 不存在 | 返回错误 |
| 本地配置也不存在 | 返回错误 |

## 核心概念

### ConfigLoader 接口

所有配置加载器（ViperLoader、ApolloLoader、NacosLoader）必须实现三个方法：

```go
type ConfigLoader interface {
    Load(key string) ([]byte, error)           // 加载配置
    Watch(key string, callback func(data []byte))  // 监听配置变更
    Close() error                             // 关闭连接
}
```

### 热更新

通过 `Watch` 方法监听配置变更，无需重启应用即可生效。

## 文件地图

| 文件 | 功能 | 复杂度 |
|------|------|--------|
| `config.go` | 核心入口，ConfigCenter 及工厂函数 | ⭐⭐ |
| `errors.go` | 错误定义（ErrKeyNotFound、ErrRemoteUnavailable 等） | ⭐ |
| `loader/interface.go` | ConfigLoader 接口定义 | ⭐ |
| `loader/apollo.go` | Apollo 加载器 | ⭐⭐ |
| `loader/nacos.go` | Nacos 加载器 | ⭐⭐ |
| `loader/viper.go` | Viper 本地加载器 | ⭐⭐ |
| `standard/*.go` | 中间件配置结构体（Redis、MySQL、Kafka 等） | ⭐ |

> ⭐ = 简单，⭐⭐ = 中等

## 本地开发

### 运行测试

```bash
# 运行所有测试
go test -v ./...

# 运行特定测试
go test -v -run "TestConfigCenter" ./...

# 带竞态检测
go test -race ./...
```

### Nacos 本地开发

启动 Nacos Docker：

```bash
docker run -d --name nacos -p 8848:8848 nacos/nacos-server:v2.4.2
```

## 相关资源

- [ONBOARDING.md](docs/ONBOARDING.md) — 新人上手指南
- [config.go](config.go) — 核心 API 实现
- [loader/interface.go](loader/interface.go) — 配置加载器接口定义

## 许可证

MIT License - see [LICENSE](LICENSE) for details.

---

<div align="center">

✨ Made with ❤️ by [gospacex](https://github.com/gospacex)

</div>