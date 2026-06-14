package configx

import (
	"fmt"
	"strconv"
	"time"
)

// String returns the value at key as a string. ErrKeyNotFound is returned
// when the key is absent; ErrTypeMismatch when the value cannot be coerced.
func (c *Config) String(key string) (string, error) {
	v, ok := c.Get(key)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	switch x := v.(type) {
	case string:
		return x, nil
	case fmt.Stringer:
		return x.String(), nil
	case bool, int, int32, int64, float32, float64, uint, uint32, uint64:
		return fmt.Sprint(x), nil
	default:
		return "", fmt.Errorf("%w: %s is %T", ErrTypeMismatch, key, v)
	}
}

// StringOr returns the string value or def when the key is missing.
func (c *Config) StringOr(key, def string) string {
	if v, err := c.String(key); err == nil {
		return v
	}
	return def
}

// Int returns the value at key as an int64.
func (c *Config) Int(key string) (int64, error) {
	v, ok := c.Get(key)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	switch x := v.(type) {
	case int:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case int64:
		return x, nil
	case uint:
		return int64(x), nil
	case uint32:
		return int64(x), nil
	case uint64:
		return int64(x), nil
	case float32:
		return int64(x), nil
	case float64:
		return int64(x), nil
	case string:
		n, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %s = %q", ErrTypeMismatch, key, x)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("%w: %s is %T", ErrTypeMismatch, key, v)
	}
}

// IntOr returns the int value or def when the key is missing or invalid.
func (c *Config) IntOr(key string, def int64) int64 {
	if v, err := c.Int(key); err == nil {
		return v
	}
	return def
}

// Float returns the value at key as a float64.
func (c *Config) Float(key string) (float64, error) {
	v, ok := c.Get(key)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	switch x := v.(type) {
	case float64:
		return x, nil
	case float32:
		return float64(x), nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case string:
		n, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %s = %q", ErrTypeMismatch, key, x)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("%w: %s is %T", ErrTypeMismatch, key, v)
	}
}

// FloatOr returns the float value or def when missing/invalid.
func (c *Config) FloatOr(key string, def float64) float64 {
	if v, err := c.Float(key); err == nil {
		return v
	}
	return def
}

// Bool returns the value at key as a bool. Recognised string values:
// "1","t","true","y","yes","on" and their false counterparts (case-insensitive).
func (c *Config) Bool(key string) (bool, error) {
	v, ok := c.Get(key)
	if !ok {
		return false, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	switch x := v.(type) {
	case bool:
		return x, nil
	case string:
		b, err := strconv.ParseBool(x)
		if err != nil {
			return false, fmt.Errorf("%w: %s = %q", ErrTypeMismatch, key, x)
		}
		return b, nil
	case int:
		return x != 0, nil
	case int64:
		return x != 0, nil
	default:
		return false, fmt.Errorf("%w: %s is %T", ErrTypeMismatch, key, v)
	}
}

// BoolOr returns the bool value or def.
func (c *Config) BoolOr(key string, def bool) bool {
	if v, err := c.Bool(key); err == nil {
		return v
	}
	return def
}

// Duration returns the value at key as a time.Duration. Strings are parsed
// with time.ParseDuration; numbers are interpreted as nanoseconds (matches
// time.Duration's underlying type).
func (c *Config) Duration(key string) (time.Duration, error) {
	v, ok := c.Get(key)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	switch x := v.(type) {
	case time.Duration:
		return x, nil
	case string:
		d, err := time.ParseDuration(x)
		if err != nil {
			return 0, fmt.Errorf("%w: %s = %q", ErrTypeMismatch, key, x)
		}
		return d, nil
	case int:
		return time.Duration(x), nil
	case int64:
		return time.Duration(x), nil
	case float64:
		return time.Duration(x), nil
	default:
		return 0, fmt.Errorf("%w: %s is %T", ErrTypeMismatch, key, v)
	}
}

// DurationOr returns the duration value or def.
func (c *Config) DurationOr(key string, def time.Duration) time.Duration {
	if v, err := c.Duration(key); err == nil {
		return v
	}
	return def
}

// StringSlice returns the value at key as []string.
func (c *Config) StringSlice(key string) ([]string, error) {
	v, ok := c.Get(key)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	switch x := v.(type) {
	case []string:
		return x, nil
	case []any:
		out := make([]string, 0, len(x))
		for i, e := range x {
			s, ok := e.(string)
			if !ok {
				return nil, fmt.Errorf("%w: %s[%d] is %T", ErrTypeMismatch, key, i, e)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%w: %s is %T", ErrTypeMismatch, key, v)
	}
}

// StringSliceOr returns the slice value or def.
func (c *Config) StringSliceOr(key string, def []string) []string {
	if v, err := c.StringSlice(key); err == nil {
		return v
	}
	return def
}
