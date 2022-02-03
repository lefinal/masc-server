package logging

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

const topicField = "logger_type"

// Loggers.
var (
	// AppLogger is the main app.App logger.
	AppLogger *zap.Logger
	// CommunicationFailLogger is the logger for failed communication.
	CommunicationFailLogger *zap.Logger
	// DBLogger is used for stuff regarding the database connection.
	DBLogger *zap.Logger
	// GamesLogger is the logger for package games.
	GamesLogger *zap.Logger
	// GatekeepingLogger is the logger for gatekeeping.
	GatekeepingLogger *zap.Logger
	// MASCMessageLogger is used for all incoming and outgoing MASC messages.
	MASCMessageLogger *zap.Logger
	// ActingLogger is the logger for acting.
	ActingLogger *zap.Logger
	// SubscriptionManagerLogger is used in acting package for managing
	// subscriptions.
	SubscriptionManagerLogger *zap.Logger
	// LightingLogger is used for all stuff regarding lighting.
	LightingLogger *zap.Logger
	// LightSwitchLogger is used for all stuff regarding light switches.
	LightSwitchLogger *zap.Logger
	// LogPublishLogger is used for stuff regarding publishing logs.
	LogPublishLogger *zap.Logger
	// NoPublishLogger is for internal stuff and does NOT call the infoLogHook.
	NoPublishLogger *zap.Logger
	// WebServerLogger is used for all stuff regarding web servers.
	WebServerLogger *zap.Logger
	// WSLogger is used for all stuff regarding websocket connections.
	WSLogger *zap.Logger
	// MQTTLogger is the logger for all MQTT stuff.
	MQTTLogger *zap.Logger
	// MQTTMessageLogger is the logger for incoming and outgoing MQTT messages.
	MQTTMessageLogger *zap.Logger
	// PeriodicStackLogger is the logger for logging periodic stack traces in
	// order to get debug information regarding deadlocks.
	PeriodicStackLogger *zap.Logger
)

// ApplyToGlobalLoggers initializes the global loggers with the given
// zap.Logger.
func ApplyToGlobalLoggers(logger *zap.Logger) {
	ActingLogger = logger.Named("acting")
	AppLogger = logger.Named("app")
	CommunicationFailLogger = logger.Named("communication-fail")
	DBLogger = logger.Named("db")
	GamesLogger = logger.Named("games")
	GatekeepingLogger = logger.Named("gatekeeping")
	NoPublishLogger = logger.Named("logging-internal").With(zap.Bool("no_publish", true))
	LightingLogger = logger.Named("lighting")
	LightSwitchLogger = logger.Named("light-switches")
	LogPublishLogger = logger.Named("log-publish")
	MASCMessageLogger = logger.Named("masc-message")
	SubscriptionManagerLogger = logger.Named("subscription-manager")
	WebServerLogger = logger.Named("web-server")
	WSLogger = logger.Named("ws")
	MQTTLogger = logger.Named("mqtt")
	MQTTMessageLogger = logger.Named("mqtt-message")
	PeriodicStackLogger = logger.Named("periodic-stack-trace").With(zap.Bool("no_publish", true))
}

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
