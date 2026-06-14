package configx

import (
	"sort"

	"github.com/gospacex/configx/internal/flatten"
)

// AllKeys returns every dotted leaf key present in the current snapshot,
// sorted lexicographically. Useful for diagnostics and tests.
func (c *Config) AllKeys() []string {
	snap := c.Snapshot()
	keys := make([]string, 0, 32)
	walk("", snap, &keys)
	sort.Strings(keys)
	return keys
}

func walk(prefix string, node any, out *[]string) {
	m, ok := node.(map[string]any)
	if !ok {
		if prefix != "" {
			*out = append(*out, prefix)
		}
		return
	}
	if len(m) == 0 && prefix != "" {
		*out = append(*out, prefix)
		return
	}
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		walk(key, v, out)
	}
}

// Sub returns a child Config rooted at key. The child shares no state with
// the parent (it is a deep-copied snapshot) and never hot-reloads — call
// Sub again after the parent reloads if you need a fresh view.
//
// Returns nil when the key is missing or does not point to a map.
func (c *Config) Sub(key string) *Config {
	v, ok := c.Get(key)
	if !ok {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	sub := &Config{opts: c.opts}
	cp := deepCopyMap(m)
	sub.effective.Store(&cp)
	return sub
}

// Set overrides a key in the current snapshot. It is intended for tests and
// dynamic feature flags; the value is replaced on the next reload, so do
// not use Set as a persistent configuration store.
//
// Set is safe for concurrent use; it publishes a fresh snapshot atomically.
func (c *Config) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cur := map[string]any{}
	if p := c.effective.Load(); p != nil {
		cur = deepCopyMap(*p)
	}
	flatten.Set(cur, key, value)
	c.effective.Store(&cur)
}
