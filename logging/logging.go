package logging

import "github.com/sirupsen/logrus"

// Loggers.
var (
	// AppLogger is the main app.App logger.
	AppLogger = logrus.New().WithField("topic", "app")
	// DBLogger is used for stuff regarding the database connection.
	DBLogger = logrus.New().WithField("topic", "db")
	// GamesLogger is the logger for package games.
	GamesLogger = logrus.New().WithField("topic", "games")
	// GatekeepingLogger is the logger for gatekeeping.
	GatekeepingLogger = logrus.New().WithField("topic", "gatekeeping")
	// MessageLogger is used for all incoming and outgoing messages.
	messageLogger = logrus.New()
	// ActingLogger is the logger for acting.
	ActingLogger = logrus.New().WithField("topic", "acting")
	// SubscriptionManagerLogger is used in acting package for managing
	// subscriptions.
	SubscriptionManagerLogger = logrus.New().WithField("topic", "subscription-manager")
	// LightingLogger is used for all stuff regarding lighting.
	LightingLogger = logrus.New().WithField("topic", "lighting")
	// WebServerLogger is used for all stuff regarding web servers.
	WebServerLogger = logrus.New().WithField("topic", "web-server")
	// WSLogger is used for all stuff regarding websocket connections.
	WSLogger = logrus.New().WithField("topic", "ws")
)

func SetMessageLogger(logger *logrus.Logger) {
	messageLogger = logger
}

func MessageLogger() *logrus.Entry {
	return messageLogger.WithField("topic", "message")
}
