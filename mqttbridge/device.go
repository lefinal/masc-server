package mqttbridge

import (
	"github.com/LeFinal/masc-server/messages"
)

// DeviceRememberer is used for remembering a deviceBridge after it got assigned a
// messages.DeviceID.
type DeviceRememberer interface {
	// GetDeviceIDByMQTTID retrieves the messages.DeviceID by the given MQTT-id. If
	// not found, an errors.ErrNotFound will be returned.
	GetDeviceIDByMQTTID(mqttID string) (messages.DeviceID, error)
	// RememberDevice remembers the device with the given mqttDeviceID under the
	// messages.DeviceID. If the device is already known, no error will be returned.
	RememberDevice(mqttID string, deviceID messages.DeviceID) error
}
