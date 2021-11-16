package app

import (
	"context"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/device_management"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/gatekeeping"
	"github.com/LeFinal/masc-server/lighting"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/stores"
	"github.com/LeFinal/masc-server/web_server"
	"github.com/LeFinal/masc-server/ws"
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
	// Connect database.
	if db, err := connectDB(app.config.DBConn, defaultMaxDBConnections); err != nil {
		return errors.Wrap(err, "connect database")
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
		return errors.Wrap(err, "load known fixtures for lighting manager")
	}
	// Create web server.
	if webServer, err := web_server.NewWebServer(web_server.Config{
		ServeAddr:    app.config.WebsocketAddr,
		WriteTimeout: 1024,
		ReadTimeout:  1024,
	}); err != nil {
		return errors.Wrap(err, "create web server")
	} else {
		app.webServer = webServer
	}
	// Create main handlers.
	app.mainHandlers = mainHandlers{
		deviceManagement:  device_management.NewDeviceManagementHandlers(app.agency, app.gatekeeper),
		fixtureManagement: lighting.NewManagementHandlers(app.agency, app.lightingManager),
		fixtureProviders:  lighting.NewFixtureProviderHandlers(app.agency, app.lightingManager),
	}
	// Boot everything.
	if err := app.gatekeeper.WakeUpAndProtect(app.agency); err != nil {
		return errors.Wrap(err, "wake up gatekeeper and protect")
	}
	go app.mainHandlers.Run(ctx)
	go app.wsHub.Run(ctx)
	if err := app.agency.Open(); err != nil {
		return errors.Wrap(err, "open agency")
	}
	app.webServer.PopulateRoutes(app.wsHub, ctx)
	go func() {
		err := app.webServer.Run(ctx)
		if err != nil {
			errors.Log(logging.AppLogger, errors.Wrap(err, "run web server"))
			return
		}
	}()
	// Wait for exit.
	<-ctx.Done()
	if err := app.gatekeeper.Retire(); err != nil {
		return errors.Wrap(err, "retire gatekeeper")
	}
	if err := app.agency.Close(); err != nil {
		return errors.Wrap(err, "close agency")
	}
	return nil
}

// mainHandlers includes main actor handlers.
type mainHandlers struct {
	deviceManagement  *device_management.DeviceManagementHandlers
	fixtureManagement *lighting.ManagementHandlers
	fixtureProviders  *lighting.FixtureProviderHandlers
	// wg waits for all running handlers.
	wg sync.WaitGroup
}

func (mh *mainHandlers) Run(ctx context.Context) {
	go mh.deviceManagement.Run(ctx)
	go mh.fixtureManagement.Run(ctx)
	go mh.fixtureProviders.Run(ctx)
}
