package configx

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gospacex/configx/loader"
)

type ConfigCenter struct {
	local     *loader.ViperLoader
	remote    loader.ConfigLoader
	loadTimer time.Duration
}

func NewConfigCenter(remoteType string, localConfigPath string, remoteAddr string) (*ConfigCenter, error) {
	return NewConfigCenterWithTimeoutWithParams(remoteType, localConfigPath, remoteAddr, 0, "", "", "")
}

func NewConfigCenterWithTimeout(remoteType string, localConfigPath string, remoteAddr string, timeout time.Duration) (*ConfigCenter, error) {
	return NewConfigCenterWithTimeoutWithParams(remoteType, localConfigPath, remoteAddr, timeout, "", "", "")
}

func NewConfigCenterWithTimeoutWithParams(remoteType string, localConfigPath string, remoteAddr string, timeout time.Duration, namespace string, group string, dataId string) (*ConfigCenter, error) {
	if remoteType != "apollo" && remoteType != "nacos" && remoteType != "" {
		return nil, ErrInvalidRemoteType
	}

	local, err := loader.NewViperLoader(localConfigPath)
	if err != nil {
		return nil, ErrLocalUnavailable
	}

	var remote loader.ConfigLoader
	if remoteType != "" {
		switch remoteType {
		case "apollo":
			remote, err = loader.NewApolloLoader(remoteAddr, "app", "default", namespace, group)
		case "nacos":
			remote, err = loader.NewNacosLoader(remoteAddr, namespace, group)
		}
		if err != nil {
			log.Printf("DEBUG: NewConfigCenterWithTimeoutWithParams remote creation failed: err=%v", err)
			remote = nil
		}
	}

	return &ConfigCenter{
		local:     local,
		remote:    remote,
		loadTimer: timeout,
	}, nil
}

func (c *ConfigCenter) Get(key string) (interface{}, error) {
	var data []byte
	var loadErr error

	// Try remote first if available
	if c.remote != nil {
		ctx, cancel := context.WithTimeout(context.Background(), c.loadTimer)
		defer cancel()

		done := make(chan struct{})

		go func() {
			data, loadErr = c.remote.Load(key)
			close(done)
		}()

		select {
		case <-ctx.Done():
			data = nil
			loadErr = context.DeadlineExceeded
		case <-done:
		}

		if loadErr == nil && len(data) > 0 {
			// Unmarshal JSON to interface{} for consistency
			var result interface{}
			if err := json.Unmarshal(data, &result); err == nil {
				return result, nil
			}
			return data, nil
		}
	}

	// Fall back to local
	data, err := c.local.Load(key)
	if err == nil && len(data) > 0 {
		// Unmarshal JSON to interface{} for consistency
		var result interface{}
		if err := json.Unmarshal(data, &result); err == nil {
			return result, nil
		}
		return data, nil
	}

	return nil, ErrKeyNotFound
}

func (c *ConfigCenter) Watch(key string, fn func(interface{})) {
	if c.remote != nil {
		c.remote.Watch(key, func(data []byte) {
			fn(data)
		})
	}
	c.local.Watch(key, func(data []byte) {
		fn(data)
	})
}

func (c *ConfigCenter) Close() error {
	if c.remote != nil {
		return c.remote.Close()
	}
	return nil
}
