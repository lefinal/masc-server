package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/LeFinal/masc-server/app"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Flags.
	configPath := flag.String("config", "config.json", "Path to the config file.")
	flag.Parse()
	// Read config.
	mascConfig, err := readConfig(*configPath)
	if err != nil {
		logrus.Error(errors.Wrap(err, "read config", nil))
		return
	}
	a := app.NewApp(mascConfig)
	// TODO: From config.
	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)
	logging.SetLogger(logger)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := a.Boot(ctx)
		if err != nil {
			logrus.Error(errors.Wrap(err, "boot", nil))
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

// readConfig reads and parses the app.Config in the given filepath.
func readConfig(filepath string) (app.Config, error) {
	// Open config file.
	configFile, err := os.Open(filepath)
	if err != nil {
		return app.Config{}, errors.Error{
			Code:    errors.ErrFatal,
			Err:     err,
			Message: "open config file",
		}
	}
	// Read content.
	byteValue, err := ioutil.ReadAll(configFile)
	if err != nil {
		_ = configFile.Close()
		return app.Config{}, errors.Error{
			Code:    errors.ErrFatal,
			Err:     err,
			Message: "read content of config file",
		}
	}
	// Parse config.
	var config app.Config
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		_ = configFile.Close()
		return app.Config{}, errors.Error{
			Code:    errors.ErrFatal,
			Err:     err,
			Message: "parse config file",
		}
	}
	// Close config.
	err = configFile.Close()
	if err != nil {
		return app.Config{}, errors.Error{
			Code:    errors.ErrFatal,
			Err:     err,
			Message: "close config file",
		}
	}
	return config, nil
}
