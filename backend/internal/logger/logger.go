package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	mu   sync.RWMutex
	inst *slog.Logger
)

func Init(level string, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	var lv slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lv = slog.LevelDebug
	case "warn", "warning":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lv})
	mu.Lock()
	inst = slog.New(h)
	mu.Unlock()
}

func L() *slog.Logger {
	mu.RLock()
	l := inst
	mu.RUnlock()
	if l == nil {
		Init("info", nil)
		return L()
	}
	return l
}
