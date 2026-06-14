// Example: load configuration from Nacos and hot-reload on remote changes.
//
// Requires a running Nacos server. Set the following env vars:
//
//	NACOS_HOST=127.0.0.1 NACOS_PORT=8848 NACOS_DATAID=app.yaml NACOS_GROUP=DEFAULT_GROUP
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gospacex/configx"
	"github.com/gospacex/configx/source/nacos"
)

func main() {
	port, _ := strconv.ParseUint(getenv("NACOS_PORT", "8848"), 10, 64)
	src, err := nacos.New(nacos.Config{
		Servers: []nacos.ServerConfig{{
			IPAddr: getenv("NACOS_HOST", "127.0.0.1"),
			Port:   port,
		}},
		Client: nacos.ClientConfig{
			NamespaceID: os.Getenv("NACOS_NAMESPACE"),
			Username:    os.Getenv("NACOS_USER"),
			Password:    os.Getenv("NACOS_PASS"),
		},
		DataID: getenv("NACOS_DATAID", "app.yaml"),
		Group:  getenv("NACOS_GROUP", "DEFAULT_GROUP"),
		Format: "yaml",
	})
	if err != nil {
		log.Fatal(err)
	}

	c, err := configx.New(configx.WithSource(src))
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	c.OnChange(func(snap map[string]any) {
		fmt.Println("[nacos reload]", snap)
	})

	fmt.Println("initial snapshot:", c.Snapshot())
	fmt.Println("waiting for changes; Ctrl-C to quit")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
