package scheduling

import (
	"github.com/google/uuid"
	"time"
)

type EventType string

const (
	EventTypeMatch EventType = "match"
)

type EventProvider interface {
	ProvideEvent() Event
}

type Event struct {
	Id          uuid.UUID
	Type        EventType
	Title       string
	Description string
	StartTime   time.Time
	EndTime     time.Time
}

func (e *Event) Identify() uuid.UUID {
	return e.Id
}
