package app

import (
	"errors"
	"github.com/gobuffalo/nulls"
)

// Config is the configuration needed in order to boot an App.
type Config struct {
	// DBConn is the connection string for the PostgreSQL database.
	DBConn string `json:"db_conn"`
	// WebsocketAddr is the address, the app will listen for connections on.
	WebsocketAddr string `json:"websocket_addr"`
	// MQTTAddr is the address where the optional MQTT-server can be found.
	MQTTAddr nulls.String `json:"mqtt_addr,omitempty"`
}

// ValidateConfig assures that the given Config is valid.
func ValidateConfig(config Config) error {
	if config.DBConn == "" {
		return errors.New("missing db connection")
	}
	if config.WebsocketAddr == "" {
		return errors.New("missing websocket address")
	}
	return nil
}
