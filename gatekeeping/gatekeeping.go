package gatekeeping

import (
	"github.com/LeFinal/masc-server/messages"
)

// Gatekeeper watches over the gate to the wide world. It handles authentication
// as well as blocking forbidden messages.
type Gatekeeper interface {
	// WakeUpAndProtect wakes up the Gatekeeper in order to protect the passed
	// Protected.
	WakeUpAndProtect(protected Protected) error
	// Retire shuts down the Gatekeeper.
	Retire() error
}

// Protected is something that is protected by a Gatekeeper.
type Protected interface {
	// WelcomeDevice is called when a new device is welcomed. When this is called,
	// authentication has already happened.
	WelcomeDevice(device *Device) error
	// SayGoodbyeToDevice is called when a device disconnects. Then it's time to say goodbye.
	SayGoodbyeToDevice(deviceID messages.DeviceID) error
}

// Device holds all information regarding a known device.
type Device struct {
	// ID is the id of the device.
	ID messages.DeviceID
	// Name is the name of the device.
	Name string
	// Roles contains all roles the device says it is able to satisfy.
	Roles []string
	// Send is the channel for outgoing messages.
	Send chan<- messages.MessageContainer
	// Receive is the channel for incoming messages.
	Receive <-chan messages.MessageContainer
}
