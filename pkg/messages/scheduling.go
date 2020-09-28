package messages

import (
	"github.com/google/uuid"
	"masc-server/pkg/scheduling"
)

// The message types
const (
	MsgTypeGetSchedule   MessageType = "get-schedule"
	MsgTypeSchedule      MessageType = "schedule"
	MsgTypeScheduleEvent MessageType = "schedule-event"
)

// ScheduleMessage is sent as a response to the GetEventsMessage
type ScheduleMessage struct {
	events []scheduling.Event
}

// PostEventMessage is sent by the client if an event should be scheduled.
type PostEventMessage struct {
	Event scheduling.Event `json:"event"`
}

// UpdateEventMessage is sent by the client if an event needs to be updated.
type UpdateEventMessage struct {
	Event scheduling.Event `json:"event"`
}

// DeleteEventMessage is sent by the client if an event should be deleted.
type DeleteEventMessage struct {
	EventId uuid.UUID `json:"event_id"`
}
