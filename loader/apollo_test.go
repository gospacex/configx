package loader

import (
	"container/list"
	"testing"

	"github.com/apolloconfig/agollo/v4/agcache"
	"github.com/apolloconfig/agollo/v4/storage"
)

type mockApolloCache struct{}

func (m *mockApolloCache) Set(key string, value interface{}, expireSeconds int) error {
	return nil
}
func (m *mockApolloCache) EntryCount() int64 { return 0 }
func (m *mockApolloCache) Get(key string) (interface{}, error) {
	return nil, nil
}
func (m *mockApolloCache) Del(key string) bool                       { return false }
func (m *mockApolloCache) Range(f func(key, value interface{}) bool) {}
func (m *mockApolloCache) Clear()                                    {}

type mockApolloClient struct {
	values map[string]string
}

func (m *mockApolloClient) GetConfig(namespace string) *storage.Config        { return nil }
func (m *mockApolloClient) GetConfigAndInit(namespace string) *storage.Config { return nil }
func (m *mockApolloClient) GetConfigCache(namespace string) agcache.CacheInterface {
	return &mockApolloCache{}
}
func (m *mockApolloClient) GetDefaultConfigCache() agcache.CacheInterface {
	return &mockApolloCache{}
}
func (m *mockApolloClient) GetApolloConfigCache() agcache.CacheInterface {
	return &mockApolloCache{}
}
func (m *mockApolloClient) GetValue(key string) string { return m.values[key] }
func (m *mockApolloClient) GetStringValue(key string, defaultValue string) string {
	if v, ok := m.values[key]; ok {
		return v
	}
	return defaultValue
}
func (m *mockApolloClient) GetIntValue(key string, defaultValue int) int { return defaultValue }
func (m *mockApolloClient) GetFloatValue(key string, defaultValue float64) float64 {
	return defaultValue
}
func (m *mockApolloClient) GetBoolValue(key string, defaultValue bool) bool { return defaultValue }
func (m *mockApolloClient) GetStringSliceValue(key string, defaultValue []string) []string {
	return defaultValue
}
func (m *mockApolloClient) GetIntSliceValue(key string, defaultValue []int) []int {
	return defaultValue
}
func (m *mockApolloClient) AddChangeListener(listener storage.ChangeListener)    {}
func (m *mockApolloClient) RemoveChangeListener(listener storage.ChangeListener) {}
func (m *mockApolloClient) GetChangeListeners() *list.List                       { return list.New() }
func (m *mockApolloClient) UseEventDispatch()                                    {}
func (m *mockApolloClient) Close()                                               {}

func TestApolloLoader_Load_Found(t *testing.T) {
	client := &mockApolloClient{
		values: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"":     "empty_key_value",
		},
	}
	loader := &ApolloLoader{client: client}

	data, err := loader.Load("key1")
	if err != nil {
		t.Fatalf("Load(%q) unexpected error: %v", "key1", err)
	}
	if string(data) != "value1" {
		t.Errorf("Load(%q) = %q, want %q", "key1", string(data), "value1")
	}

	data2, err := loader.Load("key2")
	if err != nil {
		t.Fatalf("Load(%q) unexpected error: %v", "key2", err)
	}
	if string(data2) != "value2" {
		t.Errorf("Load(%q) = %q, want %q", "key2", string(data2), "value2")
	}
}

func TestApolloLoader_Load_NotFound(t *testing.T) {
	client := &mockApolloClient{values: map[string]string{}}
	loader := &ApolloLoader{client: client}

	data, err := loader.Load("nonexistent")
	if err != nil {
		t.Errorf("Load nonexistent key: unexpected error %v", err)
	}
	if len(data) != 0 {
		t.Errorf("Load nonexistent key = %q, want empty", string(data))
	}
}

func TestApolloLoader_Load_DefaultValue(t *testing.T) {
	client := &mockApolloClient{values: map[string]string{}}
	loader := &ApolloLoader{client: client}

	data, err := loader.Load("missing_key")
	if err != nil {
		t.Fatalf("Load with default: unexpected error %v", err)
	}
	if len(data) != 0 {
		t.Errorf("Load missing_key = %q, want empty string", string(data))
	}
}

func TestApolloLoader_Watch_Registration(t *testing.T) {
	client := &mockApolloClient{values: map[string]string{"key": "value"}}
	loader := &ApolloLoader{client: client}

	loader.Watch("key", func(data []byte) {
		t.Log("callback invoked with data:", string(data))
	})
}

func TestApolloLoader_Close(t *testing.T) {
	client := &mockApolloClient{values: map[string]string{}}
	loader := &ApolloLoader{client: client}

	err := loader.Close()
	if err != nil {
		t.Errorf("Close() unexpected error: %v", err)
	}
}
