package event

import "time"

// NextLogEntriesEventPayloadEntry is used in
// EventNextLogEntriesPayload.Entries.
type NextLogEntriesEventPayloadEntry struct {
	// Time is the timestamp the log entry was created.
	Time time.Time `json:"time"`
	// Message is the log entry message.
	Message string `json:"message"`
	// Level is the log level of the entry.
	Level string `json:"level"`
	// LoggerName is the name of the logger.
	LoggerName string `json:"logger_name"`
	// Fields are the set fields for the log entry.
	Fields map[string]interface{} `json:"fields"`
}

// EventNextLogEntriesPayload is the payload for NextLogEntriesEvent.
type EventNextLogEntriesPayload struct {
	Entries []NextLogEntriesEventPayloadEntry `json:"entries"`
}