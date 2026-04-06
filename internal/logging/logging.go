package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Setup initialises the global slog.Logger based on environment configuration.
// It returns a cleanup function that should be deferred by the caller.
//
// Environment variables:
//
//	LOG_LEVEL  — debug, info, warn, error (default: info)
//	LOG_FILE   — optional path; logs are written to both stdout and the file
func Setup(levelStr, filePath string) (cleanup func(), err error) {
	level := parseLevel(levelStr)

	var handlers []slog.Handler

	// Always log JSON to stdout (picked up by Container Apps / Docker).
	handlers = append(handlers, slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	var closers []func()

	if filePath != "" {
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		closers = append(closers, func() { _ = f.Close() })
		handlers = append(handlers, slog.NewJSONHandler(f, &slog.HandlerOptions{Level: level}))
	}

	var handler slog.Handler
	if len(handlers) == 1 {
		handler = handlers[0]
	} else {
		handler = &MultiHandler{handlers: handlers}
	}

	slog.SetDefault(slog.New(handler))

	return func() {
		for _, c := range closers {
			c()
		}
	}, nil
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// MultiHandler fans out log records to multiple slog.Handlers.
type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: handlers}
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: handlers}
}

// Writer returns an io.Writer that logs each write as a slog message at the
// given level. Useful for bridging the stdlib log package to slog.
func Writer(level slog.Level) io.Writer {
	return &slogWriter{level: level}
}

type slogWriter struct {
	level slog.Level
}

func (w *slogWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\n")
	slog.Log(context.Background(), w.level, msg)
	return len(p), nil
}
