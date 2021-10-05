package gatekeeping

import (
	"github.com/LeFinal/masc-server/messages"
)

// Gatekeeper watches over the gate to the wide world.
type Gatekeeper interface {
	// WakeUpAndProtect wakes up the Gatekeeper in order to protect the passed
	// Protected.
	WakeUpAndProtect(protected Protected) error
	// Retire shuts down the Gatekeeper.
	Retire() error
	// GetDevices returns a list of all devices, including unknown ones that can be
	// accepted via AcceptDevice.
	GetDevices() []*Device
	// AcceptDevice accepts an unknown device with the given device ID and assigns
	// the passed name to it. Later the Gatekeeper will call Protected.WelcomeDevice
	// for it.
	AcceptDevice(deviceID messages.DeviceID, name string) error
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
	Name string
	// SelfDescription is how the device describes itself. This is used only for
	// human readability and to differentiate between multiple unknown devices.
	SelfDescription string
	// IsAccepted is a flag for whether the device was accepted via
	// Gatekeeper.AcceptDevice or is already known.
	IsAccepted bool
	// IsConnected is a flag for whether the device is currently connected. This
	// allows also managing (removing) devices that are currently not connected.
	IsConnected bool
	// Roles contains all roles the device says it is able to satisfy.
	Roles []string
	// Send is the channel for outgoing messages.
	Send chan<- messages.MessageContainer
	// Receive is the channel for incoming messages.
	Receive <-chan messages.MessageContainer
}
