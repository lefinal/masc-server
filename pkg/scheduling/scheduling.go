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
