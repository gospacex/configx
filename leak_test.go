package configx_test

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain catches goroutine leaks across the whole package's tests.
// Any background watcher / Source goroutine that fails to exit on Close
// will fail the build here.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
