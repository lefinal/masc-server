package logging

import "github.com/sirupsen/logrus"

// Loggers.
var (
	// GamesLogger is the logger for package games.
	GamesLogger *logrus.Logger = logrus.New()
	// GatekeepingLogger is the logger for gatekeeping.
	GatekeepingLogger *logrus.Logger = logrus.New()
	// ActingLogger is the logger for acting.
	ActingLogger *logrus.Logger = logrus.New()
	// SubscriptionManagerLogger is used in acting package for managing
	// subscriptions.
	SubscriptionManagerLogger *logrus.Logger = logrus.New()
	// LightingLogger is used for all stuff regarding lighting.
	LightingLogger *logrus.Logger = logrus.New()
	// WebServerLogger is used for all stuff regarding web servers.
	WebServerLogger *logrus.Logger = logrus.New()
	// WSLogger is used for all stuff regarding websocket connections.
	WSLogger *logrus.Logger = logrus.New()
)
