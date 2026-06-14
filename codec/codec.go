// Package codec defines the encoding / decoding interface used by configx
// to translate raw bytes coming from a Source into a generic map structure
// and back.
package codec

import (
	"fmt"
	"strings"
	"sync"
)

// Codec marshals/unmarshals a configuration payload to/from a generic
// map[string]any tree. Implementations must be safe for concurrent use.
type Codec interface {
	// Name returns the canonical lowercase format identifier (e.g. "yaml").
	Name() string
	// Unmarshal decodes raw bytes into a map[string]any.
	Unmarshal(data []byte) (map[string]any, error)
	// Marshal encodes a map[string]any back into bytes.
	Marshal(v map[string]any) ([]byte, error)
}

var (
	mu       sync.RWMutex
	registry = map[string]Codec{}
)

// Register makes a codec available under its Name(). It is safe to call
// concurrently. Re-registering an existing codec replaces the previous one.
func Register(c Codec) {
	mu.Lock()
	defer mu.Unlock()
	registry[strings.ToLower(c.Name())] = c
}

// Get returns the codec registered for the given format name.
// The name lookup is case-insensitive. Extensions like ".yaml" are accepted.
func Get(format string) (Codec, error) {
	mu.RLock()
	defer mu.RUnlock()
	key := strings.TrimPrefix(strings.ToLower(format), ".")
	// common aliases
	switch key {
	case "yml":
		key = "yaml"
	}
	c, ok := registry[key]
	if !ok {
		return nil, fmt.Errorf("codec %q not registered", format)
	}
	return c, nil
}

// MustGet is like Get but panics on error. Useful at package init.
func MustGet(format string) Codec {
	c, err := Get(format)
	if err != nil {
		panic(err)
	}
	return c
}
