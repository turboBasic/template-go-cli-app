package logging

import (
	"context"
	"log/slog"
	"strings"
)

type redactingHandler struct {
	handler slog.Handler
	groups  []string
}

func redact(handler slog.Handler) slog.Handler {
	return &redactingHandler{handler: handler}
}

func (h *redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *redactingHandler) Handle(ctx context.Context, record slog.Record) error {
	var safe slog.Record
	record.Attrs(func(attr slog.Attr) bool {
		safe.AddAttrs(redactAttr(attr, h.groups))
		return true
	})
	safe.Time = record.Time
	safe.Level = record.Level
	safe.Message = record.Message
	safe.PC = record.PC
	return h.handler.Handle(ctx, safe)
}

func (h *redactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	safe := make([]slog.Attr, len(attrs))
	for i, attr := range attrs {
		safe[i] = redactAttr(attr, h.groups)
	}
	return &redactingHandler{handler: h.handler.WithAttrs(safe), groups: h.groups}
}

func (h *redactingHandler) WithGroup(name string) slog.Handler {
	groups := h.groups
	if name != "" {
		groups = append(append([]string(nil), groups...), name)
	}
	return &redactingHandler{handler: h.handler.WithGroup(name), groups: groups}
}

func redactAttr(attr slog.Attr, groups []string) slog.Attr {
	attr.Value = attr.Value.Resolve()
	if sensitiveKey(attr.Key) || anySensitiveGroup(groups) {
		attr.Value = slog.StringValue("[REDACTED]")
		return attr
	}
	if attr.Value.Kind() == slog.KindGroup {
		children := append([]slog.Attr(nil), attr.Value.Group()...)
		for i := range children {
			children[i] = redactAttr(children[i], append(groups, attr.Key))
		}
		attr.Value = slog.GroupValue(children...)
	}
	return attr
}

func sensitiveKey(key string) bool {
	normalize := func(value string) string {
		return strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(value))
	}
	key = normalize(key)
	for _, term := range []string{"password", "token", "authorization", "api_key"} {
		if strings.Contains(key, normalize(term)) {
			return true
		}
	}
	return false
}

func anySensitiveGroup(groups []string) bool {
	for _, group := range groups {
		if sensitiveKey(group) {
			return true
		}
	}
	return false
}
