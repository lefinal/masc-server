package messages

// MessageHello is used with MessageTypeHello for saying hello to MASC.
type MessageHello struct {
	// Roles contains all roles the client is able to satisfy.
	Roles []string `json:"roles"`
}

// MessageWelcome contains additional information regarding the device and is used with MessageTypeWelcome.
type MessageWelcome struct {
	// Name is the assigned device name.
	Name string `json:"name"`
}

// MessageDeviceList contains all known devices as well as new unaccepted ones.
type MessageDeviceList struct {
	Devices []Device `json:"devices"`
}

// Device contains all relevant information regarding a device.
type Device struct {
	// ID is the device ID.
	ID string `json:"id"`
	// Name is the name of the device.
	Name string `json:"name"`
	// IsConnected describes whether the device is currently connected.
	IsConnected bool `json:"is_connected"`
	// Roles contains all roles the device says it can satisfy.
	Roles []string `json:"roles"`
}

// MessageWelcomeDevice is used for accepting a new device.
type MessageWelcomeDevice struct {
	// Name is the name that will be assigned to the new device.
	Name string `json:"name"`
}
