// configx/example/main.go
// 演示：Nacos 优先级 > 本地 Viper
//
// 核心逻辑：
//  1. 本地配置 app.name = myapp
//  2. Nacos 配置 app.name = nacos-override
//  3. 开关开启后，通过 configx 句柄读取 app.name → 得到 nacos-override
//  4. 本地有但 Nacos 没有的 key（如 cachex.addr）→ 兜底读本地
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gospacex/configx"
)

func main() {
	const nacosURL = "http://127.0.0.1:8848/nacos/v1/cs/configs"

	// ============================================================
	// 准备：发布 nacos 配置（覆盖 app.name 和 app.version）
	// ============================================================
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("准备：发布 Nacos 配置")
	fmt.Println("═══════════════════════════════════════════════════════════")
	nacosContent := `
app:
  name: nacos-override
  version: 99.0.0
  env: production

logx:
  level: warn
`
	if err := publishNacosConfig(nacosURL, "myapp", "DEFAULT_GROUP", nacosContent); err != nil {
		log.Fatal("nacos 不可用，请先启动 nacos:", err)
	}
	defer cleanupNacosConfig(nacosURL, "myapp", "DEFAULT_GROUP")
	fmt.Println("  已发布 Nacos 配置: app.name=nacos-override, app.version=99.0.0")
	fmt.Println()

	// ============================================================
	// 第一步：纯本地 Viper（开关=false）
	// ============================================================
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("第一步：configx.remote.enabled = false → 仅读本地 Viper")
	fmt.Println("═══════════════════════════════════════════════════════════")

	localCfg, err := configx.New(&configx.ConfigOptions{
		ConfigFile: "example/config.yaml",
		ConfigType: "yaml",
		// Remote 未设置 → 从配置文件读取开关（当前=false）
	})
	if err != nil {
		log.Fatal(err)
	}
	defer localCfg.Close()

	localAppName := localCfg.GetString("app.name")
	localVersion := localCfg.GetString("app.version")
	localEnv := localCfg.GetString("app.env")
	localRedis := localCfg.GetString("xmiddleware.cachex.instances.default.addr")

	fmt.Printf("  本地读取 app.name    = %s\n", localAppName)
	fmt.Printf("  本地读取 app.version = %s\n", localVersion)
	fmt.Printf("  本地读取 app.env     = %s\n", localEnv)
	fmt.Printf("  本地读取 cachex.addr = %s\n", localRedis)
	fmt.Println()

	// ============================================================
	// 第二步：开启 Nacos（开关=true）
	// ============================================================
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("第二步：configx.remote.enabled = true → Nacos 覆盖本地")
	fmt.Println("═══════════════════════════════════════════════════════════")

	mixedCfg, err := configx.New(&configx.ConfigOptions{
		ConfigFile: "example/config.yaml",
		ConfigType: "yaml",
		Remote: configx.RemoteConfig{
			Enabled:  true, // ← 开启
			Provider: "nacos",
			Nacos: configx.NacosConfig{
				ServerConfigs: []configx.NacosServerConfig{{IP: "127.0.0.1", Port: 8848}},
				NamespaceID:   "",
				Group:         "DEFAULT_GROUP",
				DataID:        "myapp",
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer mixedCfg.Close()

	mixedAppName := mixedCfg.GetString("app.name")
	mixedVersion := mixedCfg.GetString("app.version")
	mixedEnv := mixedCfg.GetString("app.env")
	mixedRedis := mixedCfg.GetString("xmiddleware.cachex.instances.default.addr")
	mixedLogLevel := mixedCfg.GetString("logx.level")

	fmt.Printf("  混合读取 app.name    = %s\n", mixedAppName)
	fmt.Printf("  混合读取 app.version = %s\n", mixedVersion)
	fmt.Printf("  混合读取 app.env     = %s\n", mixedEnv)
	fmt.Printf("  混合读取 cachex.addr = %s\n", mixedRedis)
	fmt.Printf("  混合读取 logx.level  = %s\n", mixedLogLevel)
	fmt.Println()

	// ============================================================
	// 第三步：优先级对比表
	// ============================================================
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("第三步：优先级对比")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("  Key            │ 本地 Viper │ Nacos      │ 最终结果     │ 来源")
	fmt.Println("  ───────────────┼────────────┼────────────┼──────────────┼──────────")
	printRow("app.name", localAppName, "nacos-override", mixedAppName)
	printRow("app.version", localVersion, "99.0.0", mixedVersion)
	printRow("app.env", localEnv, "production", mixedEnv)
	printRow("cachex.addr", localRedis, "(无)", mixedRedis)
	printRow("logx.level", "debug", "warn", mixedLogLevel)
	fmt.Println()

	// ============================================================
	// 验证
	// ============================================================
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("验证结果")
	fmt.Println("═══════════════════════════════════════════════════════════")
	pass := 0
	total := 0

	total++
	if mixedAppName == "nacos-override" {
		fmt.Printf("  ✅ app.name:    Nacos(%s) 覆盖了 Viper(%s)\n", mixedAppName, localAppName)
		pass++
	} else {
		fmt.Printf("  ❌ app.name:    期望 nacos-override, 实际 %s\n", mixedAppName)
	}

	total++
	if mixedVersion == "99.0.0" {
		fmt.Printf("  ✅ app.version: Nacos(%s) 覆盖了 Viper(%s)\n", mixedVersion, localVersion)
		pass++
	} else {
		fmt.Printf("  ❌ app.version: 期望 99.0.0, 实际 %s\n", mixedVersion)
	}

	total++
	if mixedEnv == "production" {
		fmt.Printf("  ✅ app.env:     Nacos(%s) 覆盖了 Viper(%s)\n", mixedEnv, localEnv)
		pass++
	} else {
		fmt.Printf("  ❌ app.env:     期望 production, 实际 %s\n", mixedEnv)
	}

	total++
	if mixedRedis == localRedis {
		fmt.Printf("  ✅ cachex.addr: Nacos 无此 key，Viper 兜底(%s)\n", localRedis)
		pass++
	} else {
		fmt.Printf("  ❌ cachex.addr: 期望 %s, 实际 %s\n", localRedis, mixedRedis)
	}

	total++
	if mixedLogLevel == "warn" {
		fmt.Printf("  ✅ logx.level:  Nacos(%s) 覆盖了 Viper(debug)\n", mixedLogLevel)
		pass++
	} else {
		fmt.Printf("  ❌ logx.level:  期望 warn, 实际 %s\n", mixedLogLevel)
	}

	fmt.Printf("\n  结果: %d/%d 通过\n", pass, total)
	fmt.Println("═══════════════════════════════════════════════════════════")
}

func printRow(key, local, nacos, final string) {
	source := "Viper"
	if local != final {
		source = "Nacos"
	}
	fmt.Printf("  %-15s│ %-10s │ %-10s │ %-12s │ %s\n",
		key, trunc(local, 10), trunc(nacos, 10), trunc(final, 12), source)
}

func trunc(s string, max int) string {
	if len(s) > max {
		return s[:max-1] + "…"
	}
	return s
}

func publishNacosConfig(baseURL, dataID, group, content string) error {
	data := url.Values{}
	data.Set("dataId", dataID)
	data.Set("group", group)
	data.Set("type", "yaml")
	data.Set("content", content)
	resp, err := http.PostForm(baseURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	time.Sleep(300 * time.Millisecond)
	return nil
}

func cleanupNacosConfig(baseURL, dataID, group string) {
	req, _ := http.NewRequest("DELETE", baseURL+"?dataId="+dataID+"&group="+group, nil)
	if req != nil {
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}
}
