package scheduling

import (
	"github.com/google/uuid"
	"masc-server/pkg/errors"
	"time"
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
