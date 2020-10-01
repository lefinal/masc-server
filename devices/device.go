package devices

import "github.com/google/uuid"

type DeviceId uuid.UUID

type Device struct {
	Id          DeviceId
	Name        string
	Description string
	Roles       []Role
}
