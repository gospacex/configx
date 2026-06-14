package loader

import (
	"os"
	"testing"
)

func TestViperLoader(t *testing.T) {
	content := `{"key": "value"}`
	tmpfile, _ := os.CreateTemp("", "config*.json")
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte(content))
	tmpfile.Close()

	loader, err := NewViperLoader(tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	data, err := loader.Load("key")
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if string(data) != `"value"` {
		t.Errorf("expected '\"value\"', got '%s'", string(data))
	}
}