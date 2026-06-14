package configx

import (
	"fmt"
	"log/slog"
)

// LoggerFunc adapts a function into the Logger interface.
type LoggerFunc func(format string, args ...any)

// Errorf implements Logger.
func (f LoggerFunc) Errorf(format string, args ...any) { f(format, args...) }

// LoggerFromSlog returns a Logger that forwards Errorf calls to the given
// *slog.Logger at slog.LevelError. The format string is rendered via fmt.Sprintf
// and emitted as the "msg" field, preserving structured-log friendliness.
//
//	cfg, _ := configx.New(
//	    configx.WithLogger(configx.LoggerFromSlog(slog.Default())),
//	    ...
//	)
func LoggerFromSlog(l *slog.Logger) Logger {
	if l == nil {
		l = slog.Default()
	}
	return LoggerFunc(func(format string, args ...any) {
		l.Error(fmt.Sprintf(format, args...))
	})
}
