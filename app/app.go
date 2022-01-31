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
	"github.com/sirupsen/logrus"
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
	logger := logrus.New()
	logLevel := logrus.InfoLevel
	if app.config.LogLevel.Valid {
		logLevel, err = logrus.ParseLevel(app.config.LogLevel.String)
		if err != nil {
			return errors.NewInternalErrorFromErr(err, "invalid log level",
				errors.Details{"was": app.config.LogLevel.String})
		}
	}
	logger.SetLevel(logLevel)
	logging.ApplyToGlobalLoggers(logger)
	// Connect database.
	if db, err := connectDB(app.config.DBConn, defaultMaxDBConnections); err != nil {
		return errors.Wrap(err, "connect database", nil)
	} else {
		app.mall = stores.NewMall(db)
	}
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
	// Boot everything.
	if err := app.gatekeeper.WakeUpAndProtect(app.agency); err != nil {
		return errors.Wrap(err, "wake up gatekeeper and protect", nil)
	}
	go app.lightingManager.Run(ctx)
	go lightswitch.RunActorReception(ctx, app.agency, app.lightSwitchManager)
	go logpublish.RunActorReception(ctx, app.agency, logging.SubscribeLogEntries(ctx, logger))
	go app.mainHandlers.Run(ctx)
	go app.wsHub.Run(ctx)
	if app.mqttBridge != nil {
		go func() {
			if err := app.mqttBridge.Run(ctx, app.mall, app.gatekeeper); err != nil {
				errors.Log(logging.AppLogger, errors.Wrap(err, "run mqtt bridge", nil))
			}
		}()
	}
	if err := app.agency.Open(); err != nil {
		return errors.Wrap(err, "open agency", nil)
	}
	app.webServer.PopulateRoutes(app.wsHub, ctx)
	go func() {
		err := app.webServer.Run(ctx)
		if err != nil {
			errors.Log(logging.AppLogger, errors.Wrap(err, "run web server", nil))
			return
		}
	}()
	// Wait for exit.
	<-ctx.Done()
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

func (mh *mainHandlers) Run(ctx context.Context) {
	go mh.deviceManagement.Run(ctx)
	// We run the operators first because they need to provide the context for
	// update handlers.
	go mh.fixtureOperators.Run(ctx)
	go mh.fixtureManagement.Run(ctx)
	go mh.fixtureProviders.Run(ctx)
}
