package messages

import (
	"github.com/gobuffalo/nulls"
	"time"
)

// MessageHello is used with MessageTypeHello for saying hello to MASC.
type MessageHello struct {
	// Roles contains all roles the client is able to satisfy.
	Roles []Role `json:"roles"`
	// SelfDescription is how the device describes itself. Used for
	// human-readability.
	SelfDescription string `json:"self_description"`
}

// MessageWelcome contains additional information regarding the device and is
// used with MessageTypeWelcome.
type MessageWelcome struct {
	// DeviceID is the assigned device ID.
	DeviceID DeviceID `json:"device_id"`
	// Name is the optionally assigned device name.
	Name nulls.String `json:"name"`
}

// MessageDeviceList contains all known devices as well as new unaccepted ones.
type MessageDeviceList struct {
	Devices []Device `json:"devices"`
}

// Device contains all relevant information regarding a device.
type Device struct {
	// ID is the device ID.
	ID DeviceID `json:"id"`
	// Name is the name of the device.
	Name nulls.String `json:"name"`
	// SelfDescription is a human-readable description of how the device describes
	// itself.
	SelfDescription string `json:"self_description"`
	// LastSeen is the last time, the device updated its online-state.
	LastSeen time.Time `json:"last_seen"`
	// IsConnected describes whether the device is currently connected.
	IsConnected bool `json:"is_connected"`
	// Roles contains all roles the device says it can satisfy.
	Roles []Role `json:"roles,omitempty"`
}

// MessageSetDeviceName is used with MessageTypeSetDeviceName.
type MessageSetDeviceName struct {
	// DeviceID is the id of the device to accept.
	DeviceID DeviceID `json:"device_id"`
	// Name is the name that will be assigned to the new device.
	Name string `json:"name"`
}

// MessageDeleteDevice is used with MessageTypeDeleteDevice.
type MessageDeleteDevice struct {
	// DeviceID is the id of the device to delete.
	DeviceID DeviceID `json:"device_id"`
}
