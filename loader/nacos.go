package loader

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

// NacosLoader 从 Nacos 配置中心加载配置
type NacosLoader struct {
	client config_client.IConfigClient
	group  string
}

// NewNacosLoader 创建 NacosLoader 实例
func NewNacosLoader(endpoint string, namespace string, group string) (*NacosLoader, error) {
	log.Printf("DEBUG: NewNacosLoader endpoint=%q namespace=%q group=%q", endpoint, namespace, group)

	parts := strings.Split(endpoint, ":")
	host := parts[0]
	var port uint64 = 8848
	if len(parts) > 1 {
		if p, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
			port = p
		}
	}

	sc := []constant.ServerConfig{
		{
			IpAddr: host,
			Port:   port,
		},
	}
	cc := constant.ClientConfig{
		NamespaceId:         namespace,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
	}
	client, err := clients.NewConfigClient(vo.NacosClientParam{
		ServerConfigs: sc,
		ClientConfig:  &cc,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Nacos client: %w", err)
	}

	log.Printf("DEBUG: Nacos client created for %s:%d, performing connection check", host, port)

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 3*time.Second)
	if err != nil {
		log.Printf("DEBUG: Nacos TCP connection check failed: %v", err)
		return nil, fmt.Errorf("Nacos connection check failed: %w", err)
	}
	conn.Close()
	log.Printf("DEBUG: Nacos TCP connection check passed for %s:%d", host, port)

	content, err := client.GetConfig(vo.ConfigParam{DataId: "__health_check__", Group: group})
	if err == nil && content == "" {
		log.Printf("WARN: Nacos health check returned empty content for non-existent key (server reachable but key not found)")
	} else if err != nil {
		log.Printf("DEBUG: Nacos health check failed: %v", err)
		return nil, fmt.Errorf("Nacos connection check failed: %w", err)
	}
	log.Printf("DEBUG: Nacos health check passed for %s:%d", host, port)
	return &NacosLoader{
		client: client,
		group:  group,
	}, nil
}

// Load 从 Nacos 加载配置
func (l *NacosLoader) Load(key string) ([]byte, error) {
	log.Printf("DEBUG: Nacos Load called with key=%q group=%q", key, l.group)

	content, err := l.client.GetConfig(vo.ConfigParam{
		DataId: key,
		Group:  l.group,
	})
	log.Printf("DEBUG: Nacos GetConfig returned content_len=%d err=%v", len(content), err)

	if err != nil {
		log.Printf("WARN: Nacos GetConfig failed for key=%s, group=%s, err=%v, trying HTTP fallback", key, l.group, err)
		return l.httpGet(key)
	}
	if content == "" {
		log.Printf("WARN: Nacos GetConfig returned empty content for key=%q group=%q", key, l.group)
		return nil, fmt.Errorf("Nacos: empty config received for key %q, group %q", key, l.group)
	}
	return []byte(content), nil
}

// httpGet 作为降级方案通过 HTTP 获取配置
func (l *NacosLoader) httpGet(key string) ([]byte, error) {
	return nil, fmt.Errorf("HTTP fallback not implemented for key %q", key)
}

// Watch 监听配置变更
func (l *NacosLoader) Watch(key string, fn func([]byte)) {
	err := l.client.ListenConfig(vo.ConfigParam{
		DataId: key,
		Group:  l.group,
		OnChange: func(namespace, group, dataId, data string) {
			log.Printf("INFO: Nacos config changed: key=%s, group=%s", key, group)
			fn([]byte(data))
		},
	})
	if err != nil {
		log.Printf("WARN: ListenConfig failed for key=%s, group=%s, err=%v", key, l.group, err)
	}
}

func (l *NacosLoader) Close() error {
	return nil
}
