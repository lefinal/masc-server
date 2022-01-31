package logging

import (
	"context"
	"github.com/sirupsen/logrus"
)

// Loggers.
var (
	// AppLogger is the main app.App logger.
	AppLogger *logrus.Entry
	// CommunicationFailLogger is the logger for failed communication.
	CommunicationFailLogger *logrus.Entry
	// DBLogger is used for stuff regarding the database connection.
	DBLogger *logrus.Entry
	// GamesLogger is the logger for package games.
	GamesLogger *logrus.Entry
	// GatekeepingLogger is the logger for gatekeeping.
	GatekeepingLogger *logrus.Entry
	// MessageLogger is used for all incoming and outgoing messages.
	MessageLogger *logrus.Entry
	// ActingLogger is the logger for acting.
	ActingLogger *logrus.Entry
	// SubscriptionManagerLogger is used in acting package for managing
	// subscriptions.
	SubscriptionManagerLogger *logrus.Entry
	// LightingLogger is used for all stuff regarding lighting.
	LightingLogger *logrus.Entry
	// LightSwitchLogger is used for all stuff regarding light switches.
	LightSwitchLogger *logrus.Entry
	// LogPublishLogger is used for stuff regarding publishing logs.
	LogPublishLogger *logrus.Entry
	// NoPublishLogger is for internal stuff and does NOT call the infoLogHook.
	NoPublishLogger *logrus.Entry
	// WebServerLogger is used for all stuff regarding web servers.
	WebServerLogger *logrus.Entry
	// WSLogger is used for all stuff regarding websocket connections.
	WSLogger *logrus.Entry
	// MQTTLogger is the logger for all MQTT stuff.
	MQTTLogger *logrus.Entry
	// MQTTMessageLogger is the logger for incoming and outgoing MQTT messages.
	MQTTMessageLogger *logrus.Entry
)

// infoLogHook is a logrus.Hook. It is created when SubscribeLogEntries is called.
type infoLogHook struct {
	ctx     context.Context
	entries chan *logrus.Entry
}

func (h *infoLogHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.InfoLevel, logrus.WarnLevel, logrus.ErrorLevel}
}

func (h *infoLogHook) Fire(entry *logrus.Entry) error {
	// Check if no publish is desired.
	if isNoPublishRaw, ok := entry.Data["no_publish"]; ok {
		isNoPublish, ok := isNoPublishRaw.(bool)
		if !ok {
			go AppLogger.WithField("entry_message", entry.Message).
				Warn("log entry used no_publish field without bool value type.")
			return nil
		}
		if isNoPublish {
			// Skip publish.
			return nil
		}
	}
	select {
	case <-h.ctx.Done():
		return h.ctx.Err()
	case h.entries <- entry:
	}
	return nil
}

// SubscribeLogEntries creates a channel for logrus entries that receives all
// entries with level info and above. You MUST read from this channel.
func SubscribeLogEntries(ctx context.Context, logger *logrus.Logger) <-chan *logrus.Entry {
	h := &infoLogHook{
		ctx:     ctx,
		entries: make(chan *logrus.Entry, 256),
	}
	logger.AddHook(h)
	return h.entries
}

// ApplyToGlobalLoggers initializes the global loggers with the given
// logrus.Logger.
func ApplyToGlobalLoggers(logger *logrus.Logger) {
	ActingLogger = logger.WithField("topic", "acting")
	AppLogger = logger.WithField("topic", "app")
	CommunicationFailLogger = logger.WithField("topic", "communication-fail")
	DBLogger = logger.WithField("topic", "db")
	GamesLogger = logger.WithField("topic", "games")
	GatekeepingLogger = logger.WithField("topic", "gatekeeping")
	NoPublishLogger = logger.WithField("topic", "logging-internal").WithField("no_publish", true)
	LightingLogger = logger.WithField("topic", "lighting")
	LightSwitchLogger = logger.WithField("topic", "light-switches")
	LogPublishLogger = logger.WithField("topic", "log-publish")
	MessageLogger = logger.WithField("topic", "message")
	SubscriptionManagerLogger = logger.WithField("topic", "subscription-manager")
	WebServerLogger = logger.WithField("topic", "web-server")
	WSLogger = logger.WithField("topic", "ws")
	MQTTLogger = logger.WithField("topic", "mqtt")
	MQTTMessageLogger = logger.WithField("topic", "mqtt_message")
}
