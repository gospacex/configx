// Package flatten provides helpers to navigate map[string]any trees with
// dotted key paths (e.g. "server.http.port") and to set values at such paths.
package flatten

import (
	"strconv"
	"strings"
)

// Get returns the value stored at the dotted key, descending into nested
// maps and slices (integer segments index into slices). Returns (nil, false)
// when any segment is missing or the type does not allow further descent.
//
// Keys themselves may not contain '.'.
func Get(root map[string]any, key string) (any, bool) {
	if key == "" {
		return root, true
	}
	parts := strings.Split(key, ".")
	var cur any = root
	for _, p := range parts {
		switch node := cur.(type) {
		case map[string]any:
			v, ok := node[p]
			if !ok {
				return nil, false
			}
			cur = v
		case []any:
			idx, err := strconv.Atoi(p)
			if err != nil || idx < 0 || idx >= len(node) {
				return nil, false
			}
			cur = node[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

// Set writes value at the dotted key, creating intermediate maps as needed.
// If an intermediate non-map exists in the way it will be overwritten with
// a new map. The root map is mutated in place and returned for chaining.
func Set(root map[string]any, key string, value any) map[string]any {
	if key == "" {
		return root
	}
	parts := strings.Split(key, ".")
	cur := root
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = value
			return root
		}
		next, ok := cur[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[p] = next
		}
		cur = next
	}
	return root
}
