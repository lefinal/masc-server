package shelliesshelly1bridge

// // detectAndCreateDevice tries to detect a Device from the given topic and
// // creates it.
// func detectDevice(topic string) (detectedMQTTDevice, bool) {
// 	topicSegments := strings.Split(topic, "/")
// 	// Check if shelly1.
// 	if strings.HasPrefix(topic, "shellies/shelly1-") && (strings.HasSuffix(topic, "/relay/0") ||
// 		strings.HasSuffix(topic, "/input/0")) {
// 		return detectedMQTTDevice{
// 			mqttID:     portal.mqttDeviceID(fmt.Sprintf("%s/%s", topicSegments[0], topicSegments[1])),
// 			deviceType: DeviceTypeShelly1,
// 		}, true
// 	}
// 	return detectedMQTTDevice{}, false
// }
//
// // createDeviceBridge creates a new Device from the given detectedMQTTDevice,
// // topic and mqttPublisher.
// func createDeviceBridge(device detectedMQTTDevice, topic string) (deviceBridge, error) {
// 	switch device.deviceType {
// 	case DeviceTypeShelly1:
// 		db, err := newNetShelliesShelly1(topic)
// 		if err != nil {
// 			return nil, errors.Wrap(err, "new shelly1", errors.Details{"mqtt_topic": topic})
// 		}
// 		return db, nil
// 	}
// 	return nil, errors.NewInternalError("unknown device type", errors.Details{
// 		"device_type": device.deviceType,
// 		"device_id":   device.mqttID,
// 		"topic":       topic,
// 	})
// }
