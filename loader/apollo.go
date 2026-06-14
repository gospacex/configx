package loader

import (
	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
)

type ApolloLoader struct {
	client agollo.Client
}

func NewApolloLoader(addr string, appID string, cluster string, namespace string, group string) (*ApolloLoader, error) {
	appConfig := &config.AppConfig{
		IP:            addr,
		AppID:         appID,
		Cluster:       cluster,
		NamespaceName: namespace,
	}

	client, err := agollo.StartWithConfig(func() (*config.AppConfig, error) {
		return appConfig, nil
	})
	if err != nil {
		return nil, err
	}
	return &ApolloLoader{client: client}, nil
}

func (l *ApolloLoader) Load(key string) ([]byte, error) {
	val := l.client.GetStringValue(key, "")
	return []byte(val), nil
}

func (l *ApolloLoader) Watch(key string, fn func([]byte)) {
	l.client.AddChangeListener(&changeListener{key: key, fn: fn})
}

func (l *ApolloLoader) Close() error {
	return nil
}

type changeListener struct {
	key string
	fn  func([]byte)
}

func (c *changeListener) OnChange(change *storage.ChangeEvent) {
	if v, ok := change.Changes[c.key]; ok {
		c.fn([]byte(v.NewValue.(string)))
	}
}

func (c *changeListener) OnNewBatchChange(change *storage.ChangeEvent) {
	c.OnChange(change)
}

func (c *changeListener) OnNewestChange(change *storage.FullChangeEvent) {
	changes := make(map[string]*storage.ConfigChange)
	for k, v := range change.Changes {
		if str, ok := v.(string); ok {
			changes[k] = &storage.ConfigChange{NewValue: str}
		}
	}
	c.fn([]byte(changes[c.key].NewValue.(string)))
}