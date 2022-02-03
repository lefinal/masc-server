package mqttbridge

import (
	"context"
	"encoding/json"
	"github.com/LeFinal/masc-server/client"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

const mqttBuffer = 16

// forwardTimeout is a timeout for forwarding messages.
var forwardTimeout = 10 * time.Second

type DeviceType string

const (
	DeviceTypeShelly1 = "shelly1"
)

type mqttPublisher interface {
	publishMQTT(ctx context.Context, topic string, payload string) error
}

type publisher interface {
	mqttPublisher
	publishMASC(ctx context.Context, messageType messages.MessageType, actorID messages.ActorID, payload interface{}) error
}

type deviceBridge interface {
	genMessageHello() messages.MessageHello
	run(ctx context.Context, deviceID messages.DeviceID, fromMQTT chan mqtt.Message, fromMASC chan messages.MessageContainer, publisher publisher) error
}

type deviceWrapper struct {
	c                *client.Client
	isAuthenticated  bool
	mqttDeviceID     mqttDeviceID
	deviceID         messages.DeviceID
	deviceRememberer DeviceRememberer
	deviceBridge     deviceBridge
	publisher        mqttPublisher
	fromMQTT         chan mqtt.Message
	fromMASC         chan messages.MessageContainer
	m                sync.RWMutex
}

func newDeviceBridgeWrapper(c *client.Client, mqttDeviceID mqttDeviceID, deviceID messages.DeviceID,
	deviceRememberer DeviceRememberer, deviceBridge deviceBridge, publisher mqttPublisher) *deviceWrapper {
	return &deviceWrapper{
		c:                c,
		mqttDeviceID:     mqttDeviceID,
		deviceID:         deviceID,
		deviceRememberer: deviceRememberer,
		deviceBridge:     deviceBridge,
		publisher:        publisher,
		fromMQTT:         make(chan mqtt.Message, mqttBuffer),
		fromMASC:         make(chan messages.MessageContainer, mqttBuffer),
	}
}

func (w *deviceWrapper) run(ctx context.Context) error {
	// First, we perform authentication as this always happens before any other
	// communication is done.
	logging.MQTTLogger.Debug("authenticating", zap.Any("initial_device_id", w.deviceID))
	assignedDeviceID, err := authenticate(ctx, w.deviceID, w.deviceBridge.genMessageHello(), w.c)
	if err != nil {
		return errors.Wrap(err, "authenticate", nil)
	}
	// Check if new device id assigned.
	if w.deviceID != assignedDeviceID {
		// Remember the device.
		logging.MQTTLogger.Debug("got assigned new device id",
			zap.Any("initial_device_id", w.deviceID),
			zap.Any("assigned_device_id", assignedDeviceID))
		if err = w.deviceRememberer.RememberDevice(string(w.mqttDeviceID), assignedDeviceID); err != nil {
			return errors.Wrap(err, "remember device", errors.Details{
				"mqtt_device_id":     w.mqttDeviceID,
				"assigned_device_id": assignedDeviceID,
			})
		}
	}
	w.deviceID = assignedDeviceID
	logging.MQTTLogger.Debug("mqtt device welcomed",
		zap.Any("mqtt_device_id", w.mqttDeviceID),
		zap.Any("device_id", w.deviceID))
	// Run.
	eg, egCtx := errgroup.WithContext(ctx)
	// Start up the device.
	eg.Go(func() error {
		if err := w.deviceBridge.run(egCtx, w.deviceID, w.fromMQTT, w.fromMASC, w); err != nil {
			return errors.Wrap(err, "run device bridge", nil)
		}
		return nil
	})
	// Pump messages from MASC, parse and check for control messages.
	eg.Go(func() error {
		for {
			select {
			case <-egCtx.Done():
				return egCtx.Err()
			case messageRaw, ok := <-w.c.Send:
				if !ok {
					return errors.NewInternalError("are we disconnected?", nil)
				}
				// Parse message container.
				var message messages.MessageContainer
				err := json.Unmarshal(messageRaw, &message)
				if err != nil {
					return errors.NewJSONError(err, "parse message container", false)
				}
				// Forward.
				select {
				case <-egCtx.Done():
					return nil
				case <-time.After(forwardTimeout):
					logging.MQTTLogger.Error("timeout while forwarding message from masc to handler. dropping...",
						zap.Any("device_id", w.deviceID),
						zap.Any("mqtt_device_id", w.mqttDeviceID),
						zap.Any("message", message))
				case w.fromMASC <- message:
				}
			}
		}
	})
	return eg.Wait()
}

func authenticate(ctx context.Context, initialDeviceID messages.DeviceID, helloMessage messages.MessageHello, c *client.Client) (messages.DeviceID, error) {
	// Generate hello message.
	helloContentRaw, err := json.Marshal(helloMessage)
	if err != nil {
		return "", errors.NewJSONError(err, "marshal hello message content", false)
	}
	helloRaw, err := json.Marshal(messages.MessageContainer{
		MessageType: messages.MessageTypeHello,
		DeviceID:    initialDeviceID,
		Content:     helloContentRaw,
	})
	if err != nil {
		return "", errors.NewJSONError(err, "marshal hello message", false)
	}
	// Send hello.
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case c.Receive <- helloRaw:
	}
	// Wait for response.
	var welcome messages.MessageContainer
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case welcomeRaw := <-c.Send:
		// Parse message container.
		err = json.Unmarshal(welcomeRaw, &welcome)
		if err != nil {
			return "", errors.NewJSONError(err, "unmarshal welcome message", false)
		}
	}
	// Parse reply.
	if welcome.MessageType != messages.MessageTypeWelcome {
		return "", errors.NewInternalError("expected welcome message", errors.Details{"was": welcome.MessageType})
	}
	// We are not interested in the actual content because the container already
	// contains all relevant information.
	return welcome.DeviceID, nil
}

func (w *deviceWrapper) handleMQTTMessage(message mqtt.Message) {
	select {
	case w.fromMQTT <- message:
	case <-time.After(forwardTimeout):
		logging.MQTTLogger.Warn("dropping incoming mqtt message due to not being picked up",
			zap.Any("mqtt_device_id", w.mqttDeviceID),
			zap.Any("device_id", w.deviceID),
			zap.Any("message_id", message.MessageID()),
			zap.Any("message_topic", message.Topic()),
			zap.Any("message_payload", message.Payload()))
	}
}

func (w *deviceWrapper) publishMQTT(ctx context.Context, topic string, payload string) error {
	if err := w.publisher.publishMQTT(ctx, topic, payload); err != nil {
		return errors.Wrap(err, "publish mqtt from device wrapper", nil)
	}
	return nil
}

func (w *deviceWrapper) publishMASC(ctx context.Context, messageType messages.MessageType, actorID messages.ActorID, content interface{}) error {
	// Marshal content.
	contentRaw, err := json.Marshal(content)
	if err != nil {
		return errors.NewJSONError(err, "marshal message content", false)
	}
	// Marshal message.
	messageRaw, err := json.Marshal(messages.MessageContainer{
		MessageType: messageType,
		DeviceID:    w.deviceID,
		ActorID:     actorID,
		Content:     contentRaw,
	})
	if err != nil {
		return errors.NewJSONError(err, "marshal message", false)
	}
	// Forward to MASC.
	select {
	case <-ctx.Done():
		return nil
	case <-time.After(forwardTimeout):
		return errors.NewInternalError("timeout while publishing masc message", errors.Details{
			"message_type": messageType,
			"actor_id":     actorID,
			"device_id":    w.deviceID,
			"content":      content,
		})
	case w.c.Receive <- messageRaw:
	}
	return nil
}
