package devices

import "github.com/google/uuid"

type Device struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Roles       []Role    `json:"roles"`
}
