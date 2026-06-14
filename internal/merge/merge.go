// Package merge implements deep-merge semantics for map[string]any trees.
package merge

// DeepMerge merges src on top of dst, returning a new map. Maps are merged
// recursively, every other value (including slices) is overwritten by src.
// Neither input is mutated.
//
// This is intentionally simple and predictable: slices are NOT concatenated,
// because configuration files often define complete replacement lists and
// concatenation surprises operators.
func DeepMerge(dst, src map[string]any) map[string]any {
	out := cloneMap(dst)
	for k, sv := range src {
		if dv, ok := out[k]; ok {
			dm, dOK := dv.(map[string]any)
			sm, sOK := sv.(map[string]any)
			if dOK && sOK {
				out[k] = DeepMerge(dm, sm)
				continue
			}
		}
		out[k] = cloneValue(sv)
	}
	return out
}

// DeepMergeAll merges a sequence of maps left-to-right; later maps win.
func DeepMergeAll(maps ...map[string]any) map[string]any {
	out := map[string]any{}
	for _, m := range maps {
		out = DeepMerge(out, m)
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneValue(v)
	}
	return out
}

func cloneValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return cloneMap(x)
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = cloneValue(e)
		}
		return out
	default:
		return v
	}
}
