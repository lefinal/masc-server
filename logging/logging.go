package logging

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

const topicField = "logger_type"

// LogEntry is a container for logs that can be handled by other components.
type LogEntry struct {
	// Time is the timestamp the log entry was created.
	Time time.Time `json:"time"`
	// Message is the log entry message.
	Message string `json:"message"`
	// Level is the log level of the entry.
	Level zapcore.Level `json:"level"`
	// LoggerName is the name of the logger.
	LoggerName string `json:"logger_name"`
	// Fields are the set fields for the log entry.
	Fields map[string]interface{} `json:"fields"`
}

// noPublishOmitCore wraps a zapcore.Core with a custom write function that
// sends entries to the given publish channel.
type noPublishOmitCore struct {
	zapcore.Core
	fields  []zap.Field
	ctx     context.Context
	publish chan<- LogEntry
}

// NewNoPublishOmitCore creates a zapcore.Core that publishes log entries with
// level zap.InfoLevel or above not containing the field no_publish to the
// returned buffered LogEntry channel.
//
// Do NOT use With on this zapcore.Core!
func NewNoPublishOmitCore(ctx context.Context) (zapcore.Core, <-chan LogEntry) {
	publish := make(chan LogEntry, 256)
	core := &noPublishOmitCore{
		Core:    zapcore.NewNopCore(),
		ctx:     ctx,
		publish: publish,
	}
	return core, publish
}

func (c *noPublishOmitCore) With(fields []zap.Field) zapcore.Core {
	return &noPublishOmitCore{
		Core:    c.Core.With(fields),
		fields:  append(c.fields, fields...),
		ctx:     c.ctx,
		publish: c.publish,
	}
}

func (c *noPublishOmitCore) Enabled(level zapcore.Level) bool {
	return level >= zap.InfoLevel
}

func (c *noPublishOmitCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(entry.Level) {
		return ce
	}
	return ce.AddCore(entry, c)
}

func (c *noPublishOmitCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Bake fields and check for no_publish field.
	fields = append(fields, c.fields...)
	enc := zapcore.NewMapObjectEncoder()
	for _, field := range fields {
		if field.Key == "no_publish" {
			// Omit entry.
			return nil
		}
		field.AddTo(enc)
	}
	// Log entry.
	select {
	case <-c.ctx.Done():
		return nil
	case c.publish <- LogEntry{
		Time:       entry.Time,
		LoggerName: entry.LoggerName,
		Message:    entry.Message,
		Level:      entry.Level,
		Fields:     enc.Fields,
	}:
	}
	return nil
}
