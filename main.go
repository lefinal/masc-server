package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/LeFinal/masc-server/app"
	"github.com/LeFinal/masc-server/errors"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	// Flags.
	configPath := flag.String("config", "config.json", "Path to the config file.")
	flag.Parse()
	// Read config.
	mascConfig, err := readConfig(*configPath)
	if err != nil {
		log.Fatalln(errors.Wrap(err, "read config", nil).Error())
	}
	a := app.NewApp(mascConfig)
	appCtx, shutdownApp := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := a.Boot(appCtx)
		if err != nil {
			shutdownApp()
			log.Fatalln(errors.Wrap(err, "boot", nil))
		}
	}()
	awaitTerminateSignal(appCtx)
	shutdownApp()
	wg.Wait()
}

// awaitTerminateSignal waits until a terminate signal is received.
func awaitTerminateSignal(ctx context.Context) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
	case <-signals:
	}
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
