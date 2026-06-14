// Package file implements a configx Source that loads its data from a single
// file on disk and (optionally) watches it for changes using fsnotify.
package file

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/gospacex/configx"
	"github.com/gospacex/configx/codec"
)

// Option configures a Source.
type Option func(*Source)

// WithFormat overrides the codec format inferred from the file extension.
func WithFormat(format string) Option {
	return func(s *Source) { s.format = format }
}

// WithWatch enables/disables fsnotify-based hot reloading. Default: true.
func WithWatch(enable bool) Option {
	return func(s *Source) { s.watchEnabled = enable }
}

// WithDebounce sets the minimum interval between two reload events that
// fsnotify can produce. Helps avoid double-fire on editors that write
// atomically. Default: 100ms.
func WithDebounce(d time.Duration) Option {
	return func(s *Source) { s.debounce = d }
}

// WithOptional marks the file as optional: if it doesn't exist at Load time
// the Source returns an empty map instead of an error.
func WithOptional(optional bool) Option {
	return func(s *Source) { s.optional = optional }
}

// Source loads a configx tree from a file on disk.
type Source struct {
	path         string
	format       string
	watchEnabled bool
	optional     bool
	debounce     time.Duration

	mu      sync.Mutex
	watcher *fsnotify.Watcher
	out     chan configx.Event
	closed  bool
}

// New constructs a Source pointing at path.
func New(path string, opts ...Option) *Source {
	s := &Source{
		path:         path,
		watchEnabled: true,
		debounce:     100 * time.Millisecond,
	}
	for _, o := range opts {
		o(s)
	}
	if s.format == "" {
		s.format = strings.TrimPrefix(filepath.Ext(path), ".")
	}
	return s
}

// ID returns "file:<absolute path>".
func (s *Source) ID() string {
	abs, err := filepath.Abs(s.path)
	if err != nil {
		abs = s.path
	}
	return "file:" + abs
}

// Load reads, decodes, and returns the file contents.
func (s *Source) Load(_ context.Context) (map[string]any, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && s.optional {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("file source %q: %w", s.path, err)
	}
	c, err := codec.Get(s.format)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", configx.ErrUnknownFormat, s.format)
	}
	return c.Unmarshal(data)
}

// Watch starts fsnotify on the file's parent directory (so atomic renames
// from editors are detected) and emits an Event whenever the file changes.
func (s *Source) Watch(ctx context.Context) (<-chan configx.Event, error) {
	if !s.watchEnabled {
		return nil, nil
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, configx.ErrSourceClosed
	}
	if s.out != nil {
		out := s.out
		s.mu.Unlock()
		return out, nil
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	// Watch the directory so we still get events after editors do
	// write-tempfile + rename, which removes our inode.
	dir := filepath.Dir(s.path)
	if err := w.Add(dir); err != nil {
		_ = w.Close()
		s.mu.Unlock()
		return nil, err
	}
	s.watcher = w
	s.out = make(chan configx.Event, 1)
	out := s.out
	s.mu.Unlock()

	go s.loop(ctx, w, out)
	return out, nil
}

func (s *Source) loop(ctx context.Context, w *fsnotify.Watcher, out chan configx.Event) {
	defer func() {
		s.mu.Lock()
		// only close if we are still the active channel (Close may have already nilled it)
		if s.out == out {
			close(out)
			s.out = nil
		}
		s.mu.Unlock()
	}()

	target, _ := filepath.Abs(s.path)
	var timer *time.Timer
	fire := func() {
		data, err := s.Load(ctx)
		if err != nil {
			return // swallow load errors; user's Watch callback won't see partial data
		}
		select {
		case out <- configx.Event{Source: s.ID(), Data: data}:
		case <-ctx.Done():
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			abs, _ := filepath.Abs(ev.Name)
			if abs != target {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(s.debounce, fire)
		case _, ok := <-w.Errors:
			if !ok {
				return
			}
		}
	}
}

// Close stops watching and releases the underlying fsnotify watcher.
// Closing is idempotent. After Close, Watch returns ErrSourceClosed.
func (s *Source) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	w := s.watcher
	s.watcher = nil
	s.mu.Unlock()
	if w != nil {
		// Closing the watcher closes its Events channel, which causes the
		// loop goroutine to return and close `out` exactly once via its defer.
		return w.Close()
	}
	return nil
}
