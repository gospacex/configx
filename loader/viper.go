package loader

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var ErrKeyNotFound = errors.New("config key not found")

type ViperLoader struct {
	v *viper.Viper
}

func NewViperLoader(configPath string) (*ViperLoader, error) {
	v := viper.New()
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	return &ViperLoader{v: v}, nil
}

func (l *ViperLoader) Load(key string) ([]byte, error) {
	if key == "" || key == "config" {
		data, err := os.ReadFile(l.v.ConfigFileUsed())
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	val := l.v.Get(key)
	if val == nil {
		return nil, ErrKeyNotFound
	}

	// For simple scalar types, return raw value; for complex types, marshal as JSON
	switch v := val.(type) {
	case string:
		return []byte(v), nil
	case int, int8, int16, int32, int64, float64, float32, bool:
		return json.Marshal(v)
	default:
		return json.Marshal(val)
	}
}

func (l *ViperLoader) Watch(key string, fn func([]byte)) {
	l.v.WatchConfig()
	l.v.OnConfigChange(func(e fsnotify.Event) {
		if data, err := l.Load(key); err == nil {
			fn(data)
		}
	})
}

func (l *ViperLoader) Close() error {
	return nil
}