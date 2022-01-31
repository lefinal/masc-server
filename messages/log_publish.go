package messages

import "time"

const (
	// MessageTypeNextLogEntries is used MessageNextLogEntries for publishing new
	// log entries to actors with acting.RoleTypeLogMonitor.
	MessageTypeNextLogEntries MessageType = "next-log-entries"
)

type MessageNextLogEntriesEntry struct {
	// Time is the timestamp the log entry was created.
	Time time.Time `json:"time"`
	// Message is the log entry message.
	Message string `json:"message"`
	// Level is the log level of the entry.
	Level string `json:"level"`
	// Fields are the set fields for the log entry.
	Fields interface{} `json:"fields"`
}

// MessageNextLogEntries is the message content for MessageTypeNextLogEntries.
type MessageNextLogEntries struct {
	Entries []MessageNextLogEntriesEntry `json:"entries"`
}
