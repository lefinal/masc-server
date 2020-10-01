package devices

import "github.com/google/uuid"

type PerformerId uuid.UUID

type Performer struct {
	Id       PerformerId
	DeviceId DeviceId
	Role     Role
}
