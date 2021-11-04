package web_server

import (
	"context"
	nativeerrors "errors"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"net/http"
	"time"
)

const (
	// DefaultServeAddr is the default address to serve on.
	DefaultServeAddr = ":8080"
	// DefaultWriteTimeout is the default timeout for writing.
	DefaultWriteTimeout = 15 * time.Second
	// DefaultReadTimeout is the default timeout for reading.
	DefaultReadTimeout = 15 * time.Second
)

type WebServer struct {
	config     Config
	httpServer *http.Server
	router     *mux.Router
	running    bool
	stopErr    chan error
}

// Config is the configuration that is used in order to create and run a web
// server.
type Config struct {
	// Address for the web server to listen to.
	ServeAddr string
	// WriteTimeout is the duration in seconds to wait until write fails with a
	// timeout.
	WriteTimeout time.Duration
	// ReadTimeout is the duration in seconds to wait until read fails with a
	// timeout.
	ReadTimeout time.Duration
}

// NewWebServer creates a new WebServer and sets up initial stuff. It expects
// the passed Config to be filled correctly. If you need default values, these
// are exported as DefaultWriteTimeout, DefaultReadTimeout,
// DefaultListRetrievalLimit and DefaultTicketExpireTime. Run it with
// WebServer.Run and do not forget to call WebServer.PopulateRoutes before.
func NewWebServer(config Config) (*WebServer, error) {
	// Setup web server.
	ws := WebServer{
		config:  config,
		router:  mux.NewRouter(),
		running: false,
		stopErr: make(chan error),
	}
	// Enable logging.
	ws.router.Use(loggingMiddleware)
	// Disable caching.
	ws.router.Use(noCacheMiddleware)
	// Setup not found handler.
	ws.router.NotFoundHandler = noCacheMiddleware(loggingMiddleware(http.NotFoundHandler()))
	// Create http server.
	if config.ServeAddr == "" {
		return nil, nativeerrors.New("no addr provided in config")
	}
	ws.httpServer = &http.Server{
		Handler:      ws.router,
		Addr:         config.ServeAddr,
		WriteTimeout: config.WriteTimeout,
		ReadTimeout:  config.ReadTimeout,
	}
	return &ws, nil
}

// Run starts the web server.
func (server *WebServer) Run(ctx context.Context) error {
	// Check if already running.
	if server.running {
		return nativeerrors.New("web server already running")
	}
	server.running = true
	// Start web server.
	go func() {
		// Enable CORS.
		handler := cors.New(cors.Options{
			AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		}).Handler(server.router)
		logging.WebServerLogger.Infof("web server running at %s", server.config.ServeAddr)
		err := http.ListenAndServe(server.config.ServeAddr, handler)
		if err != nil {
			logging.WebServerLogger.Error(errors.Wrap(err, "listen and serve"))
		}
	}()
	// Wait for stop command.
	_ = <-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	err := server.httpServer.Shutdown(ctx)
	if err != nil {
		return errors.Wrap(err, "shutdown web server")
	}
	return nil
}
