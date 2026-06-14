// Example: bind merged configuration into a struct with validation,
// using a file + environment variable overrides.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gospacex/configx"
	"github.com/gospacex/configx/source/env"
	"github.com/gospacex/configx/source/file"
)

type AppConfig struct {
	Server struct {
		Host string `configx:"host" validate:"required"`
		Port int    `configx:"port" validate:"required,min=1,max=65535"`
	} `configx:"server"`
	DB struct {
		DSN          string        `configx:"dsn" validate:"required"`
		MaxOpenConns int           `configx:"max_open_conns"`
		Timeout      time.Duration `configx:"timeout"`
	} `configx:"db"`
	Tags []string `configx:"tags"`
}

func main() {
	c, err := configx.New(
		configx.WithSource(file.New("examples/struct-binding/config.yaml", file.WithWatch(false))),
		configx.WithSource(env.New(env.WithPrefix("APP"))),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	var cfg AppConfig
	if err := c.Unmarshal(&cfg); err != nil {
		log.Fatal(err)
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Println(string(b))
}
