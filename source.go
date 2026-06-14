package configx

import "context"

// Event represents a change notification emitted by a Source.
type Event struct {
	// Source is the human readable identifier of the producer (e.g. "file:/etc/app.yaml").
	Source string
	// Data is the freshly loaded payload of the source.
	Data map[string]any
}

// Source is the abstraction for anything that can produce configuration data.
//
// Implementations MUST be safe for concurrent use. Load must be idempotent.
// Watch is optional — sources that cannot push updates should return a nil
// channel and a nil error. The returned channel must be closed by the source
// when Close is invoked.
type Source interface {
	// ID returns a stable, unique identifier for this source instance.
	// It is used in logs and diagnostics.
	ID() string

	// Load fetches the current snapshot of the source.
	Load(ctx context.Context) (map[string]any, error)

	// Watch returns a channel that emits an Event whenever the source's
	// content changes. A nil channel means the source does not support
	// watching. Implementations should debounce rapid successive changes.
	Watch(ctx context.Context) (<-chan Event, error)

	// Close releases any resources held by the source.
	Close() error
}
