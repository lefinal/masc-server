package devices

import (
	"github.com/LeFinal/masc-server/messages"
	"github.com/google/uuid"
)

type DeviceId uuid.UUID

type Device struct {
	Id          DeviceId
	Name        string
	Description string
	Roles       []Role
	Receive     chan messages.GeneralMessage
}
