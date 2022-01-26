package logging

import "github.com/sirupsen/logrus"

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
	// WebServerLogger is used for all stuff regarding web servers.
	WebServerLogger *logrus.Entry
	// WSLogger is used for all stuff regarding websocket connections.
	WSLogger *logrus.Entry
	// MQTTLogger is the logger for all MQTT stuff.
	MQTTLogger *logrus.Entry
	// MQTTMessageLogger is the logger for incoming and outgoing MQTT messages.
	MQTTMessageLogger *logrus.Entry
)

func SetLogger(logger *logrus.Logger) {
	AppLogger = logger.WithField("topic", "app")
	CommunicationFailLogger = logger.WithField("topic", "communication-fail")
	DBLogger = logger.WithField("topic", "db")
	GamesLogger = logger.WithField("topic", "games")
	GatekeepingLogger = logger.WithField("topic", "gatekeeping")
	MessageLogger = logger.WithField("topic", "message")
	ActingLogger = logger.WithField("topic", "acting")
	SubscriptionManagerLogger = logger.WithField("topic", "subscription-manager")
	LightingLogger = logger.WithField("topic", "lighting")
	LightSwitchLogger = logger.WithField("topic", "light-switches")
	WebServerLogger = logger.WithField("topic", "web-server")
	WSLogger = logger.WithField("topic", "ws")
	MQTTLogger = logger.WithField("topic", "mqtt")
	MQTTMessageLogger = logger.WithField("topic", "mqtt_message")
}
