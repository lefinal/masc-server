package web_server

import (
	"context"
	"github.com/LeFinal/masc-server/ws"
)

// PopulateRoutes populates the WebServer with the routes.
func (server *WebServer) PopulateRoutes(hub *ws.Hub, wsCtx context.Context) {
	// Websocket stuff.
	server.router.HandleFunc("/ws", ws.HandleWS(hub, wsCtx))
	// API stuff.
	apiRouter := server.router.PathPrefix("/api/v1").Subrouter()
	// Enable logging.
	apiRouter.Use(loggingMiddleware)
	// Disable caching.
	apiRouter.Use(noCacheMiddleware)
}
