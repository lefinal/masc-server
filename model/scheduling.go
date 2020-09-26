package model

import (
	"github.com/google/uuid"
	"time"
)

// The message types
const (
	MsgTypeGetSchedule MessageType = "get-schedule"
	MsgTypeSchedule    MessageType = "schedule"
)

type EventType string

const (
	EventTypeGame EventType = "game"
)

type Event interface {
	EventId() uuid.UUID
	EventType() EventType
	EventTitle() string
	EventDescription() string
	EventStartTime() time.Time
	EventEndTime() time.Time
}
