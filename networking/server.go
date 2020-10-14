package networking

import (
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

func runHubServer(hub *Hub) {
	srv := &http.Server{
		Handler:      createRouter(hub),
		Addr:         hub.addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	serverLogger.Infof("Running hub server on %s", srv.Addr)
	serverLogger.Fatal(srv.ListenAndServe())
}

// createRouter creates the router that is used by the hub server.
func createRouter(hub *Hub) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", http.NotFound)
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	return r
}
