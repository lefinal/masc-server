package devices

import "github.com/google/uuid"

type DeviceId uuid.UUID

type Device struct {
	Id          DeviceId `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Roles       []Role   `json:"roles"`
}
