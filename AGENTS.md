# AGENTS.md

## Go Module
```
github.com/gospacex/configx
```
Go 1.26.2

## Build & Test
```bash
go test ./...              # run all tests
go vet ./...               # type check
gofmt -l .                 # check formatting (repo needs formatting)
```

**Current issues:**
- `loader/nacos.go:32` — `clients.NewConfigClient` API mismatch: nacos-sdk-go changed signature, takes `vo.NacosClientParam` not two separate args
- `loader/nacos.go:49,69` — `log.Warn()`/`log.Info()` used as value (no return value); zerolog log statements are broken
- `loader/nacos.go:64` — `Watch` returns `error` but `ConfigLoader` interface `Watch(key, fn)` has no return value
- `config.go:41` — `NewNacosLoader` called with 4 args (including `dataId`) but only takes 3 (`endpoint, namespace, group`)
- Several files need `gofmt` formatting

## Architecture
- `config.go` — `ConfigCenter` entry point; wraps local + remote loaders with fallback
- `loader/` — pluggable `ConfigLoader` interface: `ViperLoader` (local), `ApolloLoader`, `NacosLoader`
- `standard/` — typed config structs: MySQL, Redis, DTM, Elasticsearch, Kafka, MongoDB, RabbitMQ, RocketMQ, Jaeger
- `thirdparty/` — passthrough configs (Alipay, etc.)
- `docs/examples/` — runnable examples: local, remote (apollo), nacos, fallback

## Key Patterns
- Remote loaders are optional; local Viper config is always the fallback
- `NewConfigCenter(remoteType, localConfigPath, remoteAddr)` — remoteType is `apollo`, `nacos`, or `""`
- `Get(key)` returns `([]byte, error)` — caller unmarshals (JSON)
- `Watch(key, fn)` registers a callback for config changes on both remote and local

## OpenSpec Workflow
Uses OpenSpec (`.claude/skills/openspec-*`):
1. `/openspec-propose` — draft a change proposal
2. `/openspec-explore` — investigate before implementing
3. `/openspec-apply-change` — implement from a proposal
4. `/openspec-archive-change` — finalize after merge

Config at `openspec/config.yaml`.

## Code Style
- Standard Go formatting (gofmt)
- Chinese docs exist under `docs/examples/`
- Errors defined in `errors.go`; loader-specific errors in each loader file