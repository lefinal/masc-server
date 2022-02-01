package app

import (
	"context"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/device_management"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/gatekeeping"
	"github.com/LeFinal/masc-server/lighting"
	"github.com/LeFinal/masc-server/lightswitch"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/logpublish"
	"github.com/LeFinal/masc-server/mqttbridge"
	"github.com/LeFinal/masc-server/stores"
	"github.com/LeFinal/masc-server/web_server"
	"github.com/LeFinal/masc-server/ws"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"sync"
)

// App is a complete MASC server instance.
type App struct {
	// mall provides persistence.
	mall *stores.Mall
	// config is the main config used for the App.
	config Config
	// webServer is used for http requests and websocket connection.
	webServer *web_server.WebServer
	// wsHub is the hub for websocket connections.
	wsHub *ws.Hub
	// gatekeeper handles new connections and device management.
	gatekeeper *gatekeeping.NetGatekeeper
	// agency does the acting.Actor management.
	agency *acting.ProtectedAgency
	// lightingManager is the manager for fixture operation, handling, etc.
	lightingManager *lighting.StoredManager
	// lightSwitchManager is the manager for light switches.
	lightSwitchManager lightswitch.Manager
	// mqttBridge is used for connecting to a MQTT-server.
	mqttBridge mqttbridge.Bridge
	// mainHandlers holds general actor handlers like device management or fixture
	// manager.
	mainHandlers mainHandlers
	publishLog   <-chan logging.LogEntry
}

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
	logger, publishLog := app.setupLogging(ctx, app.config.Log)
	app.publishLog = publishLog
	logging.ApplyToGlobalLoggers(logger)
	defer func(loggerToSync *zap.Logger) {
		_ = logger.Sync()
	}(logger)
	// Boot.
	err = app.boot(ctx)
	if err != nil {
		err = errors.Wrap(err, "boot", nil)
		errors.Log(logging.AppLogger, err)
		return err
	}
	return nil
}

func (app *App) boot(ctx context.Context) error {
	appCtx, shutdown := context.WithCancel(context.Background())
	defer shutdown()
	logging.AppLogger.Warn("booting up")
	// Connect database.
	logging.AppLogger.Debug("connecting to database")
	if db, err := connectDB(app.config.DBConn, defaultMaxDBConnections); err != nil {
		return errors.Wrap(err, "connect database", nil)
	} else {
		app.mall = stores.NewMall(db)
	}
	logging.AppLogger.Debug("database ready")
	logging.AppLogger.Debug("setting up...")
	// Create gatekeeper.
	app.gatekeeper = gatekeeping.NewNetGatekeeper(app.mall)
	// Create websocket hub.
	app.wsHub = ws.NewHub(app.gatekeeper)
	// Create agency.
	app.agency = acting.NewProtectedAgency(app.gatekeeper)
	// Create lighting manager.
	app.lightingManager = lighting.NewStoredManager(app.mall)
	if err := app.lightingManager.LoadKnownFixtures(); err != nil {
		return errors.Wrap(err, "load known fixtures for lighting manager", nil)
	}
	// Create light switch manager.
	app.lightSwitchManager = lightswitch.NewManager(app.mall, app.lightingManager)
	// Create MQTT bridge if address is provided.
	if app.config.MQTTAddr.Valid {
		app.mqttBridge = mqttbridge.NewBridge(mqttbridge.Config{MQTTAddr: app.config.MQTTAddr.String})
	}
	// Create web server.
	if webServer, err := web_server.NewWebServer(web_server.Config{
		ServeAddr:    app.config.WebsocketAddr,
		WriteTimeout: 1024,
		ReadTimeout:  1024,
	}); err != nil {
		return errors.Wrap(err, "create web server", nil)
	} else {
		app.webServer = webServer
	}
	// Create main handlers.
	app.mainHandlers = mainHandlers{
		deviceManagement:  device_management.NewDeviceManagementHandlers(app.agency, app.gatekeeper),
		fixtureManagement: lighting.NewManagementHandlers(app.agency, app.lightingManager),
		fixtureProviders:  lighting.NewFixtureProviderHandlers(app.agency, app.lightingManager),
		fixtureOperators:  lighting.NewFixtureOperatorHandlers(app.agency, app.lightingManager),
	}
	logging.AppLogger.Debug("setup completed. booting...")
	// Boot everything.
	if err := app.gatekeeper.WakeUpAndProtect(app.agency); err != nil {
		return errors.Wrap(err, "wake up gatekeeper and protect", nil)
	}
	go app.lightingManager.Run(appCtx)
	go lightswitch.RunActorReception(appCtx, app.agency, app.lightSwitchManager)
	go logpublish.RunActorReception(appCtx, app.agency, app.publishLog)
	go app.mainHandlers.Run(appCtx)
	go app.wsHub.Run(appCtx)
	if app.mqttBridge != nil {
		go func() {
			if err := app.mqttBridge.Run(appCtx, app.mall, app.gatekeeper); err != nil {
				errors.Log(logging.AppLogger, errors.Wrap(err, "run mqtt bridge", nil))
			}
		}()
	}
	if err := app.agency.Open(); err != nil {
		return errors.Wrap(err, "open agency", nil)
	}
	app.webServer.PopulateRoutes(app.wsHub, appCtx)
	go func() {
		err := app.webServer.Run(appCtx)
		if err != nil {
			errors.Log(logging.AppLogger, errors.Wrap(err, "run web server", nil))
			return
		}
	}()
	logging.AppLogger.Warn("completed issuing boot commands")
	// Wait for exit.
	<-ctx.Done()
	logging.AppLogger.Warn("shutting down")
	if err := app.gatekeeper.Retire(); err != nil {
		return errors.Wrap(err, "retire gatekeeper", nil)
	}
	if err := app.agency.Close(); err != nil {
		return errors.Wrap(err, "close agency", nil)
	}
	return nil
}

// mainHandlers includes main actor handlers.
type mainHandlers struct {
	deviceManagement  *device_management.DeviceManagementHandlers
	fixtureManagement *lighting.ManagementHandlers
	fixtureProviders  *lighting.FixtureProviderHandlers
	fixtureOperators  *lighting.FixtureOperatorHandlers
	// wg waits for all running handlers.
	wg sync.WaitGroup
}

func (app *App) setupLogging(ctx context.Context, config LogConfig) (*zap.Logger, <-chan logging.LogEntry) {
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

func (mh *mainHandlers) Run(ctx context.Context) {
	go mh.deviceManagement.Run(ctx)
	// We run the operators first because they need to provide the context for
	// update handlers.
	go mh.fixtureOperators.Run(ctx)
	go mh.fixtureManagement.Run(ctx)
	go mh.fixtureProviders.Run(ctx)
}
