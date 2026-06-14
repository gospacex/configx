// Package quick provides convenience constructors for the common
// configuration bootstrap patterns. It is a separate package so the core
// configx module stays free of cycles.
//
// All functions in this package are thin wrappers around configx.New plus
// one or more Source constructors. Use the core API directly when you need
// finer control.
package quick

import (
	"github.com/gospacex/configx"
	"github.com/gospacex/configx/source/env"
	"github.com/gospacex/configx/source/file"
)

// New is a re-export of configx.New for ergonomic single-import usage:
//
//	import "github.com/gospacex/configx/quick"
//	c, err := quick.FromFile("config.yaml")
func New(opts ...configx.Option) (*configx.Config, error) { return configx.New(opts...) }

// MustNew panics on error.
func MustNew(opts ...configx.Option) *configx.Config {
	c, err := configx.New(opts...)
	if err != nil {
		panic(err)
	}
	return c
}

// FromFile builds a Config backed by a single file source with hot-reload
// enabled. Extra Option values are appended after the file source.
//
//	c, err := quick.FromFile("config.yaml")
//	c, err := quick.FromFile("config.yaml", configx.WithSource(env.Prefixed("APP")))
func FromFile(path string, opts ...configx.Option) (*configx.Config, error) {
	all := append([]configx.Option{configx.WithSource(file.New(path))}, opts...)
	return configx.New(all...)
}

// MustFromFile is FromFile + panic.
func MustFromFile(path string, opts ...configx.Option) *configx.Config {
	c, err := FromFile(path, opts...)
	if err != nil {
		panic(err)
	}
	return c
}

// FromFiles builds a Config from several files. Later files override
// earlier ones (same precedence as WithSource). All files use the default
// hot-reload behaviour.
//
//	c, err := quick.FromFiles("config.yaml", "config.local.yaml")
func FromFiles(paths ...string) (*configx.Config, error) {
	srcs := make([]configx.Option, 0, len(paths))
	for _, p := range paths {
		srcs = append(srcs, configx.WithSource(file.New(p)))
	}
	return configx.New(srcs...)
}

// FromFileAndEnv is the canonical 12-factor combo: a single file overridden
// by environment variables with the given prefix (e.g. "APP" matches
// APP_SERVER_PORT → server.port).
//
//	c, err := quick.FromFileAndEnv("config.yaml", "APP")
func FromFileAndEnv(path, envPrefix string, opts ...configx.Option) (*configx.Config, error) {
	all := append([]configx.Option{
		configx.WithSource(file.New(path)),
		configx.WithSource(env.Prefixed(envPrefix)),
	}, opts...)
	return configx.New(all...)
}

// MustFromFileAndEnv is FromFileAndEnv + panic.
func MustFromFileAndEnv(path, envPrefix string, opts ...configx.Option) *configx.Config {
	c, err := FromFileAndEnv(path, envPrefix, opts...)
	if err != nil {
		panic(err)
	}
	return c
}
