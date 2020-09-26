package model

import "github.com/google/uuid"

type Identifiable interface {
	Identify() uuid.UUID
}
