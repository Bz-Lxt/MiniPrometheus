package logger

import (
	"context"
	"log/slog"
)

type ctxKey struct{}

func With(ctx context.Context, attrs ...any) context.Context {
	return context.WithValue(ctx, ctxKey{}, attrs)
}

func From(ctx context.Context) *slog.Logger {
	l := L()
	if ctx == nil {
		return l
	}
	if attrs, ok := ctx.Value(ctxKey{}).([]any); ok && len(attrs) > 0 {
		return l.With(attrs...)
	}
	return l
}
