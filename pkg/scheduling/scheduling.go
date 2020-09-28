package scheduling

import (
	"github.com/google/uuid"
	"masc-server/pkg/errors"
	"masc-server/pkg/networking"
	"time"
)

// The message types
const (
	MsgTypeGetSchedule   networking.MessageType = "get-schedule"
	MsgTypeSchedule      networking.MessageType = "schedule"
	MsgTypeScheduleEvent networking.MessageType = "schedule-event"
)

type EventType string

const (
	EventTypeMatch EventType = "match"
)

// Common scheduling errors
const (
	ErrEventNotFound errors.ErrorCode = "event.not-found"
)

type EventProvider interface {
	ProvideEvent() Event
}

type Event struct {
	Id          uuid.UUID `json:"id"`
	Type        EventType `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

func (e *Event) Identify() uuid.UUID {
	return e.Id
}

// ScheduleMessage is sent as a response to the GetEventsMessage
type ScheduleMessage struct {
	events []Event
}

// PostEventMessage is sent by the client if an event should be scheduled.
type PostEventMessage struct {
	Event Event `json:"event"`
}

// UpdateEventMessage is sent by the client if an event needs to be updated.
type UpdateEventMessage struct {
	Event Event `json:"event"`
}

// DeleteEventMessage is sent by the client if an event should be deleted.
type DeleteEventMessage struct {
	EventId uuid.UUID `json:"event_id"`
}
