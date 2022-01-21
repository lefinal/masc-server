package mqttbridge

import (
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"strings"
)

// detectedMQTTDevice is the mqttDeviceID and DeviceType detected from a topic
// by detectDevice.
type detectedMQTTDevice struct {
	mqttID     mqttDeviceID
	deviceType DeviceType
}

// TODO: Add device detector for each device with all topics or something like that because outgoing messages are also received again!

// detectAndCreateDevice tries to detect a Device from the given topic and
// creates it.
func detectDevice(topic string) (detectedMQTTDevice, bool) {
	topicSegments := strings.Split(topic, "/")
	// Check if shelly1.
	if strings.HasPrefix(topic, "shellies/shelly1-") && !strings.HasSuffix(topic, "/command") {
		return detectedMQTTDevice{
			mqttID:     mqttDeviceID(fmt.Sprintf("%s/%s", topicSegments[0], topicSegments[1])),
			deviceType: DeviceTypeShelly1,
		}, true
	}
	return detectedMQTTDevice{}, false
}

// createDeviceBridge creates a new Device from the given detectedMQTTDevice,
// topic and mqttPublisher.
func createDeviceBridge(device detectedMQTTDevice, topic string) (deviceBridge, error) {
	switch device.deviceType {
	case DeviceTypeShelly1:
		db, err := newNetShelliesShelly1(topic)
		if err != nil {
			return nil, errors.Wrap(err, "new shelly1", errors.Details{"mqtt_topic": topic})
		}
		return db, nil
	}
	return nil, errors.NewInternalError("unknown device type", errors.Details{
		"device_type": device.deviceType,
		"device_id":   device.mqttID,
		"topic":       topic,
	})
}
