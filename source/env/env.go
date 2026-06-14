// Package env implements a configx Source backed by process environment
// variables. Variables are mapped into a nested map[string]any tree:
//
//	APP_SERVER_HTTP_PORT=8080  →  app.server.http.port = "8080"  (when prefix="")
//	APP_SERVER_HTTP_PORT=8080  →  server.http.port     = "8080"  (when prefix="APP")
//
// Values are kept as strings; configx's type-aware getters convert them on demand.
package env

import (
	"context"
	"os"
	"strings"

	"github.com/gospacex/configx"
	"github.com/gospacex/configx/internal/flatten"
)

// Option configures the Source.
type Option func(*Source)

// WithPrefix only considers variables whose names start with prefix+"_".
// The prefix itself is stripped from the produced key.
func WithPrefix(p string) Option { return func(s *Source) { s.prefix = p } }

// WithSeparator overrides the character used to split env var names into
// key segments. Default: "_".
func WithSeparator(sep string) Option { return func(s *Source) { s.sep = sep } }

// WithLowercaseKeys produces lower-cased keys (default true).
func WithLowercaseKeys(lc bool) Option { return func(s *Source) { s.lower = lc } }

// Source reads environment variables on Load. It does not support Watch.
type Source struct {
	prefix string
	sep    string
	lower  bool
}

// New returns an env Source.
func New(opts ...Option) *Source {
	s := &Source{sep: "_", lower: true}
	for _, o := range opts {
		o(s)
	}
	return s
}

// ID returns "env:<PREFIX>".
func (s *Source) ID() string { return "env:" + s.prefix }

// Load scans os.Environ() and builds the nested map.
func (s *Source) Load(_ context.Context) (map[string]any, error) {
	out := map[string]any{}
	prefix := ""
	if s.prefix != "" {
		prefix = s.prefix + "_"
	}
	for _, kv := range os.Environ() {
		i := strings.IndexByte(kv, '=')
		if i < 0 {
			continue
		}
		k, v := kv[:i], kv[i+1:]
		if prefix != "" {
			if !strings.HasPrefix(k, prefix) {
				continue
			}
			k = k[len(prefix):]
		}
		if k == "" {
			continue
		}
		if s.lower {
			k = strings.ToLower(k)
		}
		dotted := strings.ReplaceAll(k, s.sep, ".")
		flatten.Set(out, dotted, v)
	}
	return out, nil
}

// Watch is unsupported.
func (s *Source) Watch(_ context.Context) (<-chan configx.Event, error) { return nil, nil }

// Close is a no-op.
func (s *Source) Close() error { return nil }
