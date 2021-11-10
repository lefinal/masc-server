package main

import (
	"context"
	"github.com/LeFinal/masc-server/app"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// TODO: Load config from file.
	a := app.NewApp(app.Config{
		DBConn:        "postgres://masc:masc@localhost:5432/masc",
		WebsocketAddr: ":8080",
	})
	// TODO: From config.
	messageLogger := logrus.New()
	messageLogger.SetLevel(logrus.TraceLevel)
	logging.SetMessageLogger(messageLogger)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := a.Boot(ctx)
		if err != nil {
			logrus.Error(errors.Wrap(err, "boot"))
		}
	}()
	awaitTerminateSignal()
	cancel()
}

// awaitTerminateSignal waits until a terminate signal is received.
func awaitTerminateSignal() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
}
