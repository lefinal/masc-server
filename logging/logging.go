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
)
