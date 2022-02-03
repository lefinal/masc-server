package app

import (
	"errors"
	"github.com/gobuffalo/nulls"
	"go.uber.org/zap/zapcore"
)

// Config is the configuration needed in order to boot an App.
type Config struct {
	// Log is the configuration to use for logging.
	Log LogConfig `json:"log"`
	// DBConn is the connection string for the PostgreSQL database.
	DBConn string `json:"db_conn"`
	// WebsocketAddr is the address, the app will listen for connections on.
	WebsocketAddr string `json:"websocket_addr"`
	// MQTTAddr is the address where the optional MQTT-server can be found.
	MQTTAddr nulls.String `json:"mqtt_addr,omitempty"`
}

// LogConfig is the configuration to use for logging.
type LogConfig struct {
	// StdoutLogLevel is the log level for optional standard output.
	StdoutLogLevel zapcore.Level `json:"stdout_log_level"`
	// HighPriorityOutput is the optional file where high priority log entries are
	// written to using log rotation.
	HighPriorityOutput nulls.String `json:"high_priority_output"`
	// DebugOutput is the optional file where all log entries (including debug) are
	// written to using log rotation.
	DebugOutput nulls.String `json:"debug_output"`
	// MaxSize is the maximum size in megabytes of log files before they get
	// rotated.
	MaxSize int `json:"max_size"`
	// KeepDays is the amount of days to keep log files.
	KeepDays int `json:"keep_days"`
	// SystemDebugStatsInterval is the interval in time.Minute to optionally log the
	// current system state including stats and current stack.
	SystemDebugStatsInterval nulls.Int `json:"system_debug_stats_interval"`
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
