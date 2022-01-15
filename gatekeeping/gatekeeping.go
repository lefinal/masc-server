package gatekeeping

import (
	"context"
	"github.com/LeFinal/masc-server/client"
	"github.com/LeFinal/masc-server/messages"
	"github.com/gobuffalo/nulls"
	"time"
)

// Gatekeeper watches over the gate to the wide world.
type Gatekeeper interface {
	client.Listener
	// WakeUpAndProtect wakes up the Gatekeeper in order to protect the passed
	// Protected.
	WakeUpAndProtect(protected Protected) error
	// Retire shuts down the Gatekeeper.
	Retire() error
	// GetDevices returns a list of all devices, including unknown ones that can be
	// accepted via AcceptDevice.
	GetDevices() ([]messages.Device, error)
	// SetDeviceName assigns the passed name to it.
	SetDeviceName(deviceID messages.DeviceID, name string) error
	// DeleteDevice deletes the device with the given id.
	DeleteDevice(deviceID messages.DeviceID) error
}

// Protected is something that is protected by a Gatekeeper.
type Protected interface {
	// WelcomeDevice is called when a new device is welcomed. When this is called,
	// authentication has already happened.
	WelcomeDevice(device *Device) error
	// SayGoodbyeToDevice is called when a device disconnects. Then it's time to say
	// goodbye.
	SayGoodbyeToDevice(deviceID messages.DeviceID) error
}

// Device holds all information regarding a known device.
type Device struct {
	// ID is the id of the device.
	ID messages.DeviceID
	// Name is the name of the device.
	Name nulls.String
	// SelfDescription is how the device describes itself. This is used only for
	// human readability and to distinguish between multiple unknown devices.
	SelfDescription string
	// IsConnected is a flag for whether the device is currently connected. This
	// allows also managing (removing) devices that are currently not connected.
	IsConnected bool
	// LastSeen is the last time the online-state for the device was updated.
	LastSeen time.Time
	// Roles contains all roles the device says it is able to satisfy.
	Roles []messages.Role
	// Send is the channel for outgoing messages.
	Send chan<- messages.MessageContainer
	// Receive is the channel for incoming messages.
	Receive <-chan messages.MessageContainer
	// ShutdownPumps shuts down message (un)marshalling pumps for incoming and
	// outgoing messages.
	ShutdownPumps context.CancelFunc
}
