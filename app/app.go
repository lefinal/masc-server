package app

import (
	"context"
	"github.com/lefinal/masc-server/errors"
	"github.com/lefinal/masc-server/logging"
	"github.com/lefinal/masc-server/portal"
	"github.com/lefinal/masc-server/store"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"sync"
)

// App is a complete MASC server instance.
type App struct {
	// mall provides persistence.
	mall *store.Mall
	// config is the main config used for the App.
	config Config
	// portal is used for connecting to the MQTT-server.
	portal     portal.Base
	logger     *zap.Logger
	publishLog <-chan logging.LogEntry
}

// NewApp returns a new App with the given Config that can be booted using
// App.Boot.
func NewApp(config Config) *App {
	return &App{
		config: config,
	}
}

// Boot sets everything up based on the set config and boots.
func (app *App) Boot(ctx context.Context) error {
	// Validate config.
	err := ValidateConfig(app.config)
	if err != nil {
		return errors.Error{
			Code:    errors.ErrFatal,
			Err:     err,
			Message: "invalid config",
		}
	}
	// Setup logger.
	logger, publishLog := setupLogger(ctx, app.config.Log)
	app.logger = logger
	app.publishLog = publishLog
	dbMigrationLogger = logger.Named("db-migration")
	defer func(loggerToSync *zap.Logger) {
		_ = logger.Sync()
	}(logger)
	// Boot.
	err = app.boot(ctx)
	if err != nil {
		err = errors.Wrap(err, "boot", nil)
		errors.Log(app.logger, err)
		return err
	}
	return nil
}

func (app *App) boot(ctx context.Context) error {
	var wg sync.WaitGroup
	appCtx, shutdown := context.WithCancel(context.Background())
	defer shutdown()
	app.logger.Info("booting up")
	// Connect database.
	app.logger.Debug("connecting to database")
	db, err := connectDB(ctx, app.config.DBConn)
	if err != nil {
		return errors.Wrap(err, "connect database", nil)
	}
	app.mall = store.NewMall(app.logger.Named("store"), db)
	app.logger.Debug("database ready")
	app.logger.Debug("setting up...")
	// Create portal.
	app.portal, err = portal.NewBase(app.logger.Named("portal"), portal.Config{MQTTAddr: app.config.MQTTAddr})
	if err != nil {
		return errors.Wrap(err, "new portal base", nil)
	}
	app.logger.Debug("setup completed. booting...")
	// Open portal.
	wg.Add(1)
	portalOpened := make(chan struct{})
	go func() {
		defer wg.Done()
		err := app.portal.Open(appCtx, portalOpened)
		if err != nil {
			errors.Log(app.logger, errors.Wrap(err, "open portal", nil))
			shutdown()
			return
		}
	}()
	// Await portal opened.
	select {
	case <-ctx.Done():
		return errors.NewContextAbortedError("wait for portal opened")
	case <-portalOpened:
	}
	// Create services and run them.
	services, err := createServices(app.config, app.logger, app.portal, app.mall)
	if err != nil {
		return errors.Wrap(err, "create services", nil)
	}
	servicesLifetime, shutdownServices := context.WithCancel(appCtx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer shutdown()
		if err := services.run(servicesLifetime, app.logger.Named("service-runner")); err != nil {
			errors.Log(app.logger, errors.Wrap(err, "run services", nil))
			return
		}
		app.logger.Info("service-runner finished")
	}()
	app.logger.Info("completed issuing boot commands")
	// Wait for exit.
	<-ctx.Done()
	app.logger.Warn("shutting down")
	shutdownServices()
	wg.Wait()
	return nil
}

func setupLogger(ctx context.Context, config LogConfig) (*zap.Logger, <-chan logging.LogEntry) {
	encConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	cores := make([]zapcore.Core, 0)
	// Setup stdout logger with colorful level output.
	stdOutEncConfig := encConfig
	stdOutEncConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cores = append(cores, zapcore.NewCore(
		zapcore.NewConsoleEncoder(stdOutEncConfig),
		zapcore.Lock(os.Stdout),
		zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level >= config.StdoutLogLevel
		})))
	// Setup error logger.
	cores = append(cores, zapcore.NewCore(
		zapcore.NewConsoleEncoder(encConfig),
		zapcore.Lock(os.Stderr),
		zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level >= zap.ErrorLevel
		})))
	// Setup high priority logger.
	if config.HighPriorityOutput.Valid {
		cores = append(cores, zapcore.NewCore(
			zapcore.NewConsoleEncoder(encConfig),
			zapcore.AddSync(&lumberjack.Logger{
				Filename: config.HighPriorityOutput.String,
				MaxSize:  config.MaxSize,
				MaxAge:   config.KeepDays,
			}),
			zap.LevelEnablerFunc(func(level zapcore.Level) bool {
				return level >= zap.WarnLevel
			})))
	}
	// Setup debug logger.
	if config.DebugOutput.Valid {
		cores = append(cores, zapcore.NewCore(
			zapcore.NewConsoleEncoder(encConfig),
			zapcore.AddSync(&lumberjack.Logger{
				Filename: config.DebugOutput.String,
				MaxSize:  config.MaxSize,
				MaxAge:   config.KeepDays,
			}),
			zap.LevelEnablerFunc(func(level zapcore.Level) bool {
				return level >= zap.DebugLevel
			})))
	}
	// Setup publish logger.
	publishLogger, publishLog := logging.NewNoPublishOmitCore(ctx)
	cores = append(cores, publishLogger)
	// Combine.
	logger := zap.New(zapcore.NewTee(cores...))
	return logger, publishLog
}
