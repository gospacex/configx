package apollo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_JSON(t *testing.T) {
	content := `{"db.host": "localhost", "db.port": 3306, "app.name": "test"}`
	data, err := parseConfig(content, "application")
	require.NoError(t, err)
	assert.Equal(t, "localhost", data["db.host"])
	assert.Equal(t, float64(3306), data["db.port"])
	assert.Equal(t, "test", data["app.name"])
}

func TestParseConfig_YAML(t *testing.T) {
	content := `
db:
  host: localhost
  port: 3306
app:
  name: test
`
	data, err := parseConfig(content, "application")
	require.NoError(t, err)
	// YAML nested maps are flattened by flattenMap
	assert.Equal(t, "localhost", data["db.host"])
	// YAML parser returns int for whole numbers
	assert.Equal(t, 3306, data["db.port"])
	assert.Equal(t, "test", data["app.name"])
}

func TestParseConfig_Empty(t *testing.T) {
	data, err := parseConfig("", "application")
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestParseConfig_Invalid(t *testing.T) {
	content := `not json or yaml at all: [`
	data, err := parseConfig(content, "application")
	require.NoError(t, err)
	assert.Equal(t, content, data["content"])
}

func TestFlattenMap(t *testing.T) {
	input := map[string]interface{}{
		"db": map[string]interface{}{
			"host": "localhost",
			"port": float64(3306),
		},
		"app": "myapp",
	}
	result := make(map[string]interface{})
	flattenMap(input, "", result)
	assert.Equal(t, "localhost", result["db.host"])
	assert.Equal(t, float64(3306), result["db.port"])
	assert.Equal(t, "myapp", result["app"])
}

func TestNew_MissingAppID(t *testing.T) {
	_, err := New(&ApolloConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app_id is required")
}

func TestNew_MissingIP(t *testing.T) {
	_, err := New(&ApolloConfig{AppID: "my-app"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ip is required")
}

func TestFlattenMap_NestedArray(t *testing.T) {
	input := map[string]interface{}{
		"tags": []interface{}{"a", "b", "c"},
	}
	result := make(map[string]interface{})
	flattenMap(input, "", result)
	assert.Equal(t, []string{"a", "b", "c"}, result["tags"])
}
