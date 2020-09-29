package devices

import "github.com/google/uuid"

type Device struct {
	Id    uuid.UUID `json:"id"`
	Roles []Role    `json:"roles"`
}
