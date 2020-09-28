package util

import "github.com/google/uuid"

type Identifiable interface {
	Identify() uuid.UUID
}
