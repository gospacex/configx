// Example: load a single YAML file and read typed values.
package main

import (
	"fmt"
	"log"

	"github.com/gospacex/configx"
	"github.com/gospacex/configx/source/file"
)

func main() {
	c, err := configx.New(
		configx.WithSource(file.New("config.yaml", file.WithWatch(false))),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	fmt.Println("server.host =", c.StringOr("server.host", "127.0.0.1"))
	fmt.Println("server.port =", c.IntOr("server.port", 8080))
	fmt.Println("log.level   =", c.StringOr("log.level", "info"))
}
