package networking

import (
	"github.com/gorilla/mux"
	"net/http"
)

func startHubServer(hub *Hub) {

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
