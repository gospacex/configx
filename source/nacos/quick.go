package nacos

import (
	"fmt"
	"net"
	"strconv"
)

// Quick assembles a Source from the minimum viable inputs: an "ip:port"
// endpoint string, namespace, group, and dataId. Format defaults to "yaml".
//
//	src, err := nacos.Quick("127.0.0.1:8848", "public", "DEFAULT_GROUP", "app.yaml")
//
// For multi-server, authentication, custom timeouts, or non-YAML formats,
// fall back to nacos.New(nacos.Config{...}).
func Quick(endpoint, namespace, group, dataID string) (*Source, error) {
	host, port, err := splitHostPort(endpoint)
	if err != nil {
		return nil, err
	}
	return New(Config{
		Servers: []ServerConfig{{IPAddr: host, Port: port}},
		Client:  ClientConfig{NamespaceID: namespace},
		Group:   group,
		DataID:  dataID,
	})
}

// MustQuick is Quick + panic. Use only at program init.
func MustQuick(endpoint, namespace, group, dataID string) *Source {
	s, err := Quick(endpoint, namespace, group, dataID)
	if err != nil {
		panic(err)
	}
	return s
}

func splitHostPort(s string) (string, uint64, error) {
	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		return "", 0, fmt.Errorf("nacos: invalid endpoint %q, want host:port: %w", s, err)
	}
	port, err := strconv.ParseUint(portStr, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("nacos: invalid port in %q: %w", s, err)
	}
	return host, port, nil
}
