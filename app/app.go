package app

import (
	"github.com/LeFinal/masc-server/gatekeeping"
	"github.com/LeFinal/masc-server/web_server"
	"github.com/LeFinal/masc-server/ws"
)

// App is a complete MASC server instance.
type App struct {
	// config is the main config used for the App.
	config Config
	// webServer is used for http requests and websocket connection.
	webServer *web_server.WebServer
	// wsHub is the hub for websocket connections.
	wsHub *ws.Hub
	// gatekeeper handles new connections and device management.
	gatekeeper gatekeeping.Gatekeeper
}
