package app

// Config is the configuration needed in order to boot an App.
type Config struct {
	// DBConn is the connection string for the PostgreSQL database.
	DBConn string `json:"db_conn"`
	// WebsocketAddr is the address, the app will listen for connections on.
	WebsocketAddr string `json:"websocket_addr"`
}
