package mqttbridge

import (
	"context"
	"github.com/LeFinal/masc-server/client"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"sync"
	"time"
)

const deviceTimeout = 60 * time.Second
const mqttQOS = 0

type mqttDeviceID string

// Config is the config for the Bridge.
type Config struct {
	// MQTTAddr is the address where the MQTT-server is found.
	MQTTAddr string
}

type Bridge interface {
	// Run runs the Bridge. Never call it twice!
	Run(ctx context.Context, deviceRememberer DeviceRememberer, listener client.Listener) error
}

type activeDevice struct {
	bridgeWrapper *deviceWrapper
	lifetime      context.Context
	meow          chan struct{}
}

type mqttMessage struct {
	topic   string
	payload string
}

// netBridge is the implementation of Bridge.
type netBridge struct {
	// config is the configuration to use for connecting.
	config           Config
	deviceRememberer DeviceRememberer
	// clientListener is the handler for when a new Device is discovered.
	clientListener client.Listener
	activeDevices  map[detectedMQTTDevice]*activeDevice
	publish        chan mqttMessage
	m              sync.RWMutex
}

// NewBridge creates a new Bridge. Run it with Bridge.Run.
func NewBridge(config Config) Bridge {
	return &netBridge{
		config:        config,
		activeDevices: make(map[detectedMQTTDevice]*activeDevice),
		publish:       make(chan mqttMessage, mqttBuffer),
	}
}

func (bridge *netBridge) Run(ctx context.Context, deviceRememberer DeviceRememberer, listener client.Listener) error {
	bridge.clientListener = listener
	bridge.deviceRememberer = deviceRememberer
	clientOptions := mqtt.NewClientOptions().AddBroker(bridge.config.MQTTAddr).
		SetClientID("masc-server").
		SetDefaultPublishHandler(bridge.defaultMessageHandler(ctx))
	c := mqtt.NewClient(clientOptions)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		return errors.Error{
			Code:    errors.ErrInternal,
			Err:     token.Error(),
			Message: "connect to mqtt",
			Details: errors.Details{"mqtt_addr": bridge.config.MQTTAddr},
		}
	}
	c.Subscribe("#", 0, nil)
	logging.MQTTLogger.Info("connected to mqtt server",
		zap.String("mqtt_addr", bridge.config.MQTTAddr))
	// Start message forwarding.
	go forwardMQTT(ctx, bridge.publish, c)
	// Wait for ctx finished.
	<-ctx.Done()
	// Wait a maximum of 5 seconds for finishing up.
	c.Disconnect(5000)
	return ctx.Err()
}

// forwardMQTT pumps messages from the given channel to the mqtt.Client.
func forwardMQTT(ctx context.Context, from chan mqttMessage, to mqtt.Client) {
	for {
		select {
		case <-ctx.Done():
			return
		case message := <-from:
			logging.MQTTMessageLogger.Debug(message.payload,
				zap.String("message_topic", message.topic))
			token := to.Publish(message.topic, mqttQOS, false, message.payload)
			token.Wait()
			if err := token.Error(); err != nil {
				errors.Log(logging.MQTTLogger, errors.NewInternalErrorFromErr(token.Error(), "publish mqtt message",
					errors.Details{
						"message_topic":   message.topic,
						"message_payload": message.payload,
					}))
			}
		}
	}
}

// defaultMessageHandler is the default message handler used for the TODO
func (bridge *netBridge) defaultMessageHandler(ctx context.Context) mqtt.MessageHandler {
	return func(_ mqtt.Client, message mqtt.Message) {
		logging.MQTTMessageLogger.Debug(string(message.Payload()),
			zap.String("mqtt_topic", message.Topic()),
			zap.Uint16("mqtt_message_id", message.MessageID()),
			zap.String("direction", "incoming"))
		// Detect device.
		detected, ok := detectDevice(message.Topic())
		if !ok {
			return
		}
		device, err := bridge.getDeviceOrCreateAndMeow(ctx, detected, message)
		if err != nil {
			errors.Log(logging.MQTTLogger, errors.Wrap(err, "create device", errors.Details{
				"mqtt_topic":      message.Topic(),
				"detected_device": detected,
			}))
			return
		}
		// Forward message.
		device.bridgeWrapper.handleMQTTMessage(message)
	}
}

func (bridge *netBridge) getDeviceOrCreateAndMeow(ctx context.Context, detected detectedMQTTDevice, message mqtt.Message) (*activeDevice, error) {
	bridge.m.Lock()
	defer bridge.m.Unlock()
	device, ok := bridge.activeDevices[detected]
	if !ok {
		// Create.
		newDevice, err := createAndBootDevice(ctx, message, detected, bridge.deviceRememberer, bridge, bridge.clientListener)
		if err != nil {
			return nil, errors.Wrap(err, "create and boot device", nil)
		}
		bridge.activeDevices[detected] = newDevice
		device = newDevice
		// Add remove listener.
		go func() {
			select {
			case <-ctx.Done():
			case <-newDevice.lifetime.Done():
				bridge.removeActiveDevice(detected)
			}
		}()
	}
	// Reset timeout.
	select {
	case <-ctx.Done():
	case device.meow <- struct{}{}:
	}
	return device, nil
}

// createAndBootDevice creates a new activeDevice based on the given details. It
// boots the device and start the timeout handlers. The returned context in
// activeDevice marks the lifetime of the device and is done when the timeout
// has occurred.
func createAndBootDevice(ctx context.Context, message mqtt.Message, detected detectedMQTTDevice,
	rememberer DeviceRememberer, publisher mqttPublisher, listener client.Listener) (*activeDevice, error) {
	// Check if device is already known.
	deviceID, err := rememberer.GetDeviceIDByMQTTID(string(detected.mqttID))
	if err != nil {
		// Check if not-found error.
		if e, _ := errors.Cast(err); e.Code != errors.ErrNotFound {
			return nil, errors.Wrap(err, "get device id by mqtt id", errors.Details{"mqtt_id": detected.mqttID})
		}
		deviceID = ""
	}
	// Create new device.
	deviceBridge, err := createDeviceBridge(detected, message.Topic())
	if err != nil {
		return nil, errors.Wrap(err, "create device bridge", nil)
	}
	c := &client.Client{
		ID:      uuid.New().String(),
		Send:    make(chan []byte),
		Receive: make(chan []byte),
	}
	deviceBridgeWrapper := newDeviceBridgeWrapper(c, detected.mqttID, deviceID, rememberer, deviceBridge, publisher)
	deviceCtx, shutdown := context.WithCancel(ctx)
	newActiveDevice := &activeDevice{
		bridgeWrapper: deviceBridgeWrapper,
		lifetime:      deviceCtx,
		meow:          make(chan struct{}),
	}
	// Boot device.
	go func() {
		defer listener.SayGoodbyeToClient(ctx, c)
		statusLogger := logging.MQTTLogger.With(
			zap.Any("mqtt_device_id", detected.mqttID),
			zap.Any("mqtt_device_type", detected.deviceType),
			zap.String("client_id", c.ID))
		statusLogger.Info("device bridge ready")
		err := deviceBridgeWrapper.run(deviceCtx)
		if err != nil && err != context.Canceled {
			errors.Log(logging.MQTTLogger, errors.Wrap(err, "run device bridge wrapper", errors.Details{
				"mqtt_device_id":   detected.mqttID,
				"mqtt_device_type": detected.deviceType,
				"client":           c.ID,
			}))
			return
		}
		if err == context.Canceled {
			statusLogger.Info("device bridge timed out")
		}
	}()
	// Start timeout.
	go runTimeout(ctx, newActiveDevice.meow, shutdown)
	listener.AcceptClient(ctx, c)
	return newActiveDevice, nil
}

func (bridge *netBridge) removeActiveDevice(detected detectedMQTTDevice) {
	bridge.m.Lock()
	defer bridge.m.Unlock()
	delete(bridge.activeDevices, detected)
}

// runTimeout starts a timeout handler with deviceTimeout. When the given
// channel receives, the timeout is being reset.
func runTimeout(ctx context.Context, meow chan struct{}, onTimeout context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-meow:
			// Device got message -> reset timeout.
		case <-time.After(deviceTimeout):
			// Device timed out.
			onTimeout()
			return
		}
	}
}

// publishMQTT forwards the given message to netBridge.publish.
func (bridge *netBridge) publishMQTT(ctx context.Context, topic string, payload string) error {
	select {
	case <-ctx.Done():
		return nil
	case <-time.After(forwardTimeout):
		return errors.NewInternalError("timeout while publishing mqtt message", errors.Details{
			"mqtt_topic": topic,
			"payload":    payload,
		})
	case bridge.publish <- mqttMessage{topic: topic, payload: payload}:
	}
	return nil
}

func (bridge *netBridge) SetClientListener(listener client.Listener) {
	bridge.m.Lock()
	defer bridge.m.Unlock()
	bridge.clientListener = listener
}
