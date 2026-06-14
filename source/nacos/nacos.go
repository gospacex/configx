// Package nacos implements a configx Source backed by a Nacos config server.
//
// The Source pulls one DataID/Group on Load and registers a long-poll
// listener on Watch. The raw payload is decoded by the format codec
// configured on the Source (defaults to "yaml").
package nacos

import (
	"context"
	"fmt"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"github.com/gospacex/configx"
	"github.com/gospacex/configx/codec"
)

// ServerConfig identifies a Nacos endpoint.
type ServerConfig struct {
	IPAddr      string
	Port        uint64
	ContextPath string // default: /nacos
	Scheme      string // http | https
}

// ClientConfig matches the subset of nacos client knobs we expose.
type ClientConfig struct {
	NamespaceID string
	Username    string
	Password    string
	TimeoutMs   uint64 // request timeout, default 5000
	LogLevel    string // debug|info|warn|error, default "warn"
	LogDir      string // default os.TempDir()/nacos/log
	CacheDir    string // default os.TempDir()/nacos/cache
}

// Config configures the Source.
type Config struct {
	Servers []ServerConfig
	Client  ClientConfig

	DataID string
	Group  string // default "DEFAULT_GROUP"
	Format string // codec name, default "yaml"
}

// Source pulls and watches a single DataID/Group from Nacos.
type Source struct {
	cfg    Config
	client config_client.IConfigClient

	mu       sync.Mutex
	out      chan configx.Event
	listened bool
	closed   bool
}

// New constructs a Source. The Nacos client connection is established lazily
// on the first Load/Watch so that constructors stay cheap and testable.
func New(cfg Config) (*Source, error) {
	if cfg.Group == "" {
		cfg.Group = "DEFAULT_GROUP"
	}
	if cfg.Format == "" {
		cfg.Format = "yaml"
	}
	if cfg.DataID == "" {
		return nil, fmt.Errorf("nacos: DataID is required")
	}
	if len(cfg.Servers) == 0 {
		return nil, fmt.Errorf("nacos: at least one server is required")
	}
	return &Source{cfg: cfg}, nil
}

func (s *Source) ensureClient() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil {
		return nil
	}
	srvs := make([]constant.ServerConfig, 0, len(s.cfg.Servers))
	for _, sv := range s.cfg.Servers {
		ctxPath := sv.ContextPath
		if ctxPath == "" {
			ctxPath = "/nacos"
		}
		scheme := sv.Scheme
		if scheme == "" {
			scheme = "http"
		}
		srvs = append(srvs, constant.ServerConfig{
			IpAddr:      sv.IPAddr,
			Port:        sv.Port,
			ContextPath: ctxPath,
			Scheme:      scheme,
		})
	}
	cc := constant.ClientConfig{
		NamespaceId:         s.cfg.Client.NamespaceID,
		Username:            s.cfg.Client.Username,
		Password:            s.cfg.Client.Password,
		TimeoutMs:           or64(s.cfg.Client.TimeoutMs, 5000),
		NotLoadCacheAtStart: true,
		LogLevel:            orStr(s.cfg.Client.LogLevel, "warn"),
		LogDir:              s.cfg.Client.LogDir,
		CacheDir:            s.cfg.Client.CacheDir,
	}
	cli, err := clients.NewConfigClient(vo.NacosClientParam{
		ClientConfig:  &cc,
		ServerConfigs: srvs,
	})
	if err != nil {
		return fmt.Errorf("nacos: build client: %w", err)
	}
	s.client = cli
	return nil
}

// ID returns "nacos:<namespace>/<group>/<dataId>".
func (s *Source) ID() string {
	return fmt.Sprintf("nacos:%s/%s/%s", s.cfg.Client.NamespaceID, s.cfg.Group, s.cfg.DataID)
}

// Load fetches the configuration once.
func (s *Source) Load(_ context.Context) (map[string]any, error) {
	if err := s.ensureClient(); err != nil {
		return nil, err
	}
	content, err := s.client.GetConfig(vo.ConfigParam{
		DataId: s.cfg.DataID,
		Group:  s.cfg.Group,
	})
	if err != nil {
		return nil, fmt.Errorf("nacos: get %s/%s: %w", s.cfg.Group, s.cfg.DataID, err)
	}
	if content == "" {
		return map[string]any{}, nil
	}
	c, err := codec.Get(s.cfg.Format)
	if err != nil {
		return nil, err
	}
	return c.Unmarshal([]byte(content))
}

// Watch registers a long-poll listener.
func (s *Source) Watch(ctx context.Context) (<-chan configx.Event, error) {
	if err := s.ensureClient(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, configx.ErrSourceClosed
	}
	if s.listened {
		out := s.out
		s.mu.Unlock()
		return out, nil
	}
	s.out = make(chan configx.Event, 1)
	s.listened = true
	out := s.out
	s.mu.Unlock()

	c, err := codec.Get(s.cfg.Format)
	if err != nil {
		return nil, err
	}
	err = s.client.ListenConfig(vo.ConfigParam{
		DataId: s.cfg.DataID,
		Group:  s.cfg.Group,
		OnChange: func(namespace, group, dataId, data string) {
			m, err := c.Unmarshal([]byte(data))
			if err != nil {
				return
			}
			select {
			case out <- configx.Event{Source: s.ID(), Data: m}:
			case <-ctx.Done():
			default:
			}
		},
	})
	if err != nil {
		return nil, fmt.Errorf("nacos: listen: %w", err)
	}

	// Stop listening when the parent context cancels.
	go func() {
		<-ctx.Done()
		_ = s.client.CancelListenConfig(vo.ConfigParam{
			DataId: s.cfg.DataID, Group: s.cfg.Group,
		})
		s.mu.Lock()
		if s.out != nil {
			close(s.out)
			s.out = nil
		}
		s.listened = false
		s.mu.Unlock()
	}()

	return out, nil
}

// Close cancels the listener (if any) and marks the source as closed.
func (s *Source) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.listened && s.client != nil {
		_ = s.client.CancelListenConfig(vo.ConfigParam{
			DataId: s.cfg.DataID, Group: s.cfg.Group,
		})
	}
	if s.out != nil {
		close(s.out)
		s.out = nil
	}
	return nil
}

func or64(v, def uint64) uint64 {
	if v == 0 {
		return def
	}
	return v
}

func orStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
